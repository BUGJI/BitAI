package middleware

import (
	"net/http"
	"strings"

	bithttp "bitapi/backend/internal/http/respond"
	authsvc "bitapi/backend/internal/services/auth"
	"github.com/gin-gonic/gin"
)

const (
	ContextUserID = "user_id"
	ContextRole   = "role"
)

func Auth(authService *authsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearer(c.GetHeader("Authorization"))
		if token == "" {
			bithttp.Fail(c, http.StatusUnauthorized, "缺少访问令牌")
			c.Abort()
			return
		}
		claims, err := authService.ParseAccessToken(token)
		if err != nil {
			bithttp.Fail(c, http.StatusUnauthorized, "访问令牌无效")
			c.Abort()
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextRole, claims.Role)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[role] = true
	}
	return func(c *gin.Context) {
		role := c.GetString(ContextRole)
		if !allowed[role] {
			bithttp.Fail(c, http.StatusForbidden, "无权访问")
			c.Abort()
			return
		}
		c.Next()
	}
}

func bearer(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return ""
}
