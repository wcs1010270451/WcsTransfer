package adminauth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/repository"
	adminauthsvc "wcstransfer/backend/internal/service/adminauth"
)

type Handler struct {
	store  repository.AdminAuthStore
	tokens *adminauthsvc.Service
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func NewHandler(store repository.AdminAuthStore, tokens *adminauthsvc.Service) *Handler {
	return &Handler{store: store, tokens: tokens}
}

func (h *Handler) Login(c *gin.Context) {
	if h.store == nil || h.tokens == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"type":    "service_unavailable",
				"message": "admin auth service unavailable",
			},
		})
		return
	}

	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	user, err := h.store.AuthenticateAdminUser(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"type":    "auth_error",
					"message": "invalid username or password",
				},
			})
			return
		}
		writeDatabaseError(c, err)
		return
	}

	if err := h.store.UpdateAdminUserLastLogin(c.Request.Context(), user.ID); err != nil {
		writeDatabaseError(c, err)
		return
	}

	token, err := h.tokens.IssueToken(user.ID, user.Username, user.DisplayName, 12*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"type":    "token_error",
				"message": "failed to issue admin token",
			},
		})
		return
	}

	c.JSON(http.StatusOK, entity.AdminLoginResult{User: user, Token: token})
}

func (h *Handler) Me(c *gin.Context) {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"type":    "service_unavailable",
				"message": "admin auth service unavailable",
			},
		})
		return
	}

	if claims, ok := adminauthsvc.ClaimsFromContext(c); ok {
		user, err := h.store.GetAdminUserByID(c.Request.Context(), claims.Sub)
		if err != nil {
			writeDatabaseError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": user, "mode": "session"})
		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"type":    "auth_error",
			"message": "unauthorized",
		},
	})
}

func writeBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": gin.H{
			"type":    "invalid_request",
			"message": strings.TrimSpace(message),
		},
	})
}

func writeDatabaseError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"type":    "database_error",
			"message": err.Error(),
		},
	})
}
