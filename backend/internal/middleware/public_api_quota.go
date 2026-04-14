package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"wcstransfer/backend/internal/service/clientquota"
)

func PublicAPIQuota(service *clientquota.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.Next()
			return
		}

		clientKey, ok := ClientAPIKeyFromContext(c)
		if !ok {
			c.Next()
			return
		}

		err := service.ConsumeRequest(c.Request.Context(), clientKey)
		if err == nil {
			c.Next()
			return
		}

		violation, matched := err.(*clientquota.Violation)
		if !matched {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": err.Error(),
					"type":    "quota_check_error",
				},
			})
			return
		}

		c.Header("Retry-After", retryAfterSeconds(violation.ResetAt))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": gin.H{
				"message": violation.Message,
				"type":    violation.Type,
			},
		})
	}
}

func retryAfterSeconds(resetAt time.Time) string {
	if resetAt.IsZero() {
		return "60"
	}

	seconds := int(time.Until(resetAt).Seconds())
	if seconds < 1 {
		seconds = 1
	}

	return strconv.Itoa(seconds)
}
