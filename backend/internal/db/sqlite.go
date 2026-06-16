package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bitapi/backend/internal/config"
	"bitapi/backend/internal/models"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	if err := ensureDataDir(cfg.DatabaseDSN); err != nil {
		return nil, err
	}
	conn, err := gorm.Open(sqlite.Open(cfg.DatabaseDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := conn.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if err := conn.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return nil, err
	}
	if err := conn.Exec("PRAGMA synchronous=NORMAL").Error; err != nil {
		return nil, err
	}
	if err := conn.Exec("PRAGMA temp_store=MEMORY").Error; err != nil {
		return nil, err
	}
	return conn, nil
}

func AutoMigrate(conn *gorm.DB) error {
	return conn.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.APIKey{},
		&models.Group{},
		&models.UpstreamAccount{},
		&models.GroupAccount{},
		&models.UsageLog{},
		&models.BillingDedup{},
		&models.Setting{},
		&models.VerificationChallenge{},
		&models.PaymentOrder{},
		&models.RedeemCode{},
		&models.RedeemCodeUsage{},
	)
}

func Seed(conn *gorm.DB, cfg config.Config) error {
	var count int64
	if err := conn.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(cfg.BootstrapPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user := models.User{
			Email:         cfg.BootstrapEmail,
			PasswordHash:  string(hash),
			DisplayName:   cfg.BootstrapName,
			Role:          models.RoleOwner,
			Status:        models.StatusActive,
			TokenVersion:  1,
			BalanceMicros: cfg.DefaultUserBalance,
		}
		if err := conn.Create(&user).Error; err != nil {
			return err
		}
	}

	if err := seedSetting(conn, "site.name", cfg.AppName, true); err != nil {
		return err
	}
	appName := strings.TrimSpace(cfg.AppName)
	if appName == "" {
		appName = "BitAPI"
	}
	copyright := fmt.Sprintf("Copyright © 2026 %s. All rights reserved.", appName)
	for _, item := range []struct {
		key      string
		value    string
		isPublic bool
	}{
		{"site.logo_url", "", true},
		{"site.logo_dark_url", "", true},
		{"site.logo_collapsed_url", "", true},
		{"site.copyright", copyright, true},
	} {
		if err := seedSetting(conn, item.key, item.value, item.isPublic); err != nil {
			return err
		}
	}
	if err := seedSetting(conn, "signup.enabled", "true", true); err != nil {
		return err
	}
	if err := seedDefaultModuleSettings(conn, appName); err != nil {
		return err
	}
	if err := seedDefaultPaymentSettings(conn); err != nil {
		return err
	}
	smtpDefaults := []struct {
		key      string
		value    string
		isPublic bool
	}{
		{"smtp.enabled", "false", false},
		{"smtp.host", "", false},
		{"smtp.port", "587", false},
		{"smtp.username", "", false},
		{"smtp.password", "", false},
		{"smtp.from_email", "", false},
		{"smtp.from_name", cfg.AppName, false},
		{"smtp.encryption", "starttls", false},
	}
	for _, item := range smtpDefaults {
		if err := seedSetting(conn, item.key, item.value, item.isPublic); err != nil {
			return err
		}
	}
	return seedDefaultGroup(conn)
}

