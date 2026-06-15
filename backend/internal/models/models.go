package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	RoleOwner    = "owner"
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleUser     = "user"
	RoleAuditor  = "auditor"

	StatusActive    = "active"
	StatusDisabled  = "disabled"
	StatusSuspended = "suspended"

	GroupModeBalance      = "balance"
	GroupModeSubscription = "subscription"

	PlatformOpenAI    = "openai"
	PlatformAnthropic = "anthropic"
	PlatformGemini    = "gemini"
	PlatformBedrock   = "bedrock"
	PlatformCustom    = "custom"

	BillingStatusCharged = "charged"
	BillingStatusSkipped = "skipped"
	BillingStatusFailed  = "failed"

	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusCancelled = "cancelled"
	OrderStatusRejected  = "rejected"

	RedeemStatusActive   = "active"
	RedeemStatusDisabled = "disabled"

	ChallengeTypeCaptcha = "captcha"
	ChallengeTypeEmail   = "email"
)

type Base struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type User struct {
	Base
	Email                string     `json:"email" gorm:"size:255;not null;uniqueIndex"`
	PasswordHash         string     `json:"-" gorm:"size:255;not null"`
	DisplayName          string     `json:"display_name" gorm:"size:120;not null"`
	AvatarURL            string     `json:"avatar_url" gorm:"size:500"`
	Role                 string     `json:"role" gorm:"size:32;index;not null;default:user"`
	Status               string     `json:"status" gorm:"size:32;index;not null;default:active"`
	BalanceMicros        int64      `json:"balance_micros" gorm:"not null;default:0"`
	TotalRechargedMicros int64      `json:"total_recharged_micros" gorm:"not null;default:0"`
	ConcurrencyLimit     int        `json:"concurrency_limit" gorm:"not null;default:0"`
	RPMLimit             int        `json:"rpm_limit" gorm:"not null;default:0"`
	TokenVersion         int        `json:"token_version" gorm:"not null;default:1"`
	TOTPSecretEnc        string     `json:"-" gorm:"type:text"`
	TOTPEnabled          bool       `json:"totp_enabled" gorm:"not null;default:false"`
	LastLoginAt          *time.Time `json:"last_login_at"`
	LastActiveAt         *time.Time `json:"last_active_at"`
}

type RefreshToken struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	UserID       uint       `json:"user_id" gorm:"index;not null"`
	FamilyID     string     `json:"family_id" gorm:"size:64;index;not null"`
	TokenHash    string     `json:"-" gorm:"size:128;uniqueIndex;not null"`
	TokenVersion int        `json:"token_version" gorm:"not null"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"index;not null"`
	RevokedAt    *time.Time `json:"revoked_at"`
	CreatedAt    time.Time  `json:"created_at"`
	User         User       `json:"-" gorm:"constraint:OnDelete:CASCADE"`
}

type APIKey struct {
	Base
	UserID            uint       `json:"user_id" gorm:"index;not null"`
	GroupID           *uint      `json:"group_id" gorm:"index"`
	Name              string     `json:"name" gorm:"size:120;not null"`
	Key               string     `json:"key" gorm:"size:120"`
	KeyHash           string     `json:"-" gorm:"size:128;uniqueIndex;not null"`
	KeyPrefix         string     `json:"key_prefix" gorm:"size:32;index;not null"`
	Status            string     `json:"status" gorm:"size:32;index;not null;default:active"`
	QuotaLimitMicros  int64      `json:"quota_limit_micros" gorm:"not null;default:0"`
	QuotaUsedMicros   int64      `json:"quota_used_micros" gorm:"not null;default:0"`
	RateLimit5hMicros int64      `json:"rate_limit_5h_micros" gorm:"not null;default:0"`
	RateLimit1dMicros int64      `json:"rate_limit_1d_micros" gorm:"not null;default:0"`
	RateLimit7dMicros int64      `json:"rate_limit_7d_micros" gorm:"not null;default:0"`
	Usage5hMicros     int64      `json:"usage_5h_micros" gorm:"not null;default:0"`
	Usage1dMicros     int64      `json:"usage_1d_micros" gorm:"not null;default:0"`
	Usage7dMicros     int64      `json:"usage_7d_micros" gorm:"not null;default:0"`
	Window5hStart     *time.Time `json:"window_5h_start"`
	Window1dStart     *time.Time `json:"window_1d_start"`
	Window7dStart     *time.Time `json:"window_7d_start"`
	ExpiresAt         *time.Time `json:"expires_at" gorm:"index"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	IPWhitelistJSON   string     `json:"ip_whitelist_json" gorm:"type:text"`
	IPBlacklistJSON   string     `json:"ip_blacklist_json" gorm:"type:text"`
	User              User       `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	Group             *Group     `json:"group,omitempty"`
}

