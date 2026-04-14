package admin

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/repository"
	"wcstransfer/backend/internal/service/keyhealth"
)

type Handler struct {
	store     repository.AdminStore
	keyHealth *keyhealth.Tracker
}

type createProviderRequest struct {
	Name         string          `json:"name" binding:"required"`
	Slug         string          `json:"slug" binding:"required"`
	ProviderType string          `json:"provider_type"`
	BaseURL      string          `json:"base_url" binding:"required"`
	Status       string          `json:"status"`
	Description  string          `json:"description"`
	ExtraConfig  json.RawMessage `json:"extra_config"`
}

type updateProviderRequest struct {
	Name         string          `json:"name" binding:"required"`
	Slug         string          `json:"slug" binding:"required"`
	ProviderType string          `json:"provider_type"`
	BaseURL      string          `json:"base_url" binding:"required"`
	Status       string          `json:"status"`
	Description  string          `json:"description"`
	ExtraConfig  json.RawMessage `json:"extra_config"`
}

type createClientAPIKeyRequest struct {
	Name              string  `json:"name" binding:"required"`
	Status            string  `json:"status"`
	Description       string  `json:"description"`
	RPMLimit          int     `json:"rpm_limit"`
	DailyRequestLimit int     `json:"daily_request_limit"`
	DailyTokenLimit   int     `json:"daily_token_limit"`
	ExpiresAt         *string `json:"expires_at"`
}

type updateClientAPIKeyRequest struct {
	Name              string  `json:"name" binding:"required"`
	Status            string  `json:"status"`
	Description       string  `json:"description"`
	RPMLimit          int     `json:"rpm_limit"`
	DailyRequestLimit int     `json:"daily_request_limit"`
	DailyTokenLimit   int     `json:"daily_token_limit"`
	ExpiresAt         *string `json:"expires_at"`
}

