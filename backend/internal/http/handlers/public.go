package handlers

import (
	"net/http"

	bithttp "bitapi/backend/internal/http/respond"
	"bitapi/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PublicHandler struct {
	db *gorm.DB
}

func NewPublicHandler(db *gorm.DB) *PublicHandler {
	return &PublicHandler{db: db}
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
	bithttp.OK(c, gin.H{"status": "正常", "service": "BitAPI"})
}