func seedDefaultModuleSettings(conn *gorm.DB, appName string) error {
	footerJSON, err := buildPortalFooterJSON(appName)
	if err != nil {
		return err
	}
	defaults := []struct {
		key      string
		value    string
		isPublic bool
	}{
		{"module.portal.enabled", "true", true},
		{"module.auth.register.enabled", "true", true},
		{"module.user.dashboard.enabled", "true", true},
		{"module.user.api_keys.enabled", "true", true},
		{"module.user.billing.enabled", "true", true},
		{"module.user.usage.enabled", "true", true},
		{"module.payment.order.enabled", "true", true},
		{"module.payment.redeem.enabled", "true", true},
		{"portal.hero.title", "企业级 AI API 网关", true},
		{"portal.hero.subtitle", "统一接入模型供应商，集中管理密钥、路由、额度、计费与运营。", true},
		{"portal.hero.tag", "AI API Gateway", true},
		{"portal.hero.primary_text", "进入控制台", true},
		{"portal.hero.primary_target", "", true},
		{"portal.hero.secondary_text", "创建账号", true},
		{"portal.hero.secondary_target", "/auth/register", true},
		{"portal.nav", `[{"label":"首页","target":"#top","icon":"home"},{"label":"核心能力","target":"#features","icon":"apps"},{"label":"接入流程","target":"#workflow","icon":"branch"},{"label":"模型能力","target":"#models","icon":"storage"},{"label":"费用中心","target":"/billing","icon":"gift"}]`, true},
		{"portal.metrics", `[{"label":"兼容接口","value":"OpenAI"},{"label":"计费方式","value":"余额扣费"},{"label":"部署形态","value":"私有化"},{"label":"运维视角","value":"可观测"}]`, true},
		{"portal.sections", `[{"id":"features","type":"feature-grid","eyebrow":"核心能力","title":"把模型调用、额度、密钥和账单收进一个入口","description":"面向团队和企业的 AI API Gateway，提供统一入口、模型路由、精细计费和可观测运营能力。","items":[{"icon":"apps","title":"统一模型入口","description":"将多个上游账号聚合成稳定的 OpenAI 兼容接口，业务侧只需要接入一次。"},{"icon":"branch","title":"模型路由策略","description":"按分组、模型、优先级和权重调度上游账号，减少人工切换成本。"},{"icon":"gift","title":"余额计费体系","description":"记录令牌、耗时、扣费和充值流水，支持兑换码与多渠道支付。"},{"icon":"safe","title":"密钥与权限","description":"用户密钥、管理员权限、注册验证和邮箱验证码集中管理。"}]},{"id":"workflow","type":"timeline","eyebrow":"接入流程","title":"从上游账号到业务调用，四步完成交付","description":"适合内部团队、代理平台和私有化部署场景，配置后即可给用户签发调用密钥。","items":[{"icon":"cloud","title":"接入上游账号","description":"录入主账号、代理地址和模型列表，统一维护可用资源池。"},{"icon":"settings","title":"配置分组策略","description":"设置模型映射、倍率、额度限制和调用优先级。"},{"icon":"lock","title":"签发调用密钥","description":"用户在控制台创建密钥，按 OpenAI 兼容方式调用。"},{"icon":"bar-chart","title":"持续运营观测","description":"后台查看日志、消耗、充值订单和上游状态。"}]},{"id":"models","type":"model-grid","eyebrow":"模型能力","title":"兼容主流 OpenAI 风格调用","description":"面向聊天、响应式接口、流式输出和模型列表管理，后续可按业务继续扩展适配层。","items":[{"icon":"message","title":"Chat Completions","description":"支持常见聊天补全接口和流式响应转发。"},{"icon":"code-square","title":"Responses API","description":"提供统一响应入口，方便 Codex 等客户端接入。"},{"icon":"list","title":"Models","description":"按分组返回可用模型，隐藏上游账号复杂度。"},{"icon":"thunderbolt","title":"高可用路由","description":"通过优先级、权重和状态控制提高调用稳定性。"}]},{"id":"contact","type":"cta","eyebrow":"开始使用","title":"让团队用一个地址调用所有模型","description":"进入控制台配置上游账号、创建分组并签发调用密钥。","actions":[{"label":"进入控制台","target":"/dashboard","icon":"right","type":"primary"},{"label":"查看费用中心","target":"/billing","icon":"gift","type":"secondary"}]}]`, true},
		{"portal.footer", footerJSON, true},
		{"portal.features", `[{"title":"统一模型入口","description":"将多个上游账号收敛为稳定的 OpenAI 兼容入口。"},{"title":"精细化运营","description":"支持用户、分组、密钥、额度、充值和兑换码管理。"},{"title":"可观测计费","description":"记录模型、令牌、耗时、扣费和上游调用状态。"}]`, true},
	}
	for _, item := range defaults {
		if err := seedSetting(conn, item.key, item.value, item.isPublic); err != nil {
			return err
		}
	}
	return nil
}

