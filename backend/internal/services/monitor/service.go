package monitor

import (
	"net/http"
	"strings"
	"time"

	"bitapi/backend/internal/config"
	"bitapi/backend/internal/models"
	bcrypto "bitapi/backend/internal/pkg/crypto"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	cfg    config.Config
	client *http.Client
}

type CheckResult struct {
	AccountID uint   `json:"account_id"`
	OK        bool   `json:"ok"`
	Status    int    `json:"status"`
	Error     string `json:"error,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
}

func New(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg, client: &http.Client{Timeout: 20 * time.Second}}
}

func (s *Service) CheckAccount(id uint) (CheckResult, error) {
	var account models.UpstreamAccount
	if err := s.db.First(&account, id).Error; err != nil {
		return CheckResult{}, err
	}
	credential, err := bcrypto.DecryptString(s.cfg.EncryptionKey, account.CredentialsEnc)
	if err != nil {
		return CheckResult{}, err
	}
	target := strings.TrimRight(account.BaseURL, "/") + "/v1/models"
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return CheckResult{}, err
	}
	if credential != "" {
		req.Header.Set("Authorization", "Bearer "+credential)
	}
	start := time.Now()
	resp, err := s.client.Do(req)
	latency := time.Since(start).Milliseconds()
	result := CheckResult{AccountID: id, LatencyMs: latency}
	if err != nil {
		result.Error = err.Error()
		_ = s.db.Model(&models.UpstreamAccount{}).Where("id = ?", id).Updates(map[string]any{
			"temp_disabled_reason": result.Error,
		}).Error
		return result, nil
	}
	defer resp.Body.Close()
	result.Status = resp.StatusCode
	result.OK = resp.StatusCode >= 200 && resp.StatusCode < 400
	reason := ""
	if !result.OK {
		reason = resp.Status
	}
	if err := s.db.Model(&models.UpstreamAccount{}).Where("id = ?", id).Updates(map[string]any{
		"temp_disabled_reason": reason,
	}).Error; err != nil {
		return result, err
	}
	return result, nil
}
