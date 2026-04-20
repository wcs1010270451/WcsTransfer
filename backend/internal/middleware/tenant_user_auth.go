package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"wcstransfer/backend/internal/service/tenantauth"
)

const tenantUserClaimsContextKey = "tenant_user_claims"

func TenantUserAuth(auth *tenantauth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"message": "tenant auth is not configured", "type": "auth_error"},
			})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"message": "unauthorized", "type": "auth_error"},
			})
			return
		}

		claims, err := auth.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"message": "unauthorized", "type": "auth_error"},
			})
			return
		}

		c.Set(tenantUserClaimsContextKey, claims)
		c.Next()
	}
}

func TenantUserClaimsFromContext(c *gin.Context) (tenantauth.Claims, bool) {
	value, ok := c.Get(tenantUserClaimsContextKey)
	if !ok {
		return tenantauth.Claims{}, false
	}

	claims, ok := value.(tenantauth.Claims)
	return claims, ok
}
