package admin

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/repository"
	adminauthsvc "wcstransfer/backend/internal/service/adminauth"
	"wcstransfer/backend/internal/service/clientquota"
	"wcstransfer/backend/internal/service/keyhealth"
)

type Handler struct {
	store     repository.AdminStore
	keyHealth *keyhealth.Tracker
	quota     *clientquota.Service
}

type adminActor struct {
	UserID      int64
	Username    string
	DisplayName string
	AuthMode    string
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

type createUserRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
	Status   string `json:"status"`
}

type updateUserStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type resetUserPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

type adjustUserWalletRequest struct {
	Amount float64 `json:"amount" binding:"required"`
	Note   string  `json:"note"`
}

type correctUserWalletRequest struct {
	Amount float64 `json:"amount" binding:"required"`
	Note   string  `json:"note" binding:"required"`
}

type createClientAPIKeyRequest struct {
	Name              string  `json:"name" binding:"required"`
	Status            string  `json:"status"`
	Description       string  `json:"description"`
	RPMLimit          int     `json:"rpm_limit"`
	DailyRequestLimit int     `json:"daily_request_limit"`
	DailyTokenLimit   int     `json:"daily_token_limit"`
	DailyCostLimit    float64 `json:"daily_cost_limit"`
	MonthlyCostLimit  float64 `json:"monthly_cost_limit"`
	WarningThreshold  float64 `json:"warning_threshold"`
	AllowedModelIDs   []int64 `json:"allowed_model_ids"`
	ExpiresAt         *string `json:"expires_at"`
}

type updateClientAPIKeyRequest struct {
	Name              string  `json:"name" binding:"required"`
	Status            string  `json:"status"`
	Description       string  `json:"description"`
	RPMLimit          int     `json:"rpm_limit"`
	DailyRequestLimit int     `json:"daily_request_limit"`
	DailyTokenLimit   int     `json:"daily_token_limit"`
	DailyCostLimit    float64 `json:"daily_cost_limit"`
	MonthlyCostLimit  float64 `json:"monthly_cost_limit"`
	WarningThreshold  float64 `json:"warning_threshold"`
	AllowedModelIDs   []int64 `json:"allowed_model_ids"`
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
	PublicName        string          `json:"public_name" binding:"required"`
	ProviderID        int64           `json:"provider_id" binding:"required"`
	UpstreamModel     string          `json:"upstream_model" binding:"required"`
	RouteStrategy     string          `json:"route_strategy"`
	IsEnabled         *bool           `json:"is_enabled"`
	MaxTokens         int             `json:"max_tokens"`
	Temperature       float64         `json:"temperature"`
	TimeoutSeconds    *int            `json:"timeout_seconds"`
	CostInputPer1M    float64         `json:"cost_input_per_1m"`
	CostOutputPer1M   float64         `json:"cost_output_per_1m"`
	SaleInputPer1M    float64         `json:"sale_input_per_1m"`
	SaleOutputPer1M   float64         `json:"sale_output_per_1m"`
	ReserveMultiplier float64         `json:"reserve_multiplier"`
	ReserveMinAmount  float64         `json:"reserve_min_amount"`
	Metadata          json.RawMessage `json:"metadata"`
}

type updateModelRequest struct {
	PublicName        string          `json:"public_name" binding:"required"`
	ProviderID        int64           `json:"provider_id" binding:"required"`
	UpstreamModel     string          `json:"upstream_model" binding:"required"`
	RouteStrategy     string          `json:"route_strategy"`
	IsEnabled         *bool           `json:"is_enabled"`
	MaxTokens         int             `json:"max_tokens"`
	Temperature       float64         `json:"temperature"`
	TimeoutSeconds    *int            `json:"timeout_seconds"`
	CostInputPer1M    float64         `json:"cost_input_per_1m"`
	CostOutputPer1M   float64         `json:"cost_output_per_1m"`
	SaleInputPer1M    float64         `json:"sale_input_per_1m"`
	SaleOutputPer1M   float64         `json:"sale_output_per_1m"`
	ReserveMultiplier float64         `json:"reserve_multiplier"`
	ReserveMinAmount  float64         `json:"reserve_min_amount"`
	Metadata          json.RawMessage `json:"metadata"`
}

func NewHandler(store repository.AdminStore, tracker *keyhealth.Tracker, quota *clientquota.Service) *Handler {
	return &Handler{store: store, keyHealth: tracker, quota: quota}
}

