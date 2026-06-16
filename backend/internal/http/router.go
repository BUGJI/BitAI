package http

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bitapi/backend/internal/config"
	"bitapi/backend/internal/http/handlers"
	"bitapi/backend/internal/http/middleware"
	authsvc "bitapi/backend/internal/services/auth"
	billingsvc "bitapi/backend/internal/services/billing"
	gatewaysvc "bitapi/backend/internal/services/gateway"
	keysvc "bitapi/backend/internal/services/keys"
	monitorsvc "bitapi/backend/internal/services/monitor"
	paymentsvc "bitapi/backend/internal/services/payments"
	verifysvc "bitapi/backend/internal/services/verification"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg config.Config) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"X-BitAPI-Request-ID"},
		AllowCredentials: true,
		AllowWildcard:    true,
		AllowOriginWithContextFunc: func(c *gin.Context, origin string) bool {
			return allowOrigin(c, origin, cfg.CORSOrigins)
		},
		MaxAge: 12 * time.Hour,
	}))

	authService := authsvc.New(db, cfg)
	verificationService := verifysvc.New(db)
	keyService := keysvc.New(db)
	billingService := billingsvc.New(db)
	paymentService := paymentsvc.New(db)
	monitorService := monitorsvc.New(db, cfg)
	gatewayService := gatewaysvc.New(db, cfg, billingService)

	publicHandler := handlers.NewPublicHandler(db, paymentService)
	authHandler := handlers.NewAuthHandler(authService, verificationService, db)
	userHandler := handlers.NewUserHandler(keyService, paymentService, db)
	adminHandler := handlers.NewAdminHandler(db, monitorService, paymentService, cfg.EncryptionKey)
	gatewayHandler := handlers.NewGatewayHandler(gatewayService)

	router.GET("/health", publicHandler.Health)
	router.Static("/uploads", "./data/uploads")
	router.GET("/api/v1/public/settings", publicHandler.Settings)
	router.GET("/api/v1/public/payment-providers", publicHandler.PaymentProviders)
	router.POST("/api/v1/payments/notify/:provider", publicHandler.PaymentNotify)

	api := router.Group("/api/v1")
	api.GET("/auth/captcha", authHandler.Captcha)
	api.POST("/auth/email-code", authHandler.SendEmailCode)
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)

	authed := api.Group("")
	authed.Use(middleware.Auth(authService))
	authed.GET("/auth/me", authHandler.Me)
	authed.POST("/user/avatar", userHandler.UploadAvatar)
	authed.PATCH("/user/profile", userHandler.UpdateProfile)
	authed.GET("/user/api-keys", userHandler.ListKeys)
	authed.POST("/user/api-keys", userHandler.CreateKey)
	authed.DELETE("/user/api-keys/:id", userHandler.DeleteKey)
	authed.GET("/user/usage", userHandler.Usage)
	authed.GET("/user/orders", userHandler.Orders)
	authed.POST("/user/orders", userHandler.CreateOrder)
	authed.POST("/user/redeem", userHandler.Redeem)

	admin := authed.Group("/admin")
	admin.Use(middleware.RequireRole("owner", "admin", "operator"))
	admin.GET("/stats", adminHandler.Stats)
	admin.GET("/users", adminHandler.Users)
	admin.PATCH("/users/:id", adminHandler.UpdateUser)
	admin.POST("/users/:id/recharge", adminHandler.RechargeUser)
	admin.GET("/groups", adminHandler.Groups)
	admin.POST("/groups", adminHandler.CreateGroup)
	admin.PATCH("/groups/:id", adminHandler.UpdateGroup)
	admin.DELETE("/groups/:id", adminHandler.DeleteGroup)
	admin.GET("/upstream-accounts", adminHandler.Accounts)
	admin.POST("/upstream-accounts", adminHandler.CreateAccount)
	admin.PATCH("/upstream-accounts/:id", adminHandler.UpdateAccount)
	admin.DELETE("/upstream-accounts/:id", adminHandler.DeleteAccount)
	admin.POST("/upstream-accounts/:id/check", adminHandler.CheckAccount)
	admin.GET("/group-accounts", adminHandler.GroupAccounts)
	admin.POST("/group-accounts", adminHandler.LinkGroupAccount)
	admin.PATCH("/group-accounts/:id", adminHandler.UpdateGroupAccount)
	admin.DELETE("/group-accounts/:id", adminHandler.DeleteGroupAccount)
	admin.GET("/usage", adminHandler.RecentUsage)
	admin.GET("/settings", adminHandler.Settings)
	admin.POST("/settings", adminHandler.UpsertSetting)
	admin.POST("/branding/upload", adminHandler.UploadBrandingAsset)
	admin.GET("/orders", adminHandler.Orders)
	admin.POST("/orders/:id/mark-paid", adminHandler.MarkOrderPaid)
	admin.POST("/orders/:id/reject", adminHandler.RejectOrder)
	admin.GET("/redeem-codes", adminHandler.RedeemCodes)
	admin.POST("/redeem-codes", adminHandler.CreateRedeemCode)
	admin.POST("/redeem-codes/:id/disable", adminHandler.DisableRedeemCode)
	admin.POST("/redeem-codes/:id/enable", adminHandler.EnableRedeemCode)
	admin.DELETE("/redeem-codes/:id", adminHandler.DeleteRedeemCode)

	router.GET("/v1/models", gatewayHandler.Models)
	router.POST("/v1/chat/completions", gatewayHandler.ChatCompletions)
	router.POST("/v1/responses", gatewayHandler.Responses)
	router.POST("/responses", gatewayHandler.Responses)
	mountFrontend(router)
	return router
}