type Group struct {
	Base
	Name                string `json:"name" gorm:"size:120;not null"`
	Description         string `json:"description" gorm:"type:text"`
	Platform            string `json:"platform" gorm:"size:32;index;not null;default:openai"`
	Mode                string `json:"mode" gorm:"size:32;index;not null;default:balance"`
	Status              string `json:"status" gorm:"size:32;index;not null;default:active"`
	RateMultiplierPPM   int64  `json:"rate_multiplier_ppm" gorm:"not null;default:1000000"`
	IsExclusive         bool   `json:"is_exclusive" gorm:"not null;default:false"`
	DailyLimitMicros    int64  `json:"daily_limit_micros" gorm:"not null;default:0"`
	WeeklyLimitMicros   int64  `json:"weekly_limit_micros" gorm:"not null;default:0"`
	MonthlyLimitMicros  int64  `json:"monthly_limit_micros" gorm:"not null;default:0"`
	DefaultValidityDays int    `json:"default_validity_days" gorm:"not null;default:0"`
	RPMLimit            int    `json:"rpm_limit" gorm:"not null;default:0"`
	ModelRoutingJSON    string `json:"model_routing_json" gorm:"type:text"`
	ModelMappingJSON    string `json:"model_mapping_json" gorm:"type:text"`
	ModelListJSON       string `json:"model_list_json" gorm:"type:text"`
	FeaturesJSON        string `json:"features_json" gorm:"type:text"`
	FallbackGroupID     *uint  `json:"fallback_group_id" gorm:"index"`
	SortOrder           int    `json:"sort_order" gorm:"index;not null;default:0"`
}

type UpstreamAccount struct {
	Base
	Name               string     `json:"name" gorm:"size:120;not null"`
	Platform           string     `json:"platform" gorm:"size:32;index;not null;default:openai"`
	AuthType           string     `json:"auth_type" gorm:"size:32;not null;default:api_key"`
	CredentialsEnc     string     `json:"credentials" gorm:"type:text"`
	BaseURL            string     `json:"base_url" gorm:"size:500;not null"`
	ProxyURL           string     `json:"proxy_url" gorm:"size:500"`
	Priority           int        `json:"priority" gorm:"index;not null;default:100"`
	Weight             int        `json:"weight" gorm:"not null;default:1"`
	ConcurrencyLimit   int        `json:"concurrency_limit" gorm:"not null;default:0"`
	RateMultiplierPPM  int64      `json:"rate_multiplier_ppm" gorm:"not null;default:1000000"`
	Status             string     `json:"status" gorm:"size:32;index;not null;default:active"`
	Schedulable        bool       `json:"schedulable" gorm:"index;not null"`
	LastUsedAt         *time.Time `json:"last_used_at"`
	RateLimitedUntil   *time.Time `json:"rate_limited_until"`
	OverloadedUntil    *time.Time `json:"overloaded_until"`
	TempDisabledUntil  *time.Time `json:"temp_disabled_until"`
	TempDisabledReason string     `json:"temp_disabled_reason" gorm:"type:text"`
	QuotaJSON          string     `json:"quota_json" gorm:"type:text"`
	MetadataJSON       string     `json:"metadata_json" gorm:"type:text"`
}

type GroupAccount struct {
	ID                uint            `json:"id" gorm:"primaryKey"`
	GroupID           uint            `json:"group_id" gorm:"uniqueIndex:idx_group_account;not null"`
	UpstreamAccountID uint            `json:"upstream_account_id" gorm:"uniqueIndex:idx_group_account;not null"`
	Weight            int             `json:"weight" gorm:"not null;default:1"`
	Priority          int             `json:"priority" gorm:"index;not null;default:100"`
	Enabled           bool            `json:"enabled" gorm:"not null;default:true"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	Group             Group           `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	UpstreamAccount   UpstreamAccount `json:"-" gorm:"constraint:OnDelete:CASCADE"`
}