type createProviderKeyRequest struct {
	ProviderID int64  `json:"provider_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	APIKey     string `json:"api_key" binding:"required"`
	Status     string `json:"status"`
	Weight     *int   `json:"weight"`
	Priority   *int   `json:"priority"`
	RPMLimit   int    `json:"rpm_limit"`
	TPMLimit   int64  `json:"tpm_limit"`
}

type updateProviderKeyRequest struct {
	ProviderID int64   `json:"provider_id" binding:"required"`
	Name       string  `json:"name" binding:"required"`
	APIKey     *string `json:"api_key"`
	Status     string  `json:"status"`
	Weight     *int    `json:"weight"`
	Priority   *int    `json:"priority"`
	RPMLimit   int     `json:"rpm_limit"`
	TPMLimit   int64   `json:"tpm_limit"`
}

type createModelRequest struct {
	PublicName     string          `json:"public_name" binding:"required"`
	ProviderID     int64           `json:"provider_id" binding:"required"`
	UpstreamModel  string          `json:"upstream_model" binding:"required"`
	RouteStrategy  string          `json:"route_strategy"`
	IsEnabled      *bool           `json:"is_enabled"`
	MaxTokens      int             `json:"max_tokens"`
	Temperature    float64         `json:"temperature"`
	TimeoutSeconds *int            `json:"timeout_seconds"`
	Metadata       json.RawMessage `json:"metadata"`
}

type updateModelRequest struct {
	PublicName     string          `json:"public_name" binding:"required"`
	ProviderID     int64           `json:"provider_id" binding:"required"`
	UpstreamModel  string          `json:"upstream_model" binding:"required"`
	RouteStrategy  string          `json:"route_strategy"`
	IsEnabled      *bool           `json:"is_enabled"`
	MaxTokens      int             `json:"max_tokens"`
	Temperature    float64         `json:"temperature"`
	TimeoutSeconds *int            `json:"timeout_seconds"`
	Metadata       json.RawMessage `json:"metadata"`
}

func NewHandler(store repository.AdminStore, tracker *keyhealth.Tracker) *Handler {
	return &Handler{store: store, keyHealth: tracker}
}

func (h *Handler) ListProviders(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	items, err := h.store.ListProviders(c.Request.Context())
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": len(items),
	})
}

func (h *Handler) CreateProvider(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	var request createProviderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	input := entity.CreateProviderInput{
		Name:         strings.TrimSpace(request.Name),
		Slug:         strings.TrimSpace(request.Slug),
		ProviderType: defaultString(request.ProviderType, "openai_compatible"),
		BaseURL:      strings.TrimSpace(request.BaseURL),
		Status:       defaultString(request.Status, "active"),
		Description:  strings.TrimSpace(request.Description),
		ExtraConfig:  normalizeJSON(request.ExtraConfig),
	}

	if input.Name == "" || input.Slug == "" || input.BaseURL == "" {
		writeBadRequest(c, "name, slug, and base_url are required")
		return
	}

	item, err := h.store.CreateProvider(c.Request.Context(), input)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (h *Handler) UpdateProvider(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
	if !ok {
		return
	}

	var request updateProviderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	input := entity.UpdateProviderInput{
		ID:           id,
		Name:         strings.TrimSpace(request.Name),
		Slug:         strings.TrimSpace(request.Slug),
		ProviderType: defaultString(request.ProviderType, "openai_compatible"),
		BaseURL:      strings.TrimSpace(request.BaseURL),
		Status:       defaultString(request.Status, "active"),
		Description:  strings.TrimSpace(request.Description),
		ExtraConfig:  normalizeJSON(request.ExtraConfig),
	}

	if input.Name == "" || input.Slug == "" || input.BaseURL == "" {
		writeBadRequest(c, "name, slug, and base_url are required")
		return
	}

	item, err := h.store.UpdateProvider(c.Request.Context(), input)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) ListClientAPIKeys(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	items, err := h.store.ListClientAPIKeys(c.Request.Context())
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": len(items),
	})
}

func (h *Handler) CreateClientAPIKey(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	var request createClientAPIKeyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	expiresAt, ok := parseOptionalTime(request.ExpiresAt)
	if !ok {
		writeBadRequest(c, "expires_at must be an RFC3339 datetime")
		return
	}

	input := entity.CreateClientAPIKeyInput{
		Name:              strings.TrimSpace(request.Name),
		Status:            defaultString(request.Status, "active"),
		Description:       strings.TrimSpace(request.Description),
		RPMLimit:          request.RPMLimit,
		DailyRequestLimit: request.DailyRequestLimit,
		DailyTokenLimit:   request.DailyTokenLimit,
		ExpiresAt:         expiresAt,
	}
	if input.Name == "" {
		writeBadRequest(c, "name is required")
		return
	}

	item, err := h.store.CreateClientAPIKey(c.Request.Context(), input)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (h *Handler) UpdateClientAPIKey(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
	if !ok {
		return
	}

	var request updateClientAPIKeyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	expiresAt, valid := parseOptionalTime(request.ExpiresAt)
	if !valid {
		writeBadRequest(c, "expires_at must be an RFC3339 datetime")
		return
	}

	input := entity.UpdateClientAPIKeyInput{
		ID:                id,
		Name:              strings.TrimSpace(request.Name),
		Status:            defaultString(request.Status, "active"),
		Description:       strings.TrimSpace(request.Description),
		RPMLimit:          request.RPMLimit,
		DailyRequestLimit: request.DailyRequestLimit,
		DailyTokenLimit:   request.DailyTokenLimit,
		ExpiresAt:         expiresAt,
	}
	if input.Name == "" {
		writeBadRequest(c, "name is required")
		return
	}

	item, err := h.store.UpdateClientAPIKey(c.Request.Context(), input)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) ListProviderKeys(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	items, err := h.store.ListProviderKeys(c.Request.Context())
	if err != nil {
		writeDatabaseError(c, err)
		return
	}
	if h.keyHealth != nil {
		items = h.keyHealth.EnrichKeys(items)
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": len(items),
	})
}

func (h *Handler) CreateProviderKey(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	var request createProviderKeyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	input := entity.CreateProviderKeyInput{
		ProviderID: request.ProviderID,
		Name:       strings.TrimSpace(request.Name),
		APIKey:     strings.TrimSpace(request.APIKey),
		Status:     defaultString(request.Status, "active"),
		Weight:     defaultOptionalInt(request.Weight, 100),
		Priority:   defaultOptionalInt(request.Priority, 100),
		RPMLimit:   request.RPMLimit,
		TPMLimit:   request.TPMLimit,
	}

	if input.ProviderID <= 0 || input.Name == "" || input.APIKey == "" {
		writeBadRequest(c, "provider_id, name, and api_key are required")
		return
	}

	item, err := h.store.CreateProviderKey(c.Request.Context(), input)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (h *Handler) UpdateProviderKey(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
	if !ok {
		return
	}

	var request updateProviderKeyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	var apiKey *string
	if request.APIKey != nil {
		trimmed := strings.TrimSpace(*request.APIKey)
		apiKey = &trimmed
	}

	input := entity.UpdateProviderKeyInput{
		ID:         id,
		ProviderID: request.ProviderID,
		Name:       strings.TrimSpace(request.Name),
		APIKey:     apiKey,
		Status:     defaultString(request.Status, "active"),
		Weight:     defaultOptionalInt(request.Weight, 100),
		Priority:   defaultOptionalInt(request.Priority, 100),
		RPMLimit:   request.RPMLimit,
		TPMLimit:   request.TPMLimit,
	}

	if input.ProviderID <= 0 || input.Name == "" {
		writeBadRequest(c, "provider_id and name are required")
		return
	}

	item, err := h.store.UpdateProviderKey(c.Request.Context(), input)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) ListModels(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	items, err := h.store.ListModels(c.Request.Context())
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": len(items),
	})
}

func (h *Handler) CreateModel(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	var request createModelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	input := entity.CreateModelInput{
		PublicName:     strings.TrimSpace(request.PublicName),
		ProviderID:     request.ProviderID,
		UpstreamModel:  strings.TrimSpace(request.UpstreamModel),
		RouteStrategy:  defaultString(request.RouteStrategy, "fixed"),
		IsEnabled:      defaultBool(request.IsEnabled, true),
		MaxTokens:      request.MaxTokens,
		Temperature:    request.Temperature,
		TimeoutSeconds: defaultOptionalInt(request.TimeoutSeconds, 120),
		Metadata:       normalizeJSON(request.Metadata),
	}

	if input.PublicName == "" || input.ProviderID <= 0 || input.UpstreamModel == "" {
		writeBadRequest(c, "public_name, provider_id, and upstream_model are required")
		return
	}

	item, err := h.store.CreateModel(c.Request.Context(), input)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (h *Handler) UpdateModel(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
	if !ok {
		return
	}

	var request updateModelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	input := entity.UpdateModelInput{
		ID:             id,
		PublicName:     strings.TrimSpace(request.PublicName),
		ProviderID:     request.ProviderID,
		UpstreamModel:  strings.TrimSpace(request.UpstreamModel),
		RouteStrategy:  defaultString(request.RouteStrategy, "fixed"),
		IsEnabled:      defaultBool(request.IsEnabled, true),
		MaxTokens:      request.MaxTokens,
		Temperature:    request.Temperature,
		TimeoutSeconds: defaultOptionalInt(request.TimeoutSeconds, 120),
		Metadata:       normalizeJSON(request.Metadata),
	}

	if input.PublicName == "" || input.ProviderID <= 0 || input.UpstreamModel == "" {
		writeBadRequest(c, "public_name, provider_id, and upstream_model are required")
		return
	}

	item, err := h.store.UpdateModel(c.Request.Context(), input)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) ListLogs(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
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
	providerID, ok := parseNonNegativeInt64Query(c, "provider_id")
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
			writeBadRequest(c, "success must be true or false")
			return
		}
		success = &parsed
	}

	items, err := h.store.ListRequestLogs(c.Request.Context(), entity.ListRequestLogsInput{
		Page:            page,
		PageSize:        pageSize,
		ProviderID:      providerID,
		ModelPublicName: strings.TrimSpace(c.Query("model_public_name")),
		Success:         success,
		HTTPStatus:      httpStatus,
		TraceID:         strings.TrimSpace(c.Query("trace_id")),
		CreatedFrom:     createdFrom,
		CreatedTo:       createdTo,
	})
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *Handler) ExportLogs(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	providerID, ok := parseNonNegativeInt64Query(c, "provider_id")
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
			writeBadRequest(c, "success must be true or false")
			return
		}
		success = &parsed
	}

	items, err := h.store.ExportRequestLogs(c.Request.Context(), entity.ListRequestLogsInput{
		ProviderID:      providerID,
		ModelPublicName: strings.TrimSpace(c.Query("model_public_name")),
		Success:         success,
		HTTPStatus:      httpStatus,
		TraceID:         strings.TrimSpace(c.Query("trace_id")),
		CreatedFrom:     createdFrom,
		CreatedTo:       createdTo,
	})
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	buffer := bytes.NewBuffer(nil)
	writer := csv.NewWriter(buffer)
	_ = writer.Write([]string{
		"id",
		"trace_id",
		"client_api_key_name",
		"provider_name",
		"provider_key_name",
		"model_public_name",
		"request_type",
		"http_status",
		"success",
		"latency_ms",
		"prompt_tokens",
		"completion_tokens",
		"total_tokens",
		"error_type",
		"error_message",
		"created_at",
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
			item.ErrorType,
			item.ErrorMessage,
			item.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="request_logs.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

func (h *Handler) GetLogDetail(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
	if !ok {
		return
	}

	item, err := h.store.GetRequestLog(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "request log not found",
					"type":    "not_found",
				},
			})
			return
		}
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) GetStats(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	stats, err := h.store.GetDashboardStats(c.Request.Context())
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func writeBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "invalid_request",
		},
	})
}

func writeServiceUnavailable(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": gin.H{
			"message": "database is not configured",
			"type":    "service_unavailable",
		},
	})
}

func writeDatabaseError(c *gin.Context, err error) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"message": "resource already exists",
					"type":    "conflict",
				},
			})
			return
		case "23503":
			writeBadRequest(c, "referenced resource does not exist")
			return
		case "23514":
			writeBadRequest(c, "request violates database constraints")
			return
		}
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"message": err.Error(),
			"type":    "database_error",
		},
	})
}

func normalizeJSON(input json.RawMessage) json.RawMessage {
	trimmed := strings.TrimSpace(string(input))
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}

	if !json.Valid([]byte(trimmed)) {
		return json.RawMessage(`{}`)
	}

	return json.RawMessage(trimmed)
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return strings.TrimSpace(value)
}

func defaultOptionalInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}

	return *value
}

func defaultBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}

	return *value
}

func parseResourceID(c *gin.Context) (int64, bool) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || parsed <= 0 {
		writeBadRequest(c, "id must be a positive integer")
		return 0, false
	}

	return parsed, true
}

func parsePositiveIntQuery(c *gin.Context, key string, fallback int, min int, max int) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback, true
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < min {
		writeBadRequest(c, key+" must be a positive integer")
		return 0, false
	}
	if parsed > max {
		parsed = max
	}

	return parsed, true
}

func parseNonNegativeIntQuery(c *gin.Context, key string) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, true
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		writeBadRequest(c, key+" must be a non-negative integer")
		return 0, false
	}

	return parsed, true
}

func parseNonNegativeInt64Query(c *gin.Context, key string) (int64, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, true
	}

	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed < 0 {
		writeBadRequest(c, key+" must be a non-negative integer")
		return 0, false
	}

	return parsed, true
}

func parseTimeQuery(c *gin.Context, key string) (*time.Time, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, true
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		writeBadRequest(c, key+" must be an RFC3339 datetime")
		return nil, false
	}

	return &parsed, true
}

func parseOptionalTime(value *string) (*time.Time, bool) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, true
	}

	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*value))
	if err != nil {
		return nil, false
	}

	return &parsed, true
}