func mountFrontend(router *gin.Engine) {
	dist := filepath.Clean("../frontend/dist")
	index := filepath.Join(dist, "index.html")
	if _, err := os.Stat(index); err != nil {
		dist = filepath.Clean("./frontend/dist")
		index = filepath.Join(dist, "index.html")
	}
	if _, err := os.Stat(index); err != nil {
		return
	}

	router.Static("/bitapi-assets", filepath.Join(dist, "bitapi-assets"))
	router.Static("/assets", filepath.Join(dist, "assets"))
	router.StaticFile("/favicon.png", filepath.Join(dist, "favicon.png"))
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/v1/") || path == "/responses" || strings.HasPrefix(path, "/uploads/") {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "接口不存在"})
			return
		}
		c.Header("Cache-Control", "no-store")
		c.File(index)
	})
}

func allowOrigin(c *gin.Context, origin string, allowed []string) bool {
	if origin == "" {
		return true
	}
	origin = strings.TrimSpace(strings.ToLower(origin))
	parsedOrigin, err := url.Parse(origin)
	if err != nil || parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
		return false
	}
	originHost := strings.ToLower(parsedOrigin.Hostname())
	for _, item := range allowed {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "*" || item == origin {
			return true
		}
		if strings.HasPrefix(item, "https://*.") && strings.HasPrefix(origin, "https://") {
			domain := strings.TrimPrefix(item, "https://*.")
			if originHost == domain || strings.HasSuffix(originHost, "."+domain) {
				return true
			}
		}
		if strings.HasPrefix(item, "http://*.") && strings.HasPrefix(origin, "http://") {
			domain := strings.TrimPrefix(item, "http://*.")
			if originHost == domain || strings.HasSuffix(originHost, "."+domain) {
				return true
			}
		}
	}
	if parsedOrigin.Scheme == "https" && (originHost == "bitit.cn" || strings.HasSuffix(originHost, ".bitit.cn")) {
		return true
	}
	return sameRequestHost(c.Request.Host, originHost) || sameRequestHost(c.GetHeader("X-Forwarded-Host"), originHost)
}

func sameRequestHost(raw string, originHost string) bool {
	raw = strings.TrimSpace(strings.ToLower(strings.Split(raw, ",")[0]))
	if raw == "" || originHost == "" {
		return false
	}
	parsed, err := url.Parse("//" + raw)
	if err != nil {
		return raw == originHost
	}
	return parsed.Hostname() == originHost
}
