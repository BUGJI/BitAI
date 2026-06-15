package billing

import (
	"errors"
	"time"

	"bitapi/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInsufficientBalance = errors.New("余额不足")

type Service struct {
	db *gorm.DB
}

type ChargeInput struct {
	RequestID         string
	UserID            uint
	APIKeyID          uint
	GroupID           *uint
	UpstreamAccountID *uint
	Platform          string
	ModelRequested    string
	ModelUsed         string
	PromptTokens      int
	CompletionTokens  int
	StatusCode        int
	LatencyMs         int64
	ErrorMessage      string
}

type ChargeResult struct {
	Usage models.UsageLog
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Charge(input ChargeInput) (ChargeResult, error) {
	totalTokens := input.PromptTokens + input.CompletionTokens
	chargedMicros := estimateChargeMicros(totalTokens)
	if input.StatusCode >= 400 {
		chargedMicros = 0
	}

	var usage models.UsageLog
	err := s.db.Transaction(func(tx *gorm.DB) error {
		dedup := models.BillingDedup{RequestID: input.RequestID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&dedup).Error; err != nil {
			return err
		}
		if dedup.ID == 0 {
			return nil
		}

		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, input.UserID).Error; err != nil {
			return err
		}
		if chargedMicros > 0 && user.BalanceMicros < chargedMicros {
			return ErrInsufficientBalance
		}
		if chargedMicros > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", input.UserID).UpdateColumn("balance_micros", gorm.Expr("balance_micros - ?", chargedMicros)).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.APIKey{}).Where("id = ?", input.APIKeyID).Updates(map[string]any{
				"quota_used_micros": gorm.Expr("quota_used_micros + ?", chargedMicros),
				"usage5h_micros":    gorm.Expr("usage5h_micros + ?", chargedMicros),
				"usage1d_micros":    gorm.Expr("usage1d_micros + ?", chargedMicros),
				"usage7d_micros":    gorm.Expr("usage7d_micros + ?", chargedMicros),
				"last_used_at":      time.Now(),
			}).Error; err != nil {
				return err
			}
		}

		usage = models.UsageLog{
			RequestID:         input.RequestID,
			UserID:            input.UserID,
			APIKeyID:          input.APIKeyID,
			GroupID:           input.GroupID,
			UpstreamAccountID: input.UpstreamAccountID,
			Platform:          input.Platform,
			ModelRequested:    input.ModelRequested,
			ModelUsed:         input.ModelUsed,
			PromptTokens:      input.PromptTokens,
			CompletionTokens:  input.CompletionTokens,
			TotalTokens:       totalTokens,
			CostMicros:        chargedMicros,
			ChargedMicros:     chargedMicros,
			StatusCode:        input.StatusCode,
			LatencyMs:         input.LatencyMs,
			ErrorMessage:      input.ErrorMessage,
			BillingStatus:     models.BillingStatusCharged,
			CreatedAt:         time.Now(),
		}
		return tx.Create(&usage).Error
	})
	return ChargeResult{Usage: usage}, err
}

func estimateChargeMicros(totalTokens int) int64 {
	if totalTokens <= 0 {
		return 100
	}
	return int64(totalTokens) * 10
}
