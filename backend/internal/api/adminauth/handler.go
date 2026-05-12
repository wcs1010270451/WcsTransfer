package adminauth

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"wcstransfer/backend/internal/apierror"
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
		apierror.Write(c, apierror.CodeServiceError, "admin auth service unavailable")
		return
	}

	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		apierror.Write(c, apierror.CodeInvalidBody, "invalid request body")
		return
	}

	user, err := h.store.AuthenticateAdminUser(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierror.Write(c, apierror.CodeUnauthorized, "invalid username or password")
			return
		}
		apierror.Write(c, apierror.CodeInternalError, err.Error())
		return
	}

	if err := h.store.UpdateAdminUserLastLogin(c.Request.Context(), user.ID); err != nil {
		apierror.Write(c, apierror.CodeInternalError, err.Error())
		return
	}

	token, err := h.tokens.IssueToken(user.ID, user.Username, user.DisplayName, 12*time.Hour)
	if err != nil {
		apierror.Write(c, apierror.CodeInternalError, "failed to issue admin token")
		return
	}

	c.JSON(200, entity.AdminLoginResult{User: user, Token: token})
}

func (h *Handler) Me(c *gin.Context) {
	if h.store == nil {
		apierror.Write(c, apierror.CodeServiceError, "admin auth service unavailable")
		return
	}

	if claims, ok := adminauthsvc.ClaimsFromContext(c); ok {
		user, err := h.store.GetAdminUserByID(c.Request.Context(), claims.Sub)
		if err != nil {
			apierror.Write(c, apierror.CodeInternalError, err.Error())
			return
		}
		c.JSON(200, gin.H{"user": user, "mode": "session"})
		return
	}

	apierror.Write(c, apierror.CodeUnauthorized, "unauthorized")
}
