package handlers

import (
	"net/http"
	"strconv"
	"time"

	"bitapi/backend/internal/http/middleware"
	bithttp "bitapi/backend/internal/http/respond"
	"bitapi/backend/internal/models"
	bcrypto "bitapi/backend/internal/pkg/crypto"
	monitorsvc "bitapi/backend/internal/services/monitor"
	paymentsvc "bitapi/backend/internal/services/payments"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db            *gorm.DB
	monitor       *monitorsvc.Service
	payments      *paymentsvc.Service
	encryptionKey string
}

func NewAdminHandler(db *gorm.DB, monitor *monitorsvc.Service, payments *paymentsvc.Service, encryptionKey string) *AdminHandler {
	return &AdminHandler{db: db, monitor: monitor, payments: payments, encryptionKey: encryptionKey}
}

func (h *AdminHandler) Stats(c *gin.Context) {
	var users, keys, groups, accounts, requests int64
	var charged struct {
		Total int64
	}
	_ = h.db.Model(&models.User{}).Count(&users).Error
	_ = h.db.Model(&models.APIKey{}).Count(&keys).Error
	_ = h.db.Model(&models.Group{}).Count(&groups).Error
	_ = h.db.Model(&models.UpstreamAccount{}).Count(&accounts).Error
	_ = h.db.Model(&models.UsageLog{}).Count(&requests).Error
	_ = h.db.Model(&models.UsageLog{}).Select("coalesce(sum(charged_micros), 0) as total").Scan(&charged).Error
	bithttp.OK(c, gin.H{
		"users":          users,
		"api_keys":       keys,
		"groups":         groups,
		"accounts":       accounts,
		"requests":       requests,
		"charged_micros": charged.Total,
	})
}

func (h *AdminHandler) Users(c *gin.Context) {
	var users []models.User
	if err := h.db.Order("id desc").Limit(200).Find(&users).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, users)
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, "编号无效")
		return
	}
	var req struct {
		Role                 *string `json:"role"`
		Status               *string `json:"status"`
		BalanceMicros        *int64  `json:"balance_micros"`
		TotalRechargedMicros *int64  `json:"total_recharged_micros"`
		ConcurrencyLimit     *int    `json:"concurrency_limit"`
		RPMLimit             *int    `json:"rpm_limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	updates := map[string]any{}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.BalanceMicros != nil {
		updates["balance_micros"] = *req.BalanceMicros
	}
	if req.TotalRechargedMicros != nil {
		updates["total_recharged_micros"] = *req.TotalRechargedMicros
	}
	if req.ConcurrencyLimit != nil {
		updates["concurrency_limit"] = *req.ConcurrencyLimit
	}
	if req.RPMLimit != nil {
		updates["rpm_limit"] = *req.RPMLimit
	}
	if len(updates) == 0 {
		bithttp.Fail(c, http.StatusBadRequest, "没有可更新的内容")
		return
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", uint(id64)).Updates(updates).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	var user models.User
	if err := h.db.First(&user, uint(id64)).Error; err != nil {
		bithttp.Fail(c, http.StatusNotFound, "用户不存在")
		return
	}
	bithttp.OK(c, user)
}

func (h *AdminHandler) RechargeUser(c *gin.Context) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, "编号无效")
		return
	}
	var req struct {
		AmountMicros int64 `json:"amount_micros" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.AmountMicros <= 0 {
		bithttp.Fail(c, http.StatusBadRequest, "金额必须大于 0")
		return
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", uint(id64)).Updates(map[string]any{
		"balance_micros":         gorm.Expr("balance_micros + ?", req.AmountMicros),
		"total_recharged_micros": gorm.Expr("total_recharged_micros + ?", req.AmountMicros),
	}).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	var user models.User
	if err := h.db.First(&user, uint(id64)).Error; err != nil {
		bithttp.Fail(c, http.StatusNotFound, "用户不存在")
		return
	}
	bithttp.OK(c, user)
}

func (h *AdminHandler) Groups(c *gin.Context) {
	var groups []models.Group
	if err := h.db.Order("sort_order asc, id desc").Find(&groups).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, groups)
}

func (h *AdminHandler) CreateGroup(c *gin.Context) {
	var group models.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if group.Platform == "" {
		group.Platform = models.PlatformOpenAI
	}
	if group.Mode == "" {
		group.Mode = models.GroupModeBalance
	}
	if group.Status == "" {
		group.Status = models.StatusActive
	}
	if group.RateMultiplierPPM == 0 {
		group.RateMultiplierPPM = 1000000
	}
	if err := h.db.Create(&group).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.Created(c, group)
}

