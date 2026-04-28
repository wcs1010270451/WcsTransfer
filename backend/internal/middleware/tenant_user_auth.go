package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"wcstransfer/backend/internal/service/userauth"
)

const userClaimsContextKey = "user_claims"

func TenantUserAuth(auth *userauth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"message": "user auth is not configured", "type": "auth_error"},
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

		c.Set(userClaimsContextKey, claims)
		c.Next()
	}
}

func UserClaimsFromContext(c *gin.Context) (userauth.Claims, bool) {
	value, ok := c.Get(userClaimsContextKey)
	if !ok {
		return userauth.Claims{}, false
	}

	claims, ok := value.(userauth.Claims)
	return claims, ok
}
