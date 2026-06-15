package gateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bitapi/backend/internal/config"
	"bitapi/backend/internal/models"
	bcrypto "bitapi/backend/internal/pkg/crypto"
	"bitapi/backend/internal/services/adapters"
	"bitapi/backend/internal/services/billing"
	"bitapi/backend/internal/services/ratelimit"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrUnauthorized  = errors.New("接口密钥无效")
	ErrNoUpstream    = errors.New("没有可用的上游账号")
	ErrKeyInactive   = errors.New("调用密钥未启用")
	ErrQuotaExceeded = errors.New("调用密钥额度已用尽")
	ErrKeyExpired    = errors.New("调用密钥已过期")
	ErrNoBalance     = errors.New("余额不足")
	ErrRateLimited   = errors.New("请求过于频繁，请稍后再试")
)

type Service struct {
	db      *gorm.DB
	cfg     config.Config
	billing *billing.Service
	openai  *adapters.OpenAIAdapter
	limiter *ratelimit.Limiter
}

type Principal struct {
	User   models.User
	APIKey models.APIKey
	Group  *models.Group
}

type ProxyResult struct {
	StatusCode int
	Body       []byte
	Header     http.Header
	RequestID  string
}

type ChatContext struct {
	Principal      Principal
	Group          models.Group
	Account        models.UpstreamAccount
	RequestID      string
	StartedAt      time.Time
	Body           []byte
	RequestedModel string
	RoutedModel    string
	Stream         bool
}

type ResponsesContext struct {
	Principal      Principal
	Group          models.Group
	Account        models.UpstreamAccount
	RequestID      string
	StartedAt      time.Time
	Body           []byte
	RequestedModel string
	RoutedModel    string
	Stream         bool
}

func New(db *gorm.DB, cfg config.Config, billingService *billing.Service) *Service {
	return &Service{db: db, cfg: cfg, billing: billingService, openai: adapters.NewOpenAIAdapter(), limiter: ratelimit.New()}
}

func (s *Service) Authenticate(apiKey string) (Principal, error) {
	hash := bcrypto.SHA256Hex(strings.TrimSpace(apiKey))
	var key models.APIKey
	err := s.db.Preload("User").Preload("Group").Where("key_hash = ?", hash).First(&key).Error
	if err != nil {
		return Principal{}, ErrUnauthorized
	}
	if key.Status != models.StatusActive {
		return Principal{}, ErrKeyInactive
	}
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return Principal{}, ErrKeyExpired
	}
	if key.QuotaLimitMicros > 0 && key.QuotaUsedMicros >= key.QuotaLimitMicros {
		return Principal{}, ErrQuotaExceeded
	}
	if key.User.Status != models.StatusActive {
		return Principal{}, ErrUnauthorized
	}
	principal := Principal{User: key.User, APIKey: key}
	if key.Group != nil {
		principal.Group = key.Group
	}
	return principal, nil
}

func (s *Service) BeginChat(principal Principal, body []byte) (ChatContext, error) {
	group, err := s.resolveGroup(principal)
	if err != nil {
		return ChatContext{}, err
	}
	if group.Mode == models.GroupModeBalance && principal.User.BalanceMicros <= 0 {
		return ChatContext{}, ErrNoBalance
	}
	if principal.APIKey.QuotaLimitMicros > 0 && principal.APIKey.QuotaUsedMicros >= principal.APIKey.QuotaLimitMicros {
		return ChatContext{}, ErrQuotaExceeded
	}
	if principal.User.RPMLimit > 0 && !s.limiter.Allow("user:"+strconv.Itoa(int(principal.User.ID)), principal.User.RPMLimit) {
		return ChatContext{}, ErrRateLimited
	}
	if group.RPMLimit > 0 && !s.limiter.Allow("group:"+strconv.Itoa(int(group.ID)), group.RPMLimit) {
		return ChatContext{}, ErrRateLimited
	}
	account, err := s.selectAccount(group.ID)
	if err != nil {
		return ChatContext{}, err
	}
	meta := s.openai.ExtractRequestMeta(body)
	routedModel := s.mapModel(group.ModelMappingJSON, meta.Model)
	routedBody := body
	if routedModel != "" && routedModel != meta.Model {
		routedBody, err = s.openai.RewriteModel(body, routedModel)
		if err != nil {
			return ChatContext{}, err
		}
	}
	return ChatContext{
		Principal:      principal,
		Group:          group,
		Account:        account,
		RequestID:      uuid.NewString(),
		StartedAt:      time.Now(),
		Body:           routedBody,
		RequestedModel: meta.Model,
		RoutedModel:    routedModel,
		Stream:         meta.Stream,
	}, nil
}

