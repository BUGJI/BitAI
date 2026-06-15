package http

import (
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
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"X-BitAPI-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	authService := authsvc.New(db, cfg)
	verificationService := verifysvc.New(db)
	keyService := keysvc.New(db)
	billingService := billingsvc.New(db)
	paymentService := paymentsvc.New(db)
	monitorService := monitorsvc.New(db, cfg)
	gatewayService := gatewaysvc.New(db, cfg, billingService)

	publicHandler := handlers.NewPublicHandler(db)
	authHandler := handlers.NewAuthHandler(authService, verificationService, db)
	userHandler := handlers.NewUserHandler(keyService, paymentService, db)
	adminHandler := handlers.NewAdminHandler(db, monitorService, paymentService, cfg.EncryptionKey)
	gatewayHandler := handlers.NewGatewayHandler(gatewayService)

	router.GET("/health", publicHandler.Health)
	router.GET("/api/v1/public/settings", publicHandler.Settings)

	api := router.Group("/api/v1")
	api.GET("/auth/captcha", authHandler.Captcha)
	api.POST("/auth/email-code", authHandler.SendEmailCode)
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)

	authed := api.Group("")
	authed.Use(middleware.Auth(authService))
	authed.GET("/auth/me", authHandler.Me)
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
	return router
}