func (h *AdminHandler) UpdateGroup(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	delete(req, "id")
	delete(req, "created_at")
	delete(req, "updated_at")
	if err := h.db.Model(&models.Group{}).Where("id = ?", id).Updates(req).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		bithttp.Fail(c, http.StatusNotFound, "分组不存在")
		return
	}
	bithttp.OK(c, group)
}

func (h *AdminHandler) DeleteGroup(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.db.Delete(&models.Group{}, id).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, gin.H{"deleted": true})
}

func (h *AdminHandler) Accounts(c *gin.Context) {
	var accounts []models.UpstreamAccount
	if err := h.db.Order("priority asc, id desc").Find(&accounts).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range accounts {
		accounts[i].CredentialsEnc = bcrypto.MaskSecret(accounts[i].CredentialsEnc)
	}
	bithttp.OK(c, accounts)
}

func (h *AdminHandler) CreateAccount(c *gin.Context) {
	var req struct {
		Name              string `json:"name" binding:"required"`
		Platform          string `json:"platform"`
		AuthType          string `json:"auth_type"`
		Credentials       string `json:"credentials"`
		BaseURL           string `json:"base_url"`
		ProxyURL          string `json:"proxy_url"`
		Priority          int    `json:"priority"`
		Weight            int    `json:"weight"`
		ConcurrencyLimit  int    `json:"concurrency_limit"`
		RateMultiplierPPM int64  `json:"rate_multiplier_ppm"`
		Status            string `json:"status"`
		Schedulable       *bool  `json:"schedulable"`
		QuotaJSON         string `json:"quota_json"`
		MetadataJSON      string `json:"metadata_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Platform == "" {
		req.Platform = models.PlatformOpenAI
	}
	if req.AuthType == "" {
		req.AuthType = "api_key"
	}
	if req.BaseURL == "" {
		req.BaseURL = "https://api.openai.com"
	}
	if req.Status == "" {
		req.Status = models.StatusActive
	}
	if req.Weight == 0 {
		req.Weight = 1
	}
	if req.RateMultiplierPPM == 0 {
		req.RateMultiplierPPM = 1000000
	}
	schedulable := true
	if req.Schedulable != nil {
		schedulable = *req.Schedulable
	}
	credentials := req.Credentials
	if credentials != "" {
		encrypted, err := bcrypto.EncryptString(h.encryptionKey, credentials)
		if err != nil {
			bithttp.Fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		credentials = encrypted
	}
	account := models.UpstreamAccount{
		Name:              req.Name,
		Platform:          req.Platform,
		AuthType:          req.AuthType,
		CredentialsEnc:    credentials,
		BaseURL:           req.BaseURL,
		ProxyURL:          req.ProxyURL,
		Priority:          req.Priority,
		Weight:            req.Weight,
		ConcurrencyLimit:  req.ConcurrencyLimit,
		RateMultiplierPPM: req.RateMultiplierPPM,
		Status:            req.Status,
		Schedulable:       schedulable,
		QuotaJSON:         req.QuotaJSON,
		MetadataJSON:      req.MetadataJSON,
	}
	if err := h.db.Select("*").Create(&account).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	account.CredentialsEnc = bcrypto.MaskSecret(account.CredentialsEnc)
	bithttp.Created(c, account)
}

func (h *AdminHandler) UpdateAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	delete(req, "id")
	delete(req, "created_at")
	delete(req, "updated_at")
	if credential, ok := req["credentials"]; ok {
		if value, ok := credential.(string); ok {
			if value == "" || value == "********" {
				delete(req, "credentials")
			} else {
				encrypted, err := bcrypto.EncryptString(h.encryptionKey, value)
				if err != nil {
					bithttp.Fail(c, http.StatusInternalServerError, err.Error())
					return
				}
				req["credentials"] = encrypted
			}
		}
	}
	if err := h.db.Model(&models.UpstreamAccount{}).Where("id = ?", id).Updates(req).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	var account models.UpstreamAccount
	if err := h.db.First(&account, id).Error; err != nil {
		bithttp.Fail(c, http.StatusNotFound, "上游账号不存在")
		return
	}
	account.CredentialsEnc = bcrypto.MaskSecret(account.CredentialsEnc)
	bithttp.OK(c, account)
}

func (h *AdminHandler) DeleteAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.db.Delete(&models.UpstreamAccount{}, id).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, gin.H{"deleted": true})
}

func (h *AdminHandler) CheckAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	result, err := h.monitor.CheckAccount(id)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	bithttp.OK(c, result)
}

func (h *AdminHandler) GroupAccounts(c *gin.Context) {
	var rows []map[string]any
	query := h.db.Table("group_accounts").
		Select("group_accounts.id, group_accounts.group_id, groups.name as group_name, group_accounts.upstream_account_id, upstream_accounts.name as upstream_name, group_accounts.weight, group_accounts.priority, group_accounts.enabled, group_accounts.created_at").
		Joins("left join groups on groups.id = group_accounts.group_id").
		Joins("left join upstream_accounts on upstream_accounts.id = group_accounts.upstream_account_id").
		Order("group_accounts.priority asc, group_accounts.id desc")
	if groupID := c.Query("group_id"); groupID != "" {
		query = query.Where("group_accounts.group_id = ?", groupID)
	}
	if err := query.Find(&rows).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, rows)
}

func (h *AdminHandler) LinkGroupAccount(c *gin.Context) {
	var req struct {
		GroupID           uint `json:"group_id" binding:"required"`
		UpstreamAccountID uint `json:"upstream_account_id" binding:"required"`
		Weight            int  `json:"weight"`
		Priority          int  `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Weight == 0 {
		req.Weight = 1
	}
	link := models.GroupAccount{
		GroupID:           req.GroupID,
		UpstreamAccountID: req.UpstreamAccountID,
		Weight:            req.Weight,
		Priority:          req.Priority,
		Enabled:           true,
	}
	if err := h.db.Create(&link).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.Created(c, link)
}

func (h *AdminHandler) UpdateGroupAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Weight   *int  `json:"weight"`
		Priority *int  `json:"priority"`
		Enabled  *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	updates := map[string]any{}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := h.db.Model(&models.GroupAccount{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, gin.H{"updated": true})
}

func (h *AdminHandler) DeleteGroupAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.db.Delete(&models.GroupAccount{}, id).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, gin.H{"deleted": true})
}

