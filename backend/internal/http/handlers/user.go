package handlers

import (
	"net/http"
	"strconv"
	"time"

	"bitapi/backend/internal/http/middleware"
	bithttp "bitapi/backend/internal/http/respond"
	"bitapi/backend/internal/models"
	keysvc "bitapi/backend/internal/services/keys"
	paymentsvc "bitapi/backend/internal/services/payments"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	keys     *keysvc.Service
	payments *paymentsvc.Service
	db       *gorm.DB
}

func NewUserHandler(keys *keysvc.Service, payments *paymentsvc.Service, db *gorm.DB) *UserHandler {
	return &UserHandler{keys: keys, payments: payments, db: db}
}

func (h *UserHandler) ListKeys(c *gin.Context) {
	keys, err := h.keys.List(c.GetUint(middleware.ContextUserID))
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, keys)
}

func (h *UserHandler) CreateKey(c *gin.Context) {
	var req struct {
		Name             string `json:"name" binding:"required"`
		GroupID          *uint  `json:"group_id"`
		QuotaLimitMicros int64  `json:"quota_limit_micros"`
		ExpiresAt        string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
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
	created, err := h.keys.Create(c.GetUint(middleware.ContextUserID), req.Name, req.GroupID, req.QuotaLimitMicros, expiresAt)
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.Created(c, created)
}

func (h *UserHandler) DeleteKey(c *gin.Context) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, "编号无效")
		return
	}
	if err := h.keys.Delete(c.GetUint(middleware.ContextUserID), uint(id64)); err != nil {
		bithttp.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	bithttp.OK(c, gin.H{"deleted": true})
}

func (h *UserHandler) Usage(c *gin.Context) {
	userID := c.GetUint(middleware.ContextUserID)
	var rows []map[string]any
	err := h.db.Table("usage_logs").Where("user_id = ?", userID).Order("id desc").Limit(100).Find(&rows).Error
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, rows)
}

func (h *UserHandler) Orders(c *gin.Context) {
	var rows []models.PaymentOrder
	if err := h.db.Where("user_id = ?", c.GetUint(middleware.ContextUserID)).Order("id desc").Limit(100).Find(&rows).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, rows)
}

func (h *UserHandler) CreateOrder(c *gin.Context) {
	var req struct {
		AmountMicros int64  `json:"amount_micros" binding:"required"`
		Provider     string `json:"provider"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.AmountMicros <= 0 {
		bithttp.Fail(c, http.StatusBadRequest, "金额必须大于 0")
		return
	}
	order, err := h.payments.CreateOrder(c.GetUint(middleware.ContextUserID), req.AmountMicros, req.Provider)
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.Created(c, order)
}

func (h *UserHandler) Redeem(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	usage, err := h.payments.Redeem(c.GetUint(middleware.ContextUserID), req.Code)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	bithttp.Created(c, usage)
}
