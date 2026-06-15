package handlers

import (
	"errors"
	"net/http"

	"bitapi/backend/internal/http/middleware"
	bithttp "bitapi/backend/internal/http/respond"
	authsvc "bitapi/backend/internal/services/auth"
	verifysvc "bitapi/backend/internal/services/verification"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	auth         *authsvc.Service
	verification *verifysvc.Service
	db           *gorm.DB
}

func NewAuthHandler(auth *authsvc.Service, verification *verifysvc.Service, db *gorm.DB) *AuthHandler {
	return &AuthHandler{auth: auth, verification: verification, db: db}
}

type authRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=6"`
	DisplayName  string `json:"display_name"`
	CaptchaToken string `json:"captcha_token" binding:"required"`
	CaptchaCode  string `json:"captcha_code" binding:"required"`
	EmailToken   string `json:"email_token"`
	EmailCode    string `json:"email_code"`
}

func (h *AuthHandler) Captcha(c *gin.Context) {
	captcha, err := h.verification.CreateCaptcha()
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, captcha)
}

func (h *AuthHandler) SendEmailCode(c *gin.Context) {
	var req struct {
		Email        string `json:"email" binding:"required,email"`
		CaptchaToken string `json:"captcha_token" binding:"required"`
		CaptchaCode  string `json:"captcha_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	token, err := h.verification.SendEmailCode(verifysvc.EmailCodeInput{
		Email:        req.Email,
		CaptchaToken: req.CaptchaToken,
		CaptchaCode:  req.CaptchaCode,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, verifysvc.ErrSMTPNotReady) {
			status = http.StatusServiceUnavailable
		}
		bithttp.Fail(c, status, err.Error())
		return
	}
	bithttp.OK(c, gin.H{"sent": true, "email_token": token})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.EmailToken == "" || req.EmailCode == "" {
		bithttp.Fail(c, http.StatusBadRequest, "邮箱验证码不能为空")
		return
	}
	if err := h.verification.VerifyCaptcha(req.CaptchaToken, req.CaptchaCode); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.verification.VerifyEmailCode(req.Email, req.EmailToken, req.EmailCode); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	pair, err := h.auth.Register(req.Email, req.Password, req.DisplayName)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	bithttp.Created(c, pair)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.verification.VerifyCaptcha(req.CaptchaToken, req.CaptchaCode); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	pair, err := h.auth.Login(req.Email, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, authsvc.ErrInvalidCredentials) {
			status = http.StatusUnauthorized
		}
		bithttp.Fail(c, status, err.Error())
		return
	}
	bithttp.OK(c, pair)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	pair, err := h.auth.Refresh(req.RefreshToken)
	if err != nil {
		bithttp.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	bithttp.OK(c, pair)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetUint(middleware.ContextUserID)
	var user map[string]any
	if err := h.db.Table("users").
		Where("id = ?", userID).
		Select("id", "email", "display_name", "avatar_url", "role", "status", "balance_micros", "concurrency_limit", "rpm_limit", "totp_enabled", "last_login_at", "last_active_at", "created_at").
		Take(&user).Error; err != nil {
		bithttp.Fail(c, http.StatusNotFound, "用户不存在")
		return
	}
	bithttp.OK(c, user)
}