func (s *Service) BeginResponses(principal Principal, body []byte) (ResponsesContext, error) {
	group, err := s.resolveGroup(principal)
	if err != nil {
		return ResponsesContext{}, err
	}
	if group.Mode == models.GroupModeBalance && principal.User.BalanceMicros <= 0 {
		return ResponsesContext{}, ErrNoBalance
	}
	if principal.APIKey.QuotaLimitMicros > 0 && principal.APIKey.QuotaUsedMicros >= principal.APIKey.QuotaLimitMicros {
		return ResponsesContext{}, ErrQuotaExceeded
	}
	if principal.User.RPMLimit > 0 && !s.limiter.Allow("user:"+strconv.Itoa(int(principal.User.ID)), principal.User.RPMLimit) {
		return ResponsesContext{}, ErrRateLimited
	}
	if group.RPMLimit > 0 && !s.limiter.Allow("group:"+strconv.Itoa(int(group.ID)), group.RPMLimit) {
		return ResponsesContext{}, ErrRateLimited
	}
	account, err := s.selectAccount(group.ID)
	if err != nil {
		return ResponsesContext{}, err
	}
	meta := s.openai.ExtractRequestMeta(body)
	routedModel := s.mapModel(group.ModelMappingJSON, meta.Model)
	routedBody := body
	if routedModel != "" && routedModel != meta.Model {
		routedBody, err = s.openai.RewriteModel(body, routedModel)
		if err != nil {
			return ResponsesContext{}, err
		}
	}
	return ResponsesContext{
		Principal:      principal,
		Group:          group,
		Account:        account,
		RequestID:      uuid.NewString(),
		StartedAt:      time.Now(),
		Body:           routedBody,
		RequestedModel: meta.Model,
		RoutedModel:    routedModel,
		Stream:         meta.Stream,
	}, nil
}

func (s *Service) ProxyChatCompletions(principal Principal, body []byte, headers http.Header) (ProxyResult, error) {
	chat, err := s.BeginChat(principal, body)
	if err != nil {
		return ProxyResult{}, err
	}
	return s.ProxyPreparedChat(chat, headers)
}

func (s *Service) ProxyPreparedChat(chat ChatContext, headers http.Header) (ProxyResult, error) {
	credential, credErr := bcrypto.DecryptString(s.cfg.EncryptionKey, chat.Account.CredentialsEnc)
	if credErr != nil {
		return ProxyResult{}, credErr
	}
	resp, respBody, err := s.openai.Proxy(chat.Account.BaseURL, credential, "/v1/chat/completions", chat.Body, headers)
	latency := time.Since(chat.StartedAt).Milliseconds()
	statusCode := http.StatusBadGateway
	errMsg := ""
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if err != nil {
		errMsg = err.Error()
		respBody = []byte(`{"error":{"message":"上游请求失败","type":"gateway_error"}}`)
	}
	respMeta := s.openai.ExtractResponseMeta(respBody)
	modelUsed := respMeta.Model
	if modelUsed == "" {
		modelUsed = chat.RoutedModel
	}
	s.RecordUsage(chat, respMeta, statusCode, latency, errMsg)
	header := http.Header{}
	if resp != nil {
		header = resp.Header
	}
	return ProxyResult{StatusCode: statusCode, Body: respBody, Header: header, RequestID: chat.RequestID}, nil
}

func (s *Service) ProxyChatStream(chat ChatContext, headers http.Header) (*http.Response, error) {
	credential, err := bcrypto.DecryptString(s.cfg.EncryptionKey, chat.Account.CredentialsEnc)
	if err != nil {
		return nil, err
	}
	return s.openai.ProxyStream(chat.Account.BaseURL, credential, "/v1/chat/completions", chat.Body, headers)
}

func (s *Service) ProxyPreparedResponses(ctx ResponsesContext, headers http.Header) (ProxyResult, error) {
	credential, credErr := bcrypto.DecryptString(s.cfg.EncryptionKey, ctx.Account.CredentialsEnc)
	if credErr != nil {
		return ProxyResult{}, credErr
	}
	resp, respBody, err := s.openai.Proxy(ctx.Account.BaseURL, credential, "/v1/responses", ctx.Body, headers)
	latency := time.Since(ctx.StartedAt).Milliseconds()
	statusCode := http.StatusBadGateway
	errMsg := ""
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if err != nil {
		errMsg = err.Error()
		respBody = []byte(`{"error":{"message":"上游请求失败","type":"gateway_error"}}`)
	}
	respMeta := s.openai.ExtractResponseMeta(respBody)
	s.RecordResponsesUsage(ctx, respMeta, statusCode, latency, errMsg)
	header := http.Header{}
	if resp != nil {
		header = resp.Header
	}
	return ProxyResult{StatusCode: statusCode, Body: respBody, Header: header, RequestID: ctx.RequestID}, nil
}