func buildPortalFooterJSON(appName string) (string, error) {
	if strings.TrimSpace(appName) == "" {
		appName = "BitAPI"
	}
	payload := map[string]any{
		"copyright":         fmt.Sprintf("Copyright © 2026 %s. All rights reserved.", appName),
		"qrcodeTitle":       "联系与社群",
		"qrcodeDescription": "可在后台配置二维码图片地址",
		"qrcodeImage":       "",
		"links": []map[string]string{
			{"label": "登录", "target": "/auth/login"},
			{"label": "注册", "target": "/auth/register"},
			{"label": "控制台", "target": "/dashboard"},
		},
		"friendLinks": []map[string]string{
			{"label": "Arco Design", "target": "https://arco.design/vue"},
			{"label": "Gin", "target": "https://gin-gonic.com"},
			{"label": "GORM", "target": "https://gorm.io"},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func seedDefaultPaymentSettings(conn *gorm.DB) error {
	defaults := []struct {
		key      string
		value    string
		isPublic bool
	}{
		{"payment.public_base_url", "http://localhost:8091", false},
		{"payment.return_frontend_url", "http://localhost:8091/billing", false},
		{"payment.usd_cny_rate", "7.20", false},
		{"payment.manual.enabled", "true", true},
		{"payment.epay.enabled", "false", true},
		{"payment.epay.gateway", "https://ezfp.cn", false},
		{"payment.epay.pid", "", false},
		{"payment.epay.key", "", false},
		{"payment.codepay.enabled", "false", true},
		{"payment.codepay.gateway", "https://codepay.fateqq.com/creat_order/", false},
		{"payment.codepay.id", "", false},
		{"payment.codepay.key", "", false},
		{"payment.xunhupay.enabled", "false", true},
		{"payment.xunhupay.gateway", "https://api.xunhupay.com/payment/do.html", false},
		{"payment.xunhupay.appid", "", false},
		{"payment.xunhupay.appsecret", "", false},
		{"payment.alipay_f2f.enabled", "false", true},
		{"payment.alipay_f2f.gateway", "https://openapi.alipay.com/gateway.do", false},
		{"payment.alipay_f2f.app_id", "", false},
		{"payment.alipay_f2f.private_key", "", false},
		{"payment.alipay_f2f.alipay_public_key", "", false},
		{"payment.wechat_native.enabled", "false", true},
		{"payment.wechat_native.gateway", "https://api.mch.weixin.qq.com", false},
		{"payment.wechat_native.appid", "", false},
		{"payment.wechat_native.mchid", "", false},
		{"payment.wechat_native.serial_no", "", false},
		{"payment.wechat_native.private_key", "", false},
		{"payment.wechat_native.api_v3_key", "", false},
	}
	for _, item := range defaults {
		if err := seedSetting(conn, item.key, item.value, item.isPublic); err != nil {
			return err
		}
	}
	return nil
}

func ensureDataDir(dsn string) error {
	if dsn == "" || dsn == ":memory:" {
		return nil
	}
	path := strings.TrimPrefix(dsn, "file:")
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	if path == "" || path == ":memory:" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}
	return nil
}

func seedSetting(conn *gorm.DB, key, value string, isPublic bool) error {
	var setting models.Setting
	err := conn.Where("key = ?", key).First(&setting).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return conn.Create(&models.Setting{Key: key, Value: value, IsPublic: isPublic}).Error
}

func seedDefaultGroup(conn *gorm.DB) error {
	var count int64
	if err := conn.Model(&models.Group{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return conn.Create(&models.Group{
		Name:              "默认兼容分组",
		Description:       "默认的兼容模型接口余额扣费分组。",
		Platform:          models.PlatformOpenAI,
		Mode:              models.GroupModeBalance,
		Status:            models.StatusActive,
		RateMultiplierPPM: 1000000,
		ModelMappingJSON:  "{}",
		ModelListJSON:     `["gpt-4o-mini","gpt-4.1-mini"]`,
		FeaturesJSON:      `{"chat":true,"stream":true}`,
		SortOrder:         1,
	}).Error
}
