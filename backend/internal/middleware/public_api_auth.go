package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/repository"
)

const clientAPIKeyContextKey = "client_api_key"

func PublicAPIAuth(store repository.ClientAuthStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			c.Next()
			return
		}

		rawKey := extractClientAPIKey(c)
		if strings.TrimSpace(rawKey) == "" {
			c.AbortWithStatusJSON(401, gin.H{
				"error": gin.H{
					"message": "client api key is required",
					"type":    "unauthorized",
				},
			})
			return
		}

		clientKey, err := store.AuthenticateClientAPIKey(c.Request.Context(), rawKey)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{
				"error": gin.H{
					"message": "invalid client api key",
					"type":    "unauthorized",
				},
			})
			return
		}

		c.Set(clientAPIKeyContextKey, clientKey)
		c.Next()
	}
}

func extractClientAPIKey(c *gin.Context) string {
	if value := strings.TrimSpace(c.GetHeader("X-API-Key")); value != "" {
		return value
	}

	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader == "" {
		return ""
	}

	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(strings.ToLower(authHeader), strings.ToLower(bearerPrefix)) {
		return ""
	}

	return strings.TrimSpace(authHeader[len(bearerPrefix):])
}

func ClientAPIKeyFromContext(c *gin.Context) (entity.ClientAPIKey, bool) {
	value, ok := c.Get(clientAPIKeyContextKey)
	if !ok {
		return entity.ClientAPIKey{}, false
	}

	clientKey, valid := value.(entity.ClientAPIKey)
	return clientKey, valid
}
