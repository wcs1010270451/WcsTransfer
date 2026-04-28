package middleware

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/repository"
)

const clientAPIKeyContextKey = "client_api_key"

func PublicAPIAuth(store repository.ClientAuthStore, logWriter repository.RequestLogWriter) gin.HandlerFunc {
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
		if clientKey.UserID > 0 && clientKey.UserWalletBalance <= 0 {
			writeAuthRejectionLog(c, logWriter, clientKey, 402, "wallet_empty", "wallet balance is empty")
			c.AbortWithStatusJSON(402, gin.H{
				"error": gin.H{
					"message": "wallet balance is empty",
					"type":    "wallet_empty",
				},
			})
			return
		}
		if clientKey.UserID > 0 && clientKey.UserWalletBalance < clientKey.UserMinAvailBalance {
			writeAuthRejectionLog(c, logWriter, clientKey, 402, "wallet_below_minimum", "wallet balance is below the minimum available balance")
			c.AbortWithStatusJSON(402, gin.H{
				"error": gin.H{
					"message": "wallet balance is below the minimum available balance",
					"type":    "wallet_below_minimum",
				},
			})
			return
		}

		c.Set(clientAPIKeyContextKey, clientKey)
		c.Next()
	}
}

func writeAuthRejectionLog(c *gin.Context, logWriter repository.RequestLogWriter, clientKey entity.ClientAPIKey, httpStatus int, errorType string, message string) {
	if logWriter == nil {
		return
	}

	startedAt := time.Now()
	latencyMS := int(time.Since(startedAt).Milliseconds())
	metadata, _ := json.Marshal(map[string]any{
		"user_id":                 clientKey.UserID,
		"user_email":              clientKey.UserEmail,
		"user_wallet_balance":     clientKey.UserWalletBalance,
		"user_min_avail_balance":  clientKey.UserMinAvailBalance,
		"client_api_key_name":     clientKey.Name,
	})
	_, _ = logWriter.CreateRequestLog(c.Request.Context(), entity.CreateRequestLogInput{
		TraceID:         strings.TrimSpace(c.Writer.Header().Get("X-Request-Id")),
		RequestType:     "auth_reject",
		ClientAPIKeyID:  clientKey.ID,
		ClientIP:        c.ClientIP(),
		RequestMethod:   c.Request.Method,
		RequestPath:     c.FullPath(),
		HTTPStatus:      httpStatus,
		Success:         false,
		LatencyMS:       latencyMS,
		ErrorType:       errorType,
		ErrorMessage:    message,
		RequestPayload:  nil,
		ResponsePayload: nil,
		Metadata:        metadata,
	})
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