type UsageLog struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	RequestID         string    `json:"request_id" gorm:"size:64;index;not null"`
	UserID            uint      `json:"user_id" gorm:"index;not null"`
	APIKeyID          uint      `json:"api_key_id" gorm:"index;not null"`
	GroupID           *uint     `json:"group_id" gorm:"index"`
	UpstreamAccountID *uint     `json:"upstream_account_id" gorm:"index"`
	Platform          string    `json:"platform" gorm:"size:32;index;not null"`
	ModelRequested    string    `json:"model_requested" gorm:"size:120;index"`
	ModelUsed         string    `json:"model_used" gorm:"size:120;index"`
	PromptTokens      int       `json:"prompt_tokens" gorm:"not null;default:0"`
	CompletionTokens  int       `json:"completion_tokens" gorm:"not null;default:0"`
	TotalTokens       int       `json:"total_tokens" gorm:"not null;default:0"`
	CostMicros        int64     `json:"cost_micros" gorm:"not null;default:0"`
	ChargedMicros     int64     `json:"charged_micros" gorm:"not null;default:0"`
	StatusCode        int       `json:"status_code" gorm:"not null;default:0"`
	LatencyMs         int64     `json:"latency_ms" gorm:"not null;default:0"`
	ErrorMessage      string    `json:"error_message" gorm:"type:text"`
	BillingStatus     string    `json:"billing_status" gorm:"size:32;index;not null;default:charged"`
	CreatedAt         time.Time `json:"created_at" gorm:"index"`
}

type BillingDedup struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RequestID string    `json:"request_id" gorm:"size:64;uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`
}

type Setting struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"size:120;uniqueIndex;not null"`
	Value     string    `json:"value" gorm:"type:text"`
	IsPublic  bool      `json:"is_public" gorm:"index;not null;default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VerificationChallenge struct {
	Base
	Type      string     `json:"type" gorm:"size:32;index;not null"`
	Token     string     `json:"token" gorm:"size:80;uniqueIndex;not null"`
	Target    string     `json:"target" gorm:"size:255;index"`
	CodeHash  string     `json:"-" gorm:"size:128;not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"index;not null"`
	UsedAt    *time.Time `json:"used_at" gorm:"index"`
}

type PaymentOrder struct {
	Base
	UserID       uint       `json:"user_id" gorm:"index;not null"`
	OrderNo      string     `json:"order_no" gorm:"size:64;uniqueIndex;not null"`
	AmountMicros int64      `json:"amount_micros" gorm:"not null"`
	Status       string     `json:"status" gorm:"size:32;index;not null"`
	Provider     string     `json:"provider" gorm:"size:64;not null"`
	PaidAt       *time.Time `json:"paid_at"`
	MetadataJSON string     `json:"metadata_json" gorm:"type:text"`
	User         User       `json:"-" gorm:"constraint:OnDelete:CASCADE"`
}

type RedeemCode struct {
	Base
	Code         string     `json:"code" gorm:"size:32;index"`
	CodeHash     string     `json:"-" gorm:"size:128;uniqueIndex;not null"`
	CodePrefix   string     `json:"code_prefix" gorm:"size:32;index;not null"`
	AmountMicros int64      `json:"amount_micros" gorm:"not null"`
	Status       string     `json:"status" gorm:"size:32;index;not null"`
	MaxUses      int        `json:"max_uses" gorm:"not null"`
	UsedCount    int        `json:"used_count" gorm:"not null;default:0"`
	ExpiresAt    *time.Time `json:"expires_at" gorm:"index"`
	CreatedByID  uint       `json:"created_by_id" gorm:"index;not null"`
}

type RedeemCodeUsage struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RedeemCodeID uint      `json:"redeem_code_id" gorm:"uniqueIndex:idx_redeem_user;not null"`
	UserID       uint      `json:"user_id" gorm:"uniqueIndex:idx_redeem_user;index;not null"`
	AmountMicros int64     `json:"amount_micros" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
}
