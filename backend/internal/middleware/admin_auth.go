package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	adminauthsvc "wcstransfer/backend/internal/service/adminauth"
)

func AdminAuth(tokens *adminauthsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tokens == nil || !tokens.IsConfigured() {
			c.Next()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if token == "" {
			abortAdminUnauthorized(c)
			return
		}

		claims, err := tokens.ParseToken(token)
		if err == nil {
			c.Set("admin_auth_mode", "session")
			c.Set("admin_claims", claims)
			c.Next()
			return
		}

		abortAdminUnauthorized(c)
	}
}

func abortAdminUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"message": "unauthorized",
			"type":    "auth_error",
		},
	})
}