func (s *Service) ProxyResponsesStream(ctx ResponsesContext, headers http.Header) (*http.Response, error) {
	credential, err := bcrypto.DecryptString(s.cfg.EncryptionKey, ctx.Account.CredentialsEnc)
	if err != nil {
		return nil, err
	}
	return s.openai.ProxyStream(ctx.Account.BaseURL, credential, "/v1/responses", ctx.Body, headers)
}

func (s *Service) RecordResponsesUsage(ctx ResponsesContext, respMeta adapters.ResponseMeta, statusCode int, latencyMs int64, errMsg string) {
	modelUsed := respMeta.Model
	if modelUsed == "" {
		modelUsed = ctx.RoutedModel
	}
	_, _ = s.billing.Charge(billing.ChargeInput{
		RequestID:         ctx.RequestID,
		UserID:            ctx.Principal.User.ID,
		APIKeyID:          ctx.Principal.APIKey.ID,
		GroupID:           &ctx.Group.ID,
		UpstreamAccountID: &ctx.Account.ID,
		Platform:          ctx.Group.Platform,
		ModelRequested:    ctx.RequestedModel,
		ModelUsed:         modelUsed,
		PromptTokens:      respMeta.PromptTokens,
		CompletionTokens:  respMeta.CompletionTokens,
		StatusCode:        statusCode,
		LatencyMs:         latencyMs,
		ErrorMessage:      errMsg,
	})
}

func (s *Service) RecordUsage(chat ChatContext, respMeta adapters.ResponseMeta, statusCode int, latencyMs int64, errMsg string) {
	modelUsed := respMeta.Model
	if modelUsed == "" {
		modelUsed = chat.RoutedModel
	}
	_, _ = s.billing.Charge(billing.ChargeInput{
		RequestID:         chat.RequestID,
		UserID:            chat.Principal.User.ID,
		APIKeyID:          chat.Principal.APIKey.ID,
		GroupID:           &chat.Group.ID,
		UpstreamAccountID: &chat.Account.ID,
		Platform:          chat.Group.Platform,
		ModelRequested:    chat.RequestedModel,
		ModelUsed:         modelUsed,
		PromptTokens:      respMeta.PromptTokens,
		CompletionTokens:  respMeta.CompletionTokens,
		StatusCode:        statusCode,
		LatencyMs:         latencyMs,
		ErrorMessage:      errMsg,
	})
}

func (s *Service) Models(principal Principal) ([]string, error) {
	group, err := s.resolveGroup(principal)
	if err != nil {
		return nil, err
	}
	var models []string
	_ = json.Unmarshal([]byte(group.ModelListJSON), &models)
	if len(models) == 0 {
		models = []string{"gpt-4o-mini", "gpt-4.1-mini"}
	}
	return models, nil
}

func (s *Service) resolveGroup(principal Principal) (models.Group, error) {
	if principal.Group != nil {
		return *principal.Group, nil
	}
	var group models.Group
	err := s.db.Where("platform = ? AND status = ?", models.PlatformOpenAI, models.StatusActive).Order("sort_order asc, id asc").First(&group).Error
	if err != nil {
		return models.Group{}, err
	}
	return group, nil
}

func (s *Service) selectAccount(groupID uint) (models.UpstreamAccount, error) {
	now := time.Now()
	var account models.UpstreamAccount
	err := s.db.
		Joins("JOIN group_accounts ON group_accounts.upstream_account_id = upstream_accounts.id").
		Where("group_accounts.group_id = ? AND group_accounts.enabled = ?", groupID, true).
		Where("upstream_accounts.status = ? AND upstream_accounts.schedulable = ?", models.StatusActive, true).
		Where("(upstream_accounts.rate_limited_until IS NULL OR upstream_accounts.rate_limited_until < ?)", now).
		Where("(upstream_accounts.overloaded_until IS NULL OR upstream_accounts.overloaded_until < ?)", now).
		Where("(upstream_accounts.temp_disabled_until IS NULL OR upstream_accounts.temp_disabled_until < ?)", now).
		Order("group_accounts.priority asc, upstream_accounts.priority asc, upstream_accounts.last_used_at asc").
		First(&account).Error
	if err != nil {
		err = s.db.Where("status = ? AND schedulable = ? AND platform = ?", models.StatusActive, true, models.PlatformOpenAI).
			Order("priority asc, last_used_at asc").First(&account).Error
		if err != nil {
			return models.UpstreamAccount{}, ErrNoUpstream
		}
	}
	nowPtr := time.Now()
	_ = s.db.Model(&models.UpstreamAccount{}).Where("id = ?", account.ID).Update("last_used_at", nowPtr).Error
	return account, nil
}

func (s *Service) mapModel(mappingJSON, requested string) string {
	if requested == "" {
		return requested
	}
	var mapping map[string]string
	if err := json.Unmarshal([]byte(mappingJSON), &mapping); err != nil {
		return requested
	}
	if mapped := strings.TrimSpace(mapping[requested]); mapped != "" {
		return mapped
	}
	return requested
}
