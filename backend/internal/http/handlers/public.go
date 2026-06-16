package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	bithttp "bitapi/backend/internal/http/respond"
	"bitapi/backend/internal/models"
	paymentsvc "bitapi/backend/internal/services/payments"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PublicHandler struct {
	db       *gorm.DB
	payments *paymentsvc.Service
}

func NewPublicHandler(db *gorm.DB, payments *paymentsvc.Service) *PublicHandler {
	return &PublicHandler{db: db, payments: payments}
}

func (h *PublicHandler) Settings(c *gin.Context) {
	var settings []models.Setting
	if err := h.db.Where("is_public = ?", true).Find(&settings).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	values := map[string]string{}
	for _, setting := range settings {
		values[setting.Key] = setting.Value
	}
	bithttp.OK(c, values)
}

func (h *PublicHandler) Health(c *gin.Context) {
	bithttp.OK(c, gin.H{"status": "正常", "service": h.siteName()})
}

func (h *PublicHandler) PaymentProviders(c *gin.Context) {
	options, err := h.payments.PublicProviders()
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, options)
}

func (h *PublicHandler) PaymentNotify(c *gin.Context) {
	provider := c.Param("provider")
	values := map[string]string{}
	if err := c.Request.ParseForm(); err == nil {
		for key, list := range c.Request.Form {
			if len(list) > 0 {
				values[key] = list[0]
			}
		}
	}
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if len(values) == 0 && len(body) > 0 {
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err == nil {
			for key, value := range payload {
				values[key] = stringify(value)
			}
		}
	}
	response, err := h.payments.VerifyAndComplete(provider, values, body, c.Request.Header)
	if err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	c.String(http.StatusOK, response)
}

func stringify(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	case float64, bool:
		return fmt.Sprint(typed)
	default:
		payload, _ := json.Marshal(typed)
		return string(payload)
	}
}

func (h *PublicHandler) siteName() string {
	var setting models.Setting
	if err := h.db.Where("key = ?", "site.name").First(&setting).Error; err == nil {
		if name := strings.TrimSpace(setting.Value); name != "" {
			return name
		}
	}
	return "BitAPI"
}
