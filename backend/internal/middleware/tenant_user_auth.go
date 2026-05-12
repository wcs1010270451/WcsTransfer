package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"wcstransfer/backend/internal/apierror"
	"wcstransfer/backend/internal/service/userauth"
)

const userClaimsContextKey = "user_claims"

func TenantUserAuth(auth *userauth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth == nil {
			apierror.Abort(c, apierror.CodeServiceError, "user auth is not configured")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if token == "" {
			apierror.Abort(c, apierror.CodeUnauthorized, "unauthorized")
			return
		}

		claims, err := auth.ParseToken(token)
		if err != nil {
			apierror.Abort(c, apierror.CodeUnauthorized, "unauthorized")
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