func (h *AdminHandler) RecentUsage(c *gin.Context) {
	limit := 200
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}
	var rows []models.UsageLog
	if err := h.db.Order("id desc").Limit(limit).Find(&rows).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, rows)
}

func (h *AdminHandler) Settings(c *gin.Context) {
	var settings []models.Setting
	if err := h.db.Order("key asc").Find(&settings).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, settings)
}

func (h *AdminHandler) UpsertSetting(c *gin.Context) {
	var req struct {
		Key      string `json:"key" binding:"required"`
		Value    string `json:"value"`
		IsPublic bool   `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var setting models.Setting
	err := h.db.Where("key = ?", req.Key).First(&setting).Error
	if err != nil {
		setting = models.Setting{Key: req.Key, Value: req.Value, IsPublic: req.IsPublic}
		if err := h.db.Create(&setting).Error; err != nil {
			bithttp.Fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		bithttp.Created(c, setting)
		return
	}
	setting.Value = req.Value
	setting.IsPublic = req.IsPublic
	if err := h.db.Save(&setting).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, setting)
}

func (h *AdminHandler) Orders(c *gin.Context) {
	var rows []models.PaymentOrder
	if err := h.db.Order("id desc").Limit(500).Find(&rows).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, rows)
}

func (h *AdminHandler) MarkOrderPaid(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	order, err := h.payments.MarkOrderPaid(id)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	bithttp.OK(c, order)
}

func (h *AdminHandler) RejectOrder(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	order, err := h.payments.RejectOrder(id)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	bithttp.OK(c, order)
}

func (h *AdminHandler) RedeemCodes(c *gin.Context) {
	var rows []models.RedeemCode
	if err := h.db.Order("id desc").Limit(500).Find(&rows).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, rows)
}

func (h *AdminHandler) CreateRedeemCode(c *gin.Context) {
	var req struct {
		AmountMicros int64  `json:"amount_micros" binding:"required"`
		MaxUses      int    `json:"max_uses"`
		ExpiresAt    string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.AmountMicros <= 0 {
		bithttp.Fail(c, http.StatusBadRequest, "金额必须大于 0")
		return
	}
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			bithttp.Fail(c, http.StatusBadRequest, "过期时间必须使用 RFC3339 格式")
			return
		}
		expiresAt = &parsed
	}
	created, err := h.payments.CreateRedeemCode(c.GetUint(middleware.ContextUserID), req.AmountMicros, req.MaxUses, expiresAt)
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.Created(c, created)
}

func (h *AdminHandler) DisableRedeemCode(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.payments.DisableRedeemCode(id)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	bithttp.OK(c, item)
}

func (h *AdminHandler) EnableRedeemCode(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.payments.EnableRedeemCode(id)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	bithttp.OK(c, item)
}

func (h *AdminHandler) DeleteRedeemCode(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.payments.DeleteRedeemCode(id); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	bithttp.OK(c, gin.H{"deleted": true})
}

func parseID(c *gin.Context) (uint, bool) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, "编号无效")
		return 0, false
	}
	return uint(id64), true
}
