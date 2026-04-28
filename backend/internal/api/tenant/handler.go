package tenant

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/middleware"
	"wcstransfer/backend/internal/repository"
	"wcstransfer/backend/internal/service/userauth"
)

type Handler struct {
	authStore repository.UserAuthStore
	keyStore  repository.UserClientKeyStore
	tokens    *userauth.Service
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

func NewHandler(authStore repository.UserAuthStore, keyStore repository.UserClientKeyStore, tokens *userauth.Service) *Handler {
	return &Handler{
		authStore: authStore,
		keyStore:  keyStore,
		tokens:    tokens,
	}
}

func (h *Handler) Login(c *gin.Context) {
	if h.authStore == nil || h.tokens == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user auth is not configured")
		return
	}

	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	user, err := h.authStore.AuthenticateUser(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if !errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusBadRequest
		}
		writeError(c, status, "auth_error", "invalid email or password")
		return
	}

	_ = h.authStore.UpdateUserLastLogin(c.Request.Context(), user.ID)
	token, err := h.tokens.IssueToken(user.ID, user.Email, user.FullName, 7*24*time.Hour)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "token_issue_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, entity.UserLoginResult{User: user, Token: token})
}

func (h *Handler) Me(c *gin.Context) {
	if h.authStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user auth is not configured")
		return
	}

	claims, ok := middleware.UserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	user, err := h.authStore.GetUserByID(c.Request.Context(), claims.Sub)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) ListClientKeys(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user key store is not configured")
		return
	}

	claims, ok := middleware.UserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	items, err := h.keyStore.ListUserClientAPIKeys(c.Request.Context(), claims.Sub)
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
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user key store is not configured")
		return
	}

	claims, ok := middleware.UserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
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

	item, err := h.keyStore.CreateUserClientAPIKey(c.Request.Context(), entity.CreateClientAPIKeyInput{
		UserID:            claims.Sub,
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

func (h *Handler) DisableClientKey(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user key store is not configured")
		return
	}

	claims, ok := middleware.UserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	id, err := parseResourceID(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid key id")
		return
	}

	item, err := h.keyStore.DisableUserClientAPIKey(c.Request.Context(), claims.Sub, id)
	if err != nil {
		writeError(c, http.StatusBadRequest, "update_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) Models(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user key store is not configured")
		return
	}

	items, err := h.keyStore.ListModels(c.Request.Context())
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
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user key store is not configured")
		return
	}

	claims, ok := middleware.UserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	stats, err := h.keyStore.GetUserPortalStats(c.Request.Context(), claims.Sub)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) WalletLedger(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user key store is not configured")
		return
	}

	claims, ok := middleware.UserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	page, ok := parsePositiveIntQuery(c, "page", 1, 1, 1000000)
	if !ok {
		return
	}
	pageSize, ok := parsePositiveIntQuery(c, "page_size", 20, 1, 200)
	if !ok {
		return
	}

	items, err := h.keyStore.ListUserWalletLedger(c.Request.Context(), claims.Sub, page, pageSize)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *Handler) ExportBilling(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user key store is not configured")
		return
	}

	claims, ok := middleware.UserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
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

	items, err := h.keyStore.ExportUserRequestLogs(c.Request.Context(), claims.Sub, entity.ListRequestLogsInput{
		ModelPublicName: strings.TrimSpace(c.Query("model_public_name")),
		Success:         success,
		HTTPStatus:      httpStatus,
		TraceID:         strings.TrimSpace(c.Query("trace_id")),
		CreatedFrom:     createdFrom,
		CreatedTo:       createdTo,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	buffer := bytes.NewBuffer(nil)
	writer := csv.NewWriter(buffer)
	_ = writer.Write([]string{
		"id", "trace_id", "client_api_key_name", "provider_name", "provider_key_name",
		"model_public_name", "request_type", "http_status", "success", "latency_ms",
		"prompt_tokens", "completion_tokens", "total_tokens",
		"reserved_amount", "cost_amount", "billable_amount", "gross_profit",
		"error_type", "error_message", "created_at",
	})
	for _, item := range items {
		_ = writer.Write([]string{
			strconv.FormatInt(item.ID, 10),
			item.TraceID,
			item.ClientAPIKeyName,
			item.ProviderName,
			item.ProviderKeyName,
			item.ModelPublicName,
			item.RequestType,
			strconv.Itoa(item.HTTPStatus),
			strconv.FormatBool(item.Success),
			strconv.Itoa(item.LatencyMS),
			strconv.Itoa(item.PromptTokens),
			strconv.Itoa(item.CompletionTokens),
			strconv.Itoa(item.TotalTokens),
			strconv.FormatFloat(item.ReservedAmount, 'f', 8, 64),
			strconv.FormatFloat(item.CostAmount, 'f', 8, 64),
			strconv.FormatFloat(item.BillableAmount, 'f', 8, 64),
			strconv.FormatFloat(item.BillableAmount-item.CostAmount, 'f', 8, 64),
			item.ErrorType,
			item.ErrorMessage,
			item.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		writeError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="portal_billing.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

func (h *Handler) Logs(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user key store is not configured")
		return
	}

	claims, ok := middleware.UserClaimsFromContext(c)
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

	result, err := h.keyStore.ListUserRequestLogs(c.Request.Context(), claims.Sub, entity.ListRequestLogsInput{
		Page:            page,
		PageSize:        pageSize,
		ModelPublicName: strings.TrimSpace(c.Query("model_public_name")),
		Success:         success,
		HTTPStatus:      httpStatus,
		TraceID:         strings.TrimSpace(c.Query("trace_id")),
		CreatedFrom:     createdFrom,
		CreatedTo:       createdTo,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) LogDetail(c *gin.Context) {
	if h.keyStore == nil {
		writeError(c, http.StatusServiceUnavailable, "service_unavailable", "user key store is not configured")
		return
	}

	claims, ok := middleware.UserClaimsFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "auth_error", "unauthorized")
		return
	}

	id, err := parseResourceID(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid log id")
		return
	}

	item, err := h.keyStore.GetUserRequestLog(c.Request.Context(), claims.Sub, id)
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
