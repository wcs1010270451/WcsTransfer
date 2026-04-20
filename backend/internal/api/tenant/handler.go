package tenant

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/middleware"
	"wcstransfer/backend/internal/repository"
	"wcstransfer/backend/internal/service/tenantauth"
)

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

type Handler struct {
	authStore repository.TenantAuthStore
	keyStore  repository.TenantClientKeyStore
	tokens    *tenantauth.Service
}

type registerRequest struct {
	TenantName string `json:"tenant_name" binding:"required"`
	TenantSlug string `json:"tenant_slug"`
	Email      string `json:"email" binding:"required"`
	Password   string `json:"password" binding:"required"`
	FullName   string `json:"full_name" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type createClientKeyRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	ExpiresAt   *string `json:"expires_at"`
}

func NewHandler(authStore repository.TenantAuthStore, keyStore repository.TenantClientKeyStore, tokens *tenantauth.Service) *Handler {
	return &Handler{
		authStore: authStore,
		keyStore:  keyStore,
		tokens:    tokens,
	}
}

func (h *Handler) Register(c *gin.Context) {
	if h.authStore == nil || h.tokens == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "tenant auth is not configured")
		return
	}

	var request registerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if len(strings.TrimSpace(request.Password)) < 8 {
		writeError(c, http.StatusBadRequest, "invalid_request", "password must be at least 8 characters")
		return
	}

	slug := normalizeSlug(request.TenantSlug)
	if slug == "" {
		slug = normalizeSlug(request.TenantName)
	}
	if slug == "" {
		writeError(c, http.StatusBadRequest, "invalid_request", "tenant slug is required")
		return
	}

	user, err := h.authStore.RegisterTenantUser(c.Request.Context(), entity.RegisterTenantUserInput{
		TenantName: strings.TrimSpace(request.TenantName),
		TenantSlug: slug,
		Email:      strings.TrimSpace(request.Email),
		Password:   request.Password,
		FullName:   strings.TrimSpace(request.FullName),
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, "register_failed", err.Error())
		return
	}

	_ = h.authStore.UpdateTenantUserLastLogin(c.Request.Context(), user.ID)
	token, err := h.tokens.IssueToken(user.ID, user.TenantID, user.Email, user.FullName, 7*24*time.Hour)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "token_issue_failed", err.Error())
		return
	}

	c.JSON(http.StatusCreated, entity.TenantLoginResult{User: user, Token: token})
}

func (h *Handler) Login(c *gin.Context) {
	if h.authStore == nil || h.tokens == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "tenant auth is not configured")
		return
	}

	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	user, err := h.authStore.AuthenticateTenantUser(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if err != pgx.ErrNoRows {
			status = http.StatusBadRequest
		}
		writeError(c, status, "auth_error", "invalid email or password")
		return
	}

	_ = h.authStore.UpdateTenantUserLastLogin(c.Request.Context(), user.ID)
	token, err := h.tokens.IssueToken(user.ID, user.TenantID, user.Email, user.FullName, 7*24*time.Hour)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "token_issue_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, entity.TenantLoginResult{User: user, Token: token})
}

func (h *Handler) Me(c *gin.Context) {
	if h.authStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "tenant auth is not configured")
		return
	}

	claims, ok := middleware.TenantUserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	user, err := h.authStore.GetTenantUserByID(c.Request.Context(), claims.Sub)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	tenant, err := h.authStore.GetTenantByID(c.Request.Context(), user.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "tenant": tenant})
}

func (h *Handler) ListClientKeys(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "tenant key store is not configured")
		return
	}

	claims, ok := middleware.TenantUserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	items, err := h.keyStore.ListTenantClientAPIKeys(c.Request.Context(), claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": len(items),
	})
}

func (h *Handler) CreateClientKey(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "tenant key store is not configured")
		return
	}

	claims, ok := middleware.TenantUserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	tenant, err := h.authStore.GetTenantByID(c.Request.Context(), claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}
	if tenant.Status != "active" {
		writeError(c, http.StatusForbidden, "tenant_pending", "workspace is pending activation")
		return
	}

	var request createClientKeyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	var expiresAt *time.Time
	if request.ExpiresAt != nil && strings.TrimSpace(*request.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*request.ExpiresAt))
		if err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "expires_at must be RFC3339")
			return
		}
		expiresAt = &parsed
	}

	item, err := h.keyStore.CreateTenantClientAPIKey(c.Request.Context(), entity.CreateClientAPIKeyInput{
		TenantID:          claims.TenantID,
		CreatedByUserID:   claims.Sub,
		Name:              strings.TrimSpace(request.Name),
		Status:            "active",
		Description:       strings.TrimSpace(request.Description),
		RPMLimit:          0,
		DailyRequestLimit: 0,
		DailyTokenLimit:   0,
		DailyCostLimit:    0,
		MonthlyCostLimit:  0,
		WarningThreshold:  80,
		ExpiresAt:         expiresAt,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, "create_failed", err.Error())
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (h *Handler) Models(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "tenant key store is not configured")
		return
	}

	claims, ok := middleware.TenantUserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	items, err := h.keyStore.ListTenantModels(c.Request.Context(), claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": len(items),
	})
}

func (h *Handler) Stats(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "tenant key store is not configured")
		return
	}

	claims, ok := middleware.TenantUserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	stats, err := h.keyStore.GetTenantPortalStats(c.Request.Context(), claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) Logs(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "tenant key store is not configured")
		return
	}

	claims, ok := middleware.TenantUserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	page, ok := parsePositiveIntQuery(c, "page", 1, 1, 1000000)
	if !ok {
		return
	}
	pageSize, ok := parsePositiveIntQuery(c, "page_size", 20, 1, 100)
	if !ok {
		return
	}
	httpStatus, ok := parseNonNegativeIntQuery(c, "http_status")
	if !ok {
		return
	}
	createdFrom, ok := parseTimeQuery(c, "created_from")
	if !ok {
		return
	}
	createdTo, ok := parseTimeQuery(c, "created_to")
	if !ok {
		return
	}

	var success *bool
	if rawSuccess := strings.TrimSpace(c.Query("success")); rawSuccess != "" {
		parsed, err := strconv.ParseBool(rawSuccess)
		if err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "success must be true or false")
			return
		}
		success = &parsed
	}

	input := entity.ListRequestLogsInput{
		Page:            page,
		PageSize:        pageSize,
		TenantID:        claims.TenantID,
		ModelPublicName: strings.TrimSpace(c.Query("model_public_name")),
		Success:         success,
		HTTPStatus:      httpStatus,
		TraceID:         strings.TrimSpace(c.Query("trace_id")),
		CreatedFrom:     createdFrom,
		CreatedTo:       createdTo,
	}

	items, err := h.keyStore.ListTenantRequestLogs(c.Request.Context(), claims.TenantID, input)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *Handler) LogDetail(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "tenant key store is not configured")
		return
	}

	claims, ok := middleware.TenantUserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	id, err := parseResourceID(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid log id")
		return
	}

	item, err := h.keyStore.GetTenantRequestLog(c.Request.Context(), claims.TenantID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(c, http.StatusNotFound, "not_found", "request log not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) DisableClientKey(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "tenant key store is not configured")
		return
	}

	claims, ok := middleware.TenantUserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	id, err := parseResourceID(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid key id")
		return
	}

	item, err := h.keyStore.DisableTenantClientAPIKey(c.Request.Context(), claims.TenantID, id)
	if err != nil {
		writeError(c, http.StatusBadRequest, "update_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, item)
}

func normalizeSlug(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	trimmed = slugSanitizer.ReplaceAllString(trimmed, "-")
	return strings.Trim(trimmed, "-")
}

func parseResourceID(value string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &id)
	if err != nil || id <= 0 {
		return 0, err
	}
	return id, nil
}

func parsePositiveIntQuery(c *gin.Context, key string, defaultValue int, minValue int, maxValue int) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultValue, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minValue || value > maxValue {
		writeError(c, http.StatusBadRequest, "invalid_request", key+" is invalid")
		return 0, false
	}
	return value, true
}

func parseNonNegativeIntQuery(c *gin.Context, key string) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		writeError(c, http.StatusBadRequest, "invalid_request", key+" is invalid")
		return 0, false
	}
	return value, true
}

func parseTimeQuery(c *gin.Context, key string) (*time.Time, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", key+" must be RFC3339")
		return nil, false
	}
	return &parsed, true
}

func writeError(c *gin.Context, statusCode int, errorType string, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"type":    errorType,
			"message": message,
		},
	})
}