func (h *Handler) ListUsers(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	items, err := h.store.ListUsers(c.Request.Context())
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) CreateUser(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	var request createUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	item, err := h.store.CreateUser(c.Request.Context(), entity.CreateUserInput{
		Email:    strings.TrimSpace(request.Email),
		Password: request.Password,
		FullName: strings.TrimSpace(request.FullName),
		Status:   defaultString(strings.TrimSpace(request.Status), "active"),
	})
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	h.recordAdminAction(c, "user.create", "user", item.ID, item.Email, gin.H{
		"email":     item.Email,
		"full_name": item.FullName,
		"status":    item.Status,
	})

	c.JSON(http.StatusCreated, item)
}

func (h *Handler) UpdateUserStatus(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
	if !ok {
		return
	}

	var request updateUserStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	item, err := h.store.UpdateUserStatus(c.Request.Context(), entity.UpdateUserStatusInput{
		UserID: id,
		Status: strings.TrimSpace(request.Status),
	})
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	h.recordAdminAction(c, "user.update_status", "user", item.ID, item.Email, gin.H{"status": item.Status})
	c.JSON(http.StatusOK, item)
}

func (h *Handler) ResetUserPassword(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
	if !ok {
		return
	}

	var request resetUserPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}

	if err := h.store.ResetUserPassword(c.Request.Context(), entity.ResetUserPasswordInput{
		UserID:   id,
		Password: request.Password,
	}); err != nil {
		writeDatabaseError(c, err)
		return
	}

	h.recordAdminAction(c, "user.reset_password", "user", id, "", nil)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdjustUserWallet(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
	if !ok {
		return
	}

	var request adjustUserWalletRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}
	if request.Amount <= 0 {
		writeBadRequest(c, "amount must be greater than 0")
		return
	}

	item, err := h.store.AdjustUserWallet(c.Request.Context(), entity.UserWalletAdjustmentInput{
		UserID:     id,
		Amount:     request.Amount,
		Note:       strings.TrimSpace(request.Note),
		OperatorID: h.currentAdminActor(c).UserID,
	})
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	h.recordAdminAction(c, "user.wallet.credit", "user", item.ID, item.Email, gin.H{
		"amount":         request.Amount,
		"note":           strings.TrimSpace(request.Note),
		"wallet_balance": item.WalletBalance,
	})
	c.JSON(http.StatusOK, item)
}

func (h *Handler) CorrectUserWallet(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
	if !ok {
		return
	}

	var request correctUserWalletRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeBadRequest(c, "invalid request body")
		return
	}
	if request.Amount == 0 {
		writeBadRequest(c, "amount must not be zero")
		return
	}
	if strings.TrimSpace(request.Note) == "" {
		writeBadRequest(c, "note is required")
		return
	}

	item, err := h.store.CorrectUserWallet(c.Request.Context(), entity.UserWalletCorrectionInput{
		UserID:     id,
		Amount:     request.Amount,
		Note:       strings.TrimSpace(request.Note),
		OperatorID: h.currentAdminActor(c).UserID,
	})
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	h.recordAdminAction(c, "user.wallet.correct", "user", item.ID, item.Email, gin.H{
		"amount":         request.Amount,
		"note":           strings.TrimSpace(request.Note),
		"wallet_balance": item.WalletBalance,
	})
	c.JSON(http.StatusOK, item)
}

func (h *Handler) ListUserWalletLedger(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
	if !ok {
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

	items, err := h.store.ListUserWalletLedger(c.Request.Context(), id, page, pageSize)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *Handler) ExportUserBilling(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	id, ok := parseResourceID(c)
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

	items, err := h.store.ExportUserRequestLogs(c.Request.Context(), id, entity.ListRequestLogsInput{
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
		"id", "trace_id", "client_api_key_name", "provider_name", "provider_key_name",
		"model_public_name", "request_type", "http_status", "success", "latency_ms",
		"prompt_tokens", "completion_tokens", "total_tokens",
		"reserved_amount", "cost_amount", "billable_amount", "gross_profit",
		"error_type", "error_message", "created_at",
	})
	for _, item := range items {
		_ = writer.Write([]string{
			strconv.FormatInt(item.ID, 10),
			item.TraceID, item.ClientAPIKeyName, item.ProviderName, item.ProviderKeyName,
			item.ModelPublicName, item.RequestType,
			strconv.Itoa(item.HTTPStatus), strconv.FormatBool(item.Success), strconv.Itoa(item.LatencyMS),
			strconv.Itoa(item.PromptTokens), strconv.Itoa(item.CompletionTokens), strconv.Itoa(item.TotalTokens),
			strconv.FormatFloat(item.ReservedAmount, 'f', 8, 64),
			strconv.FormatFloat(item.CostAmount, 'f', 8, 64),
			strconv.FormatFloat(item.BillableAmount, 'f', 8, 64),
			strconv.FormatFloat(item.BillableAmount-item.CostAmount, 'f', 8, 64),
			item.ErrorType, item.ErrorMessage, item.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="user_billing.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

func (h *Handler) GetUserBillingReconciliation(c *gin.Context) {
	if h.store == nil {
		writeServiceUnavailable(c)
		return
	}

	items, err := h.store.GetUserBillingReconciliation(c.Request.Context())
	if err != nil {
		writeDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
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

	h.recordAdminAction(c, "provider.create", "provider", item.ID, item.Name, gin.H{
		"slug":          item.Slug,
		"provider_type": item.ProviderType,
		"status":        item.Status,
	})

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

	h.recordAdminAction(c, "provider.update", "provider", item.ID, item.Name, gin.H{
		"slug":          item.Slug,
		"provider_type": item.ProviderType,
		"status":        item.Status,
	})

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
	items, err = h.enrichClientKeyUsage(c.Request.Context(), items)
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
		DailyCostLimit:    request.DailyCostLimit,
		MonthlyCostLimit:  request.MonthlyCostLimit,
		WarningThreshold:  defaultFloat64(request.WarningThreshold, 80),
		AllowedModelIDs:   normalizeInt64Slice(request.AllowedModelIDs),
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

	h.recordAdminAction(c, "client_key.create", "client_api_key", item.ID, item.Name, gin.H{
		"user_id":            item.UserID,
		"status":             item.Status,
		"allowed_model_ids":  item.AllowedModelIDs,
		"daily_cost_limit":   item.DailyCostLimit,
		"monthly_cost_limit": item.MonthlyCostLimit,
	})

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
		DailyCostLimit:    request.DailyCostLimit,
		MonthlyCostLimit:  request.MonthlyCostLimit,
		WarningThreshold:  defaultFloat64(request.WarningThreshold, 80),
		AllowedModelIDs:   normalizeInt64Slice(request.AllowedModelIDs),
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

	h.recordAdminAction(c, "client_key.update", "client_api_key", item.ID, item.Name, gin.H{
		"user_id":            item.UserID,
		"status":             item.Status,
		"allowed_model_ids":  item.AllowedModelIDs,
		"daily_cost_limit":   item.DailyCostLimit,
		"monthly_cost_limit": item.MonthlyCostLimit,
	})

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

	h.recordAdminAction(c, "provider_key.create", "provider_key", item.ID, item.Name, gin.H{
		"provider_id": item.ProviderID,
		"status":      item.Status,
		"priority":    item.Priority,
		"weight":      item.Weight,
	})

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

	h.recordAdminAction(c, "provider_key.update", "provider_key", item.ID, item.Name, gin.H{
		"provider_id": item.ProviderID,
		"status":      item.Status,
		"priority":    item.Priority,
		"weight":      item.Weight,
		"api_key_set": request.APIKey != nil && strings.TrimSpace(*request.APIKey) != "",
	})

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
		PublicName:        strings.TrimSpace(request.PublicName),
		ProviderID:        request.ProviderID,
		UpstreamModel:     strings.TrimSpace(request.UpstreamModel),
		RouteStrategy:     defaultString(request.RouteStrategy, "fixed"),
		IsEnabled:         defaultBool(request.IsEnabled, true),
		MaxTokens:         request.MaxTokens,
		Temperature:       request.Temperature,
		TimeoutSeconds:    defaultOptionalInt(request.TimeoutSeconds, 120),
		CostInputPer1M:    request.CostInputPer1M,
		CostOutputPer1M:   request.CostOutputPer1M,
		SaleInputPer1M:    request.SaleInputPer1M,
		SaleOutputPer1M:   request.SaleOutputPer1M,
		ReserveMultiplier: defaultPositiveFloat(request.ReserveMultiplier, 1),
		ReserveMinAmount:  request.ReserveMinAmount,
		Metadata:          normalizeJSON(request.Metadata),
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

	h.recordAdminAction(c, "model.create", "model", item.ID, item.PublicName, gin.H{
		"provider_id":        item.ProviderID,
		"route_strategy":     item.RouteStrategy,
		"is_enabled":         item.IsEnabled,
		"sale_input_per_1m":  item.SaleInputPer1M,
		"sale_output_per_1m": item.SaleOutputPer1M,
	})

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
		ID:                id,
		PublicName:        strings.TrimSpace(request.PublicName),
		ProviderID:        request.ProviderID,
		UpstreamModel:     strings.TrimSpace(request.UpstreamModel),
		RouteStrategy:     defaultString(request.RouteStrategy, "fixed"),
		IsEnabled:         defaultBool(request.IsEnabled, true),
		MaxTokens:         request.MaxTokens,
		Temperature:       request.Temperature,
		TimeoutSeconds:    defaultOptionalInt(request.TimeoutSeconds, 120),
		CostInputPer1M:    request.CostInputPer1M,
		CostOutputPer1M:   request.CostOutputPer1M,
		SaleInputPer1M:    request.SaleInputPer1M,
		SaleOutputPer1M:   request.SaleOutputPer1M,
		ReserveMultiplier: defaultPositiveFloat(request.ReserveMultiplier, 1),
		ReserveMinAmount:  request.ReserveMinAmount,
		Metadata:          normalizeJSON(request.Metadata),
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

	h.recordAdminAction(c, "model.update", "model", item.ID, item.PublicName, gin.H{
		"provider_id":        item.ProviderID,
		"route_strategy":     item.RouteStrategy,
		"is_enabled":         item.IsEnabled,
		"sale_input_per_1m":  item.SaleInputPer1M,
		"sale_output_per_1m": item.SaleOutputPer1M,
	})

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
		"reserved_amount",
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
			strconv.FormatFloat(item.ReservedAmount, 'f', 8, 64),
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

	clientKeys, err := h.store.ListClientAPIKeys(c.Request.Context())
	if err != nil {
		writeDatabaseError(c, err)
		return
	}
	clientKeys, err = h.enrichClientKeyUsage(c.Request.Context(), clientKeys)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}
	stats.QuotaPressure = buildQuotaPressure(clientKeys, 5)
	stats.BudgetPressure = buildBudgetPressure(clientKeys, 5)

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) enrichClientKeyUsage(ctx context.Context, items []entity.ClientAPIKey) ([]entity.ClientAPIKey, error) {
	if h.quota == nil || len(items) == 0 {
		return items, nil
	}

	usageByID, err := h.quota.GetUsageBatch(ctx, items)
	if err != nil {
		return nil, err
	}

	for index := range items {
		usage, ok := usageByID[items[index].ID]
		if !ok {
			continue
		}
		usageCopy := usage
		items[index].Usage = &usageCopy
	}

	return items, nil
}

func buildQuotaPressure(items []entity.ClientAPIKey, limit int) []entity.ClientQuotaPressure {
	pressure := make([]entity.ClientQuotaPressure, 0)
	for _, item := range items {
		if item.Usage == nil {
			continue
		}

		highest := maxUsagePercent(
			item.Usage.RPMUsagePercent,
			item.Usage.DailyRequestUsagePercent,
			item.Usage.DailyTokenUsagePercent,
		)
		if highest <= 0 {
			continue
		}

		dimensions := make([]string, 0, 3)
		if item.Usage.IsRPMLimited {
			dimensions = append(dimensions, "rpm")
		}
		if item.Usage.IsDailyRequestLimited {
			dimensions = append(dimensions, "daily_requests")
		}
		if item.Usage.IsDailyTokenLimited {
			dimensions = append(dimensions, "daily_tokens")
		}
		if len(dimensions) == 0 && highest < 60 {
			continue
		}

		pressure = append(pressure, entity.ClientQuotaPressure{
			ClientAPIKeyID:           item.ID,
			ClientAPIKeyName:         item.Name,
			HighestUsagePercent:      highest,
			RPMUsagePercent:          item.Usage.RPMUsagePercent,
			DailyRequestUsagePercent: item.Usage.DailyRequestUsagePercent,
			DailyTokenUsagePercent:   item.Usage.DailyTokenUsagePercent,
			LimitedDimensions:        dimensions,
		})
	}

	sort.Slice(pressure, func(i, j int) bool {
		if pressure[i].HighestUsagePercent == pressure[j].HighestUsagePercent {
			return pressure[i].ClientAPIKeyID > pressure[j].ClientAPIKeyID
		}
		return pressure[i].HighestUsagePercent > pressure[j].HighestUsagePercent
	})

	if limit > 0 && len(pressure) > limit {
		return pressure[:limit]
	}
	return pressure
}

func maxUsagePercent(values ...float64) float64 {
	var max float64
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

func buildBudgetPressure(items []entity.ClientAPIKey, limit int) []entity.ClientBudgetPressure {
	pressure := make([]entity.ClientBudgetPressure, 0)
	for _, item := range items {
		if item.CostUsage == nil {
			continue
		}

		highest := maxUsagePercent(
			item.CostUsage.DailyCostUsagePercent,
			item.CostUsage.MonthlyCostUsagePercent,
		)
		if highest <= 0 {
			continue
		}

		dimensions := make([]string, 0, 2)
		if item.CostUsage.IsDailyCostLimited {
			dimensions = append(dimensions, "daily_cost")
		}
		if item.CostUsage.IsMonthlyCostLimited {
			dimensions = append(dimensions, "monthly_cost")
		}
		if len(dimensions) == 0 && !item.CostUsage.IsWarningTriggered && highest < 60 {
			continue
		}

		pressure = append(pressure, entity.ClientBudgetPressure{
			ClientAPIKeyID:          item.ID,
			ClientAPIKeyName:        item.Name,
			HighestUsagePercent:     highest,
			DailyCostUsagePercent:   item.CostUsage.DailyCostUsagePercent,
			MonthlyCostUsagePercent: item.CostUsage.MonthlyCostUsagePercent,
			IsWarningTriggered:      item.CostUsage.IsWarningTriggered,
			LimitedDimensions:       dimensions,
		})
	}

	sort.Slice(pressure, func(i, j int) bool {
		if pressure[i].HighestUsagePercent == pressure[j].HighestUsagePercent {
			return pressure[i].ClientAPIKeyID > pressure[j].ClientAPIKeyID
		}
		return pressure[i].HighestUsagePercent > pressure[j].HighestUsagePercent
	})

	if limit > 0 && len(pressure) > limit {
		return pressure[:limit]
	}
	return pressure
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

func defaultFloat64(value float64, fallback float64) float64 {
	if value == 0 {
		return fallback
	}

	return value
}

func defaultPositiveFloat(value float64, fallback float64) float64 {
	if value <= 0 {
		return fallback
	}

	return value
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

func normalizeInt64Slice(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(values))
	items := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}

	return items
}

func (h *Handler) currentAdminActor(c *gin.Context) adminActor {
	actor := adminActor{}
	if rawMode, ok := c.Get("admin_auth_mode"); ok {
		if mode, valid := rawMode.(string); valid {
			actor.AuthMode = strings.TrimSpace(mode)
		}
	}
	if claims, ok := adminauthsvc.ClaimsFromContext(c); ok {
		actor.UserID = claims.Sub
		actor.Username = strings.TrimSpace(claims.Username)
		actor.DisplayName = strings.TrimSpace(claims.DisplayName)
	}
	if actor.AuthMode == "" {
		actor.AuthMode = "session"
	}
	return actor
}

func (h *Handler) recordAdminAction(c *gin.Context, action string, resourceType string, resourceID int64, resourceName string, metadata any) {
	if h.store == nil {
		return
	}

	actor := h.currentAdminActor(c)
	var payload json.RawMessage
	if metadata != nil {
		if encoded, err := json.Marshal(metadata); err == nil {
			payload = encoded
		}
	}

	// Best-effort write to avoid turning a successful mutation into a confusing 500 after commit.
	_ = h.store.CreateAdminActionLog(c.Request.Context(), entity.CreateAdminActionLogInput{
		AdminUserID:      actor.UserID,
		AdminUsername:    actor.Username,
		AdminDisplayName: actor.DisplayName,
		AuthMode:         actor.AuthMode,
		Action:           action,
		ResourceType:     resourceType,
		ResourceID:       resourceID,
		ResourceName:     strings.TrimSpace(resourceName),
		RequestMethod:    c.Request.Method,
		RequestPath:      c.FullPath(),
		ClientIP:         c.ClientIP(),
		Metadata:         payload,
	})
}
