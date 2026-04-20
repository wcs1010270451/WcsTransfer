package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/middleware"
	"wcstransfer/backend/internal/repository"
	"wcstransfer/backend/internal/service/clientquota"
	"wcstransfer/backend/internal/service/keyhealth"
)

const maxLoggedPayloadBytes = 64 * 1024

const (
	keyCooldownTransient    = 10 * time.Second
	keyCooldownRateLimited  = 30 * time.Second
	keyCooldownUnauthorized = 10 * time.Minute
)

type Handler struct {
	store      repository.PublicModelStore
	logWriter  repository.RequestLogWriter
	httpClient *http.Client
	counters   sync.Map
	keyHealth  *keyhealth.Tracker
	quota      *clientquota.Service
}

type chatLogState struct {
	traceID          string
	requestType      string
	modelPublicName  string
	upstreamModel    string
	providerID       int64
	providerKeyID    int64
	clientAPIKeyID   int64
	clientAPIKeyName string
	clientIP         string
	requestMethod    string
	requestPath      string
	httpStatus       int
	success          bool
	latencyMS        int
	promptTokens     int
	completionTokens int
	totalTokens      int
	costAmount       float64
	billableAmount   float64
	errorType        string
	errorMessage     string
	requestPayload   json.RawMessage
	responsePayload  json.RawMessage
	metadata         map[string]any
}

type proxyAttempt struct {
	Key   entity.ProviderKey
	Phase string
}

type chatCompletionOptions struct {
	RequestBody      []byte
	RequestType      string
	RouteStrategy    string
	ProviderKeyID    int64
	IgnoreKeyHealth  bool
	EmitDebugHeaders bool
}

type adminDebugChatRequest struct {
	Payload       json.RawMessage `json:"payload" binding:"required"`
	ProviderKeyID int64           `json:"provider_key_id"`
	RouteStrategy string          `json:"route_strategy"`
}

func NewHandler(
	store repository.PublicModelStore,
	logWriter repository.RequestLogWriter,
	httpClient *http.Client,
	tracker *keyhealth.Tracker,
	quota *clientquota.Service,
) *Handler {
	client := httpClient
	if client == nil {
		client = &http.Client{}
	}
	if tracker == nil {
		tracker = keyhealth.NewTracker()
	}

	return &Handler{
		store:      store,
		logWriter:  logWriter,
		httpClient: client,
		keyHealth:  tracker,
		quota:      quota,
	}
}

func (h *Handler) ListModels(c *gin.Context) {
	items := make([]gin.H, 0)
	clientKey, hasClientKey := middleware.ClientAPIKeyFromContext(c)
	if h.store != nil {
		models, err := h.store.ListEnabledModels(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": err.Error(),
					"type":    "database_error",
				},
			})
			return
		}

		for _, model := range models {
			if hasClientKey && !clientKeyAllowsModel(clientKey, model.PublicName) {
				continue
			}
			items = append(items, gin.H{
				"id":         model.PublicName,
				"object":     "model",
				"created":    model.CreatedAt.Unix(),
				"owned_by":   model.ProviderName,
				"permission": []any{},
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   items,
	})
}

func (h *Handler) ChatCompletions(c *gin.Context) {
	h.handleChatCompletions(c, chatCompletionOptions{
		RequestType: "chat_completions",
	})
}

func (h *Handler) Embeddings(c *gin.Context) {
	h.handleEmbeddings(c, chatCompletionOptions{
		RequestType: "embeddings",
	})
}

func (h *Handler) AdminDebugChatCompletions(c *gin.Context) {
	var request adminDebugChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "invalid request body",
				"type":    "invalid_request",
			},
		})
		return
	}

	routeStrategy := strings.TrimSpace(request.RouteStrategy)
	if routeStrategy != "" && !isSupportedRouteStrategy(routeStrategy) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "route_strategy must be one of fixed, failover, round_robin",
				"type":    "invalid_request",
			},
		})
		return
	}

	h.handleChatCompletions(c, chatCompletionOptions{
		RequestBody:      request.Payload,
		RequestType:      "chat_completions_debug",
		RouteStrategy:    routeStrategy,
		ProviderKeyID:    request.ProviderKeyID,
		IgnoreKeyHealth:  request.ProviderKeyID > 0,
		EmitDebugHeaders: true,
	})
}

func (h *Handler) AdminDebugEmbeddings(c *gin.Context) {
	var request adminDebugChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "invalid request body",
				"type":    "invalid_request",
			},
		})
		return
	}

	routeStrategy := strings.TrimSpace(request.RouteStrategy)
	if routeStrategy != "" && !isSupportedRouteStrategy(routeStrategy) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "route_strategy must be one of fixed, failover, round_robin",
				"type":    "invalid_request",
			},
		})
		return
	}

	h.handleEmbeddings(c, chatCompletionOptions{
		RequestBody:      request.Payload,
		RequestType:      "embeddings_debug",
		RouteStrategy:    routeStrategy,
		ProviderKeyID:    request.ProviderKeyID,
		IgnoreKeyHealth:  request.ProviderKeyID > 0,
		EmitDebugHeaders: true,
	})
}

func (h *Handler) handleChatCompletions(c *gin.Context, options chatCompletionOptions) {
	startedAt := time.Now()
	logState := chatLogState{
		traceID:       strings.TrimSpace(c.Writer.Header().Get("X-Request-Id")),
		requestType:   firstNonEmpty(options.RequestType, "chat_completions"),
		clientIP:      c.ClientIP(),
		requestMethod: c.Request.Method,
		requestPath:   c.FullPath(),
		metadata: map[string]any{
			"stream": false,
		},
	}
	if clientKey, ok := middleware.ClientAPIKeyFromContext(c); ok {
		logState.clientAPIKeyID = clientKey.ID
		logState.clientAPIKeyName = clientKey.Name
		logState.metadata["client_api_key_name"] = clientKey.Name
		defer func(key entity.ClientAPIKey) {
			if logState.totalTokens > 0 {
				_ = h.quota.AddTokenUsage(c.Request.Context(), key, logState.totalTokens)
			}
		}(clientKey)
	}

	defer func() {
		h.writeRequestLog(c.Request.Context(), startedAt, logState)
	}()

	writeJSONError := func(statusCode int, errorType string, message string) {
		logState.httpStatus = statusCode
		logState.errorType = errorType
		logState.errorMessage = message
		logState.responsePayload = wrappedPayloadJSON(map[string]any{
			"error": map[string]any{
				"message": message,
				"type":    errorType,
			},
		})
		h.writeDebugHeaders(c.Writer.Header(), logState, options)
		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"message": message,
				"type":    errorType,
			},
		})
	}

	if h.store == nil {
		writeJSONError(http.StatusServiceUnavailable, "service_unavailable", "model store is not configured")
		return
	}

	body := options.RequestBody
	if len(body) == 0 {
		var err error
		body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			writeJSONError(http.StatusBadRequest, "invalid_request", "failed to read request body")
			return
		}
	}

	logState.requestPayload = payloadForStorage(body, "application/json", false)

	payload, publicModel, err := parseChatCompletionPayload(body)
	if err != nil {
		writeJSONError(http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	logState.modelPublicName = publicModel
	if clientKey, ok := middleware.ClientAPIKeyFromContext(c); ok {
		if !clientKeyAllowsModel(clientKey, publicModel) {
			logState.httpStatus = http.StatusForbidden
			logState.errorType = "model_forbidden"
			logState.errorMessage = "client api key is not allowed to access this model"
			logState.responsePayload = wrappedPayloadJSON(map[string]any{
				"error": map[string]any{
					"message": logState.errorMessage,
					"type":    logState.errorType,
				},
			})
			h.writeDebugHeaders(c.Writer.Header(), logState, options)
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"message": logState.errorMessage,
					"type":    logState.errorType,
				},
			})
			return
		}
		if exceeded, message := clientKeyBudgetExceeded(clientKey); exceeded {
			logState.httpStatus = http.StatusTooManyRequests
			logState.errorType = "budget_exceeded"
			logState.errorMessage = message
			logState.responsePayload = wrappedPayloadJSON(map[string]any{
				"error": map[string]any{
					"message": logState.errorMessage,
					"type":    logState.errorType,
				},
			})
			h.writeDebugHeaders(c.Writer.Header(), logState, options)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": logState.errorMessage,
					"type":    logState.errorType,
				},
			})
			return
		}
		if clientKey.CostUsage != nil && clientKey.CostUsage.IsWarningTriggered {
			logState.metadata["budget_warning"] = true
		}
	}

	streamRequested := extractStreamFlag(payload)
	logState.metadata["stream"] = streamRequested
	if streamRequested {
		ensureStreamOptionsIncludeUsage(payload)
		logState.metadata["stream_include_usage"] = true
	}

	route, err := h.store.ResolveModelRoute(c.Request.Context(), publicModel)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(http.StatusNotFound, "not_found", "model not found or unavailable")
			return
		}

		writeJSONError(http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	logState.upstreamModel = route.Model.UpstreamModel
	logState.providerID = route.Provider.ID
	logState.metadata["provider_slug"] = route.Provider.Slug
	logState.metadata["provider_type"] = route.Provider.ProviderType
	logState.metadata["route_strategy"] = route.Model.RouteStrategy
	logState.metadata["cost_input_per_1m"] = route.Model.CostInputPer1M
	logState.metadata["cost_output_per_1m"] = route.Model.CostOutputPer1M
	logState.metadata["sale_input_per_1m"] = route.Model.SaleInputPer1M
	logState.metadata["sale_output_per_1m"] = route.Model.SaleOutputPer1M

	route, err = applyChatRouteOptions(route, options, logState.metadata)
	if err != nil {
		writeJSONError(http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	attempts, skippedKeys, err := h.buildProxyAttempts(route, options.IgnoreKeyHealth)
	if err != nil {
		writeJSONError(http.StatusServiceUnavailable, "routing_error", err.Error())
		return
	}
	logState.metadata["candidate_key_count"] = len(route.Keys)
	if len(skippedKeys) > 0 {
		logState.metadata["temporarily_skipped_keys"] = skippedKeys
	}

	payload["model"] = route.Model.UpstreamModel
	if route.Model.MaxTokens > 0 {
		if _, exists := payload["max_tokens"]; !exists {
			payload["max_tokens"] = route.Model.MaxTokens
		}
	}
	if _, exists := payload["temperature"]; !exists {
		payload["temperature"] = route.Model.Temperature
	}

	rewrittenBody, err := json.Marshal(payload)
	if err != nil {
		writeJSONError(http.StatusInternalServerError, "internal_error", "failed to encode upstream payload")
		return
	}

	timeout := 120 * time.Second
	if route.Model.TimeoutSeconds > 0 {
		timeout = time.Duration(route.Model.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	failoverCount := 0
	retryCount := 0
	attemptMetadata := make([]map[string]any, 0, len(attempts))
	logState.metadata["routing_attempts"] = attemptMetadata

	for attemptIndex, attempt := range attempts {
		hasNextAttempt := attemptIndex < len(attempts)-1
		nextUsesSameKey := false
		if hasNextAttempt {
			nextUsesSameKey = attempts[attemptIndex+1].Key.ID == attempt.Key.ID
		}
		logState.providerKeyID = attempt.Key.ID
		logState.metadata["provider_key_name"] = attempt.Key.Name

		upstreamRequest, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			buildUpstreamURL(route.Provider.BaseURL, "/chat/completions"),
			bytes.NewReader(rewrittenBody),
		)
		if err != nil {
			writeJSONError(http.StatusInternalServerError, "internal_error", "failed to create upstream request")
			return
		}

		upstreamRequest.Header.Set("Authorization", "Bearer "+attempt.Key.APIKey)
		upstreamRequest.Header.Set("Content-Type", contentTypeOrDefault(c.GetHeader("Content-Type"), "application/json"))
		upstreamRequest.Header.Set("Accept", contentTypeOrDefault(c.GetHeader("Accept"), "application/json"))
		copyOptionalHeader(c, upstreamRequest, "OpenAI-Beta")

		attemptRecord := map[string]any{
			"attempt":           attemptIndex + 1,
			"phase":             attempt.Phase,
			"provider_key_id":   attempt.Key.ID,
			"provider_key_name": attempt.Key.Name,
		}

		response, requestErr := h.httpClient.Do(upstreamRequest)
		if requestErr != nil {
			h.penalizeKey(attempt.Key.ID, keyCooldownTransient, "request_error")
			attemptRecord["error_type"] = "upstream_error"
			attemptRecord["error_message"] = requestErr.Error()
			attemptRecord["retryable"] = true

			if hasNextAttempt {
				attemptRecord["decision"] = nextAttemptDecision(attempts[attemptIndex+1].Phase)
				if attempts[attemptIndex+1].Key.ID == attempt.Key.ID {
					retryCount++
				} else {
					failoverCount++
				}
				attemptMetadata = append(attemptMetadata, attemptRecord)
				continue
			}

			attemptRecord["decision"] = "return_error"
			attemptMetadata = append(attemptMetadata, attemptRecord)
			logState.metadata["routing_attempts"] = attemptMetadata
			logState.metadata["failover_count"] = failoverCount
			logState.metadata["retry_count"] = retryCount
			writeJSONError(http.StatusBadGateway, "upstream_error", requestErr.Error())
			return
		}

		responseContentType := response.Header.Get("Content-Type")
		attemptRecord["http_status"] = response.StatusCode
		attemptRecord["response_content_type"] = responseContentType

		if shouldFailoverStatus(response.StatusCode) && hasNextAttempt && (!nextUsesSameKey || shouldRetryStatus(response.StatusCode)) {
			responseBody, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			h.applyStatusPenalty(attempt.Key.ID, response.StatusCode)
			attemptRecord["error_type"] = "upstream_response_error"
			attemptRecord["error_message"] = extractErrorMessage(responseBody)
			attemptRecord["retryable"] = shouldRetryStatus(response.StatusCode)
			attemptRecord["decision"] = nextAttemptDecision(attempts[attemptIndex+1].Phase)
			attemptMetadata = append(attemptMetadata, attemptRecord)
			if attempts[attemptIndex+1].Key.ID == attempt.Key.ID {
				retryCount++
			} else {
				failoverCount++
			}
			continue
		}

		attemptRecord["decision"] = "return_response"
		attemptMetadata = append(attemptMetadata, attemptRecord)
		logState.metadata["routing_attempts"] = attemptMetadata
		logState.metadata["failover_count"] = failoverCount
		logState.metadata["retry_count"] = retryCount
		logState.httpStatus = response.StatusCode
		logState.success = response.StatusCode >= 200 && response.StatusCode < 300
		logState.metadata["response_content_type"] = responseContentType
		if logState.success {
			h.clearKeyPenalty(attempt.Key.ID)
		}

		h.writeDebugHeaders(c.Writer.Header(), logState, options)
		copyResponseHeaders(c.Writer.Header(), response.Header)
		c.Status(response.StatusCode)

		if isStreamingResponse(responseContentType) {
			defer response.Body.Close()
			logState.metadata["stream_response"] = true
			preview, truncated, streamErr := streamUpstreamResponse(c, response.Body, logState.metadata, &logState, route.Model)
			logState.responsePayload = wrappedPayloadJSON(map[string]any{
				"content_type": responseContentType,
				"stream":       true,
				"truncated":    truncated,
				"preview":      string(preview),
			})
			if streamErr != nil {
				logState.success = false
				logState.errorType = "stream_proxy_error"
				logState.errorMessage = streamErr.Error()
			}
			return
		}

		responseBody, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if readErr != nil {
			logState.httpStatus = http.StatusBadGateway
			logState.success = false
			logState.errorType = "upstream_read_error"
			logState.errorMessage = readErr.Error()
			logState.responsePayload = wrappedPayloadJSON(map[string]any{
				"error": map[string]any{
					"message": logState.errorMessage,
					"type":    logState.errorType,
				},
			})
			c.Writer.Header().Set("Content-Type", "application/json")
			h.writeDebugHeaders(c.Writer.Header(), logState, options)
			c.Status(http.StatusBadGateway)
			_, _ = c.Writer.Write(logState.responsePayload)
			return
		}

		logState.responsePayload = payloadForStorage(responseBody, responseContentType, streamRequested)
		if !logState.success && logState.errorType == "" {
			logState.errorType = "upstream_response_error"
			logState.errorMessage = extractErrorMessage(responseBody)
		}

		applyUsage(logState.metadata, &logState, responseBody, route.Model)

		if _, err := c.Writer.Write(responseBody); err != nil {
			logState.success = false
			logState.errorType = "client_write_error"
			logState.errorMessage = err.Error()
		}
		return
	}
}

func (h *Handler) handleEmbeddings(c *gin.Context, options chatCompletionOptions) {
	startedAt := time.Now()
	logState := chatLogState{
		traceID:       strings.TrimSpace(c.Writer.Header().Get("X-Request-Id")),
		requestType:   firstNonEmpty(options.RequestType, "embeddings"),
		clientIP:      c.ClientIP(),
		requestMethod: c.Request.Method,
		requestPath:   c.FullPath(),
		metadata:      map[string]any{},
	}
	if clientKey, ok := middleware.ClientAPIKeyFromContext(c); ok {
		logState.clientAPIKeyID = clientKey.ID
		logState.clientAPIKeyName = clientKey.Name
		logState.metadata["client_api_key_name"] = clientKey.Name
		defer func(key entity.ClientAPIKey) {
			if logState.totalTokens > 0 {
				_ = h.quota.AddTokenUsage(c.Request.Context(), key, logState.totalTokens)
			}
		}(clientKey)
	}
	defer func() {
		h.writeRequestLog(c.Request.Context(), startedAt, logState)
	}()

	writeJSONError := func(statusCode int, errorType string, message string) {
		logState.httpStatus = statusCode
		logState.errorType = errorType
		logState.errorMessage = message
		logState.responsePayload = wrappedPayloadJSON(map[string]any{
			"error": map[string]any{"message": message, "type": errorType},
		})
		h.writeDebugHeaders(c.Writer.Header(), logState, options)
		c.JSON(statusCode, gin.H{
			"error": gin.H{"message": message, "type": errorType},
		})
	}

	if h.store == nil {
		writeJSONError(http.StatusServiceUnavailable, "service_unavailable", "model store is not configured")
		return
	}

	body := options.RequestBody
	if len(body) == 0 {
		var err error
		body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			writeJSONError(http.StatusBadRequest, "invalid_request", "failed to read request body")
			return
		}
	}
	logState.requestPayload = payloadForStorage(body, "application/json", false)

	payload, publicModel, err := parseEmbeddingsPayload(body)
	if err != nil {
		writeJSONError(http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	logState.modelPublicName = publicModel
	if clientKey, ok := middleware.ClientAPIKeyFromContext(c); ok {
		if !clientKeyAllowsModel(clientKey, publicModel) {
			writeJSONError(http.StatusForbidden, "model_forbidden", "client api key is not allowed to access this model")
			return
		}
		if exceeded, message := clientKeyBudgetExceeded(clientKey); exceeded {
			writeJSONError(http.StatusTooManyRequests, "budget_exceeded", message)
			return
		}
		if clientKey.CostUsage != nil && clientKey.CostUsage.IsWarningTriggered {
			logState.metadata["budget_warning"] = true
		}
	}

	route, err := h.store.ResolveModelRoute(c.Request.Context(), publicModel)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(http.StatusNotFound, "not_found", "model not found or unavailable")
			return
		}
		writeJSONError(http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	logState.upstreamModel = route.Model.UpstreamModel
	logState.providerID = route.Provider.ID
	logState.metadata["provider_slug"] = route.Provider.Slug
	logState.metadata["provider_type"] = route.Provider.ProviderType
	logState.metadata["route_strategy"] = route.Model.RouteStrategy
	logState.metadata["cost_input_per_1m"] = route.Model.CostInputPer1M
	logState.metadata["cost_output_per_1m"] = route.Model.CostOutputPer1M
	logState.metadata["sale_input_per_1m"] = route.Model.SaleInputPer1M
	logState.metadata["sale_output_per_1m"] = route.Model.SaleOutputPer1M

	route, err = applyChatRouteOptions(route, options, logState.metadata)
	if err != nil {
		writeJSONError(http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	attempts, skippedKeys, err := h.buildProxyAttempts(route, options.IgnoreKeyHealth)
	if err != nil {
		writeJSONError(http.StatusServiceUnavailable, "routing_error", err.Error())
		return
	}
	logState.metadata["candidate_key_count"] = len(route.Keys)
	if len(skippedKeys) > 0 {
		logState.metadata["temporarily_skipped_keys"] = skippedKeys
	}

	payload["model"] = route.Model.UpstreamModel
	rewrittenBody, err := json.Marshal(payload)
	if err != nil {
		writeJSONError(http.StatusInternalServerError, "internal_error", "failed to encode upstream payload")
		return
	}

	timeout := 120 * time.Second
	if route.Model.TimeoutSeconds > 0 {
		timeout = time.Duration(route.Model.TimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	failoverCount := 0
	retryCount := 0
	attemptMetadata := make([]map[string]any, 0, len(attempts))
	logState.metadata["routing_attempts"] = attemptMetadata

	for attemptIndex, attempt := range attempts {
		hasNextAttempt := attemptIndex < len(attempts)-1
		logState.providerKeyID = attempt.Key.ID
		logState.metadata["provider_key_name"] = attempt.Key.Name

		upstreamRequest, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			buildUpstreamURL(route.Provider.BaseURL, "/embeddings"),
			bytes.NewReader(rewrittenBody),
		)
		if err != nil {
			writeJSONError(http.StatusInternalServerError, "internal_error", "failed to create upstream request")
			return
		}
		upstreamRequest.Header.Set("Authorization", "Bearer "+attempt.Key.APIKey)
		upstreamRequest.Header.Set("Content-Type", contentTypeOrDefault(c.GetHeader("Content-Type"), "application/json"))
		upstreamRequest.Header.Set("Accept", contentTypeOrDefault(c.GetHeader("Accept"), "application/json"))
		copyOptionalHeader(c, upstreamRequest, "OpenAI-Beta")

		attemptRecord := map[string]any{
			"attempt":           attemptIndex + 1,
			"phase":             attempt.Phase,
			"provider_key_id":   attempt.Key.ID,
			"provider_key_name": attempt.Key.Name,
		}

		response, requestErr := h.httpClient.Do(upstreamRequest)
		if requestErr != nil {
			h.penalizeKey(attempt.Key.ID, keyCooldownTransient, "request_error")
			attemptRecord["error_type"] = "upstream_error"
			attemptRecord["error_message"] = requestErr.Error()
			attemptRecord["retryable"] = true
			if hasNextAttempt {
				attemptRecord["decision"] = nextAttemptDecision(attempts[attemptIndex+1].Phase)
				attemptMetadata = append(attemptMetadata, attemptRecord)
				if attempts[attemptIndex+1].Key.ID == attempt.Key.ID {
					retryCount++
				} else {
					failoverCount++
				}
				continue
			}
			attemptRecord["decision"] = "return_error"
			attemptMetadata = append(attemptMetadata, attemptRecord)
			logState.metadata["routing_attempts"] = attemptMetadata
			logState.metadata["failover_count"] = failoverCount
			logState.metadata["retry_count"] = retryCount
			writeJSONError(http.StatusBadGateway, "upstream_error", requestErr.Error())
			return
		}

		responseContentType := response.Header.Get("Content-Type")
		attemptRecord["http_status"] = response.StatusCode
		attemptRecord["response_content_type"] = responseContentType

		if shouldFailoverStatus(response.StatusCode) && hasNextAttempt {
			responseBody, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			h.applyStatusPenalty(attempt.Key.ID, response.StatusCode)
			attemptRecord["error_type"] = "upstream_response_error"
			attemptRecord["error_message"] = extractErrorMessage(responseBody)
			attemptRecord["decision"] = nextAttemptDecision(attempts[attemptIndex+1].Phase)
			attemptMetadata = append(attemptMetadata, attemptRecord)
			if attempts[attemptIndex+1].Key.ID == attempt.Key.ID {
				retryCount++
			} else {
				failoverCount++
			}
			continue
		}

		attemptRecord["decision"] = "return_response"
		attemptMetadata = append(attemptMetadata, attemptRecord)
		logState.metadata["routing_attempts"] = attemptMetadata
		logState.metadata["failover_count"] = failoverCount
		logState.metadata["retry_count"] = retryCount
		logState.httpStatus = response.StatusCode
		logState.success = response.StatusCode >= 200 && response.StatusCode < 300
		logState.metadata["response_content_type"] = responseContentType
		if logState.success {
			h.clearKeyPenalty(attempt.Key.ID)
		}

		h.writeDebugHeaders(c.Writer.Header(), logState, options)
		copyResponseHeaders(c.Writer.Header(), response.Header)
		c.Status(response.StatusCode)

		responseBody, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if readErr != nil {
			writeJSONError(http.StatusBadGateway, "upstream_read_error", readErr.Error())
			return
		}

		logState.responsePayload = payloadForStorage(responseBody, responseContentType, false)
		if !logState.success && logState.errorType == "" {
			logState.errorType = "upstream_response_error"
			logState.errorMessage = extractErrorMessage(responseBody)
		}
		applyUsage(logState.metadata, &logState, responseBody, route.Model)

		if _, err := c.Writer.Write(responseBody); err != nil {
			logState.success = false
			logState.errorType = "client_write_error"
			logState.errorMessage = err.Error()
		}
		return
	}
}

func parseChatCompletionPayload(body []byte) (map[string]any, string, error) {
	payload := make(map[string]any)
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", errors.New("request body must be valid JSON")
	}

	rawModel, ok := payload["model"]
	if !ok {
		return nil, "", errors.New("model is required")
	}

	modelName, ok := rawModel.(string)
	if !ok || strings.TrimSpace(modelName) == "" {
		return nil, "", errors.New("model must be a non-empty string")
	}

	return payload, strings.TrimSpace(modelName), nil
}

func parseEmbeddingsPayload(body []byte) (map[string]any, string, error) {
	payload := make(map[string]any)
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", errors.New("request body must be valid JSON")
	}

	rawModel, ok := payload["model"]
	if !ok {
		return nil, "", errors.New("model is required")
	}
	modelName, ok := rawModel.(string)
	if !ok || strings.TrimSpace(modelName) == "" {
		return nil, "", errors.New("model must be a non-empty string")
	}
	if _, ok := payload["input"]; !ok {
		return nil, "", errors.New("input is required")
	}

	return payload, strings.TrimSpace(modelName), nil
}

func (h *Handler) orderedKeys(route entity.ModelRoute, ignoreKeyHealth bool) ([]entity.ProviderKey, []map[string]any, error) {
	if len(route.Keys) == 0 {
		return nil, nil, errors.New("no active provider key is available for the requested model")
	}

	keys := append([]entity.ProviderKey(nil), route.Keys...)
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Priority != keys[j].Priority {
			return keys[i].Priority < keys[j].Priority
		}
		if keys[i].Weight != keys[j].Weight {
			return keys[i].Weight > keys[j].Weight
		}
		return keys[i].ID < keys[j].ID
	})

	if route.Model.RouteStrategy == "round_robin" {
		counterValue, _ := h.counters.LoadOrStore(route.Model.PublicName, &atomic.Uint64{})
		counter := counterValue.(*atomic.Uint64)
		index := int(counter.Add(1)-1) % len(keys)
		rotated := append([]entity.ProviderKey{}, keys[index:]...)
		rotated = append(rotated, keys[:index]...)
		keys = rotated
	}

	if ignoreKeyHealth {
		return keys, nil, nil
	}

	filtered := make([]entity.ProviderKey, 0, len(keys))
	skipped := make([]map[string]any, 0)
	now := time.Now()
	for _, key := range keys {
		penalty, penalized := h.currentKeyPenalty(key.ID, now)
		if !penalized {
			filtered = append(filtered, key)
			continue
		}

		skipped = append(skipped, map[string]any{
			"provider_key_id":   key.ID,
			"provider_key_name": key.Name,
			"reason":            penalty.Reason,
			"blocked_until":     penalty.Until.UTC().Format(time.RFC3339),
		})
	}

	if len(filtered) == 0 {
		return keys, skipped, nil
	}

	return filtered, skipped, nil
}

func (h *Handler) buildProxyAttempts(route entity.ModelRoute, ignoreKeyHealth bool) ([]proxyAttempt, []map[string]any, error) {
	keys, skipped, err := h.orderedKeys(route, ignoreKeyHealth)
	if err != nil {
		return nil, nil, err
	}

	attempts := make([]proxyAttempt, 0, len(keys)+1)
	for _, key := range keys {
		attempts = append(attempts, proxyAttempt{
			Key:   key,
			Phase: "primary",
		})
	}

	lastKey := keys[len(keys)-1]
	attempts = append(attempts, proxyAttempt{
		Key:   lastKey,
		Phase: "retry",
	})

	return attempts, skipped, nil
}

func applyChatRouteOptions(route entity.ModelRoute, options chatCompletionOptions, metadata map[string]any) (entity.ModelRoute, error) {
	if options.RouteStrategy != "" {
		route.Model.RouteStrategy = options.RouteStrategy
		metadata["route_strategy"] = options.RouteStrategy
		metadata["route_strategy_override"] = options.RouteStrategy
	}

	if options.ProviderKeyID <= 0 {
		return route, nil
	}

	for _, key := range route.Keys {
		if key.ID == options.ProviderKeyID {
			route.Keys = []entity.ProviderKey{key}
			route.Model.RouteStrategy = "fixed"
			metadata["provider_key_override_id"] = key.ID
			metadata["provider_key_override_name"] = key.Name
			metadata["route_strategy"] = "fixed"
			return route, nil
		}
	}

	return route, errors.New("selected provider key is not available for the requested model")
}

func (h *Handler) currentKeyPenalty(keyID int64, now time.Time) (keyhealth.State, bool) {
	if h.keyHealth == nil {
		return keyhealth.State{}, false
	}

	return h.keyHealth.Current(keyID, now)
}

func (h *Handler) penalizeKey(keyID int64, duration time.Duration, reason string) {
	if h.keyHealth == nil {
		return
	}

	h.keyHealth.Penalize(keyID, duration, reason)
}

func (h *Handler) clearKeyPenalty(keyID int64) {
	if h.keyHealth == nil {
		return
	}

	h.keyHealth.Clear(keyID)
}

func (h *Handler) applyStatusPenalty(keyID int64, statusCode int) {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		h.penalizeKey(keyID, keyCooldownUnauthorized, "auth_error")
	case http.StatusTooManyRequests:
		h.penalizeKey(keyID, keyCooldownRateLimited, "rate_limited")
	default:
		if statusCode >= http.StatusInternalServerError {
			h.penalizeKey(keyID, keyCooldownTransient, "upstream_error")
		}
	}
}

func shouldFailoverStatus(statusCode int) bool {
	return statusCode == http.StatusUnauthorized ||
		statusCode == http.StatusForbidden ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}

func shouldRetryStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func nextAttemptDecision(nextPhase string) string {
	if nextPhase == "retry" {
		return "retry"
	}

	return "failover"
}

func isSupportedRouteStrategy(value string) bool {
	switch strings.TrimSpace(value) {
	case "fixed", "failover", "round_robin":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}

func metadataInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func intToString(value int) string {
	return strconv.Itoa(value)
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

func (h *Handler) writeDebugHeaders(headers http.Header, state chatLogState, options chatCompletionOptions) {
	if !options.EmitDebugHeaders {
		return
	}

	if state.providerKeyID > 0 {
		headers.Set("X-Wcs-Debug-Provider-Key-Id", int64ToString(state.providerKeyID))
	}
	if keyName, ok := state.metadata["provider_key_name"].(string); ok && strings.TrimSpace(keyName) != "" {
		headers.Set("X-Wcs-Debug-Provider-Key-Name", keyName)
	}
	if strategy, ok := state.metadata["route_strategy"].(string); ok && strings.TrimSpace(strategy) != "" {
		headers.Set("X-Wcs-Debug-Route-Strategy", strategy)
	}
	headers.Set("X-Wcs-Debug-Retry-Count", intToString(metadataInt(state.metadata["retry_count"])))
	headers.Set("X-Wcs-Debug-Failover-Count", intToString(metadataInt(state.metadata["failover_count"])))
}

func (h *Handler) writeRequestLog(ctx context.Context, startedAt time.Time, state chatLogState) {
	if h.logWriter == nil {
		return
	}

	metadataBytes, _ := json.Marshal(state.metadata)
	latencyMS := int(time.Since(startedAt).Milliseconds())
	if state.latencyMS > 0 {
		latencyMS = state.latencyMS
	}

	_ = h.logWriter.CreateRequestLog(ctx, entity.CreateRequestLogInput{
		TraceID:          state.traceID,
		RequestType:      state.requestType,
		ModelPublicName:  state.modelPublicName,
		UpstreamModel:    state.upstreamModel,
		ProviderID:       state.providerID,
		ProviderKeyID:    state.providerKeyID,
		ClientAPIKeyID:   state.clientAPIKeyID,
		ClientIP:         state.clientIP,
		RequestMethod:    state.requestMethod,
		RequestPath:      state.requestPath,
		HTTPStatus:       state.httpStatus,
		Success:          state.success,
		LatencyMS:        latencyMS,
		PromptTokens:     state.promptTokens,
		CompletionTokens: state.completionTokens,
		TotalTokens:      state.totalTokens,
		CostAmount:       state.costAmount,
		BillableAmount:   state.billableAmount,
		ErrorType:        state.errorType,
		ErrorMessage:     state.errorMessage,
		RequestPayload:   state.requestPayload,
		ResponsePayload:  state.responsePayload,
		Metadata:         metadataBytes,
	})

	if state.success && state.clientAPIKeyID > 0 && state.billableAmount > 0 {
		_ = h.logWriter.DeductTenantWalletUsage(ctx, state.clientAPIKeyID, state.billableAmount, "request "+state.traceID)
	}
}

func buildUpstreamURL(baseURL string, path string) string {
	if strings.TrimSpace(baseURL) == "" {
		return ""
	}

	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
	}

	baseSegments := splitPathSegments(parsed.Path)
	pathSegments := splitPathSegments(path)
	if len(pathSegments) == 0 {
		return parsed.String()
	}
	if pathHasSuffix(baseSegments, pathSegments) {
		return parsed.String()
	}
	if len(baseSegments) > 0 && len(pathSegments) > 0 && baseSegments[len(baseSegments)-1] == pathSegments[0] {
		pathSegments = pathSegments[1:]
	}
	segments := append(append([]string{}, baseSegments...), pathSegments...)
	parsed.Path = "/" + strings.Join(segments, "/")
	return parsed.String()
}

func splitPathSegments(value string) []string {
	trimmed := strings.Trim(strings.TrimSpace(value), "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func pathHasSuffix(baseSegments []string, suffixSegments []string) bool {
	if len(suffixSegments) == 0 || len(baseSegments) < len(suffixSegments) {
		return false
	}
	start := len(baseSegments) - len(suffixSegments)
	for index := range suffixSegments {
		if baseSegments[start+index] != suffixSegments[index] {
			return false
		}
	}
	return true
}

func copyResponseHeaders(destination http.Header, source http.Header) {
	for key, values := range source {
		if shouldSkipResponseHeader(key) {
			continue
		}

		for _, value := range values {
			destination.Add(key, value)
		}
	}
}

func shouldSkipResponseHeader(key string) bool {
	switch http.CanonicalHeaderKey(key) {
	case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Te",
		"Trailers", "Transfer-Encoding", "Upgrade", "Content-Length":
		return true
	default:
		return false
	}
}

func contentTypeOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func copyOptionalHeader(c *gin.Context, request *http.Request, header string) {
	if value := strings.TrimSpace(c.GetHeader(header)); value != "" {
		request.Header.Set(header, value)
	}
}

func ensureStreamOptionsIncludeUsage(payload map[string]any) {
	raw, ok := payload["stream_options"]
	if !ok {
		payload["stream_options"] = map[string]any{
			"include_usage": true,
		}
		return
	}

	options, ok := raw.(map[string]any)
	if !ok {
		payload["stream_options"] = map[string]any{
			"include_usage": true,
		}
		return
	}

	options["include_usage"] = true
	payload["stream_options"] = options
}

func isStreamingResponse(contentType string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(contentType)), "text/event-stream")
}

func extractStreamFlag(payload map[string]any) bool {
	raw, ok := payload["stream"]
	if !ok {
		return false
	}

	stream, ok := raw.(bool)
	return ok && stream
}

func payloadForStorage(body []byte, contentType string, streamed bool) json.RawMessage {
	if len(body) == 0 {
		return json.RawMessage(`{}`)
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) <= maxLoggedPayloadBytes && json.Valid(trimmed) && !streamed {
		return append(json.RawMessage{}, trimmed...)
	}

	preview := trimmed
	truncated := false
	if len(preview) > maxLoggedPayloadBytes {
		preview = preview[:maxLoggedPayloadBytes]
		truncated = true
	}

	return wrappedPayloadJSON(map[string]any{
		"content_type": contentType,
		"stream":       streamed,
		"truncated":    truncated,
		"preview":      string(preview),
	})
}

func streamUpstreamResponse(c *gin.Context, body io.Reader, metadata map[string]any, state *chatLogState, model entity.Model) ([]byte, bool, error) {
	buffer := make([]byte, 32*1024)
	flusher, _ := c.Writer.(http.Flusher)
	preview := bytes.NewBuffer(nil)
	parser := bytes.NewBuffer(nil)
	truncated := false

	for {
		n, err := body.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if _, writeErr := c.Writer.Write(chunk); writeErr != nil {
				return preview.Bytes(), truncated, writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}

			if preview.Len() < maxLoggedPayloadBytes {
				remaining := maxLoggedPayloadBytes - preview.Len()
				if n > remaining {
					preview.Write(chunk[:remaining])
					truncated = true
				} else {
					preview.Write(chunk)
				}
			} else {
				truncated = true
			}

			parser.Write(chunk)
			extractUsageFromStreamBuffer(parser, metadata, state, model)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return preview.Bytes(), truncated, nil
			}

			return preview.Bytes(), truncated, err
		}
	}
}

func extractUsageFromStreamBuffer(buffer *bytes.Buffer, metadata map[string]any, state *chatLogState, model entity.Model) {
	for {
		raw := buffer.Bytes()
		separatorIndex := bytes.Index(raw, []byte("\n\n"))
		if separatorIndex < 0 {
			return
		}

		event := make([]byte, separatorIndex)
		copy(event, raw[:separatorIndex])
		buffer.Next(separatorIndex + 2)

		lines := bytes.Split(event, []byte("\n"))
		for _, line := range lines {
			line = bytes.TrimSpace(line)
			if !bytes.HasPrefix(line, []byte("data:")) {
				continue
			}

			payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
			if bytes.Equal(payload, []byte("[DONE]")) || len(payload) == 0 {
				continue
			}

			applyUsage(metadata, state, payload, model)
		}
	}
}

func wrappedPayloadJSON(value map[string]any) json.RawMessage {
	body, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}

	return body
}

func applyUsage(metadata map[string]any, state *chatLogState, responseBody []byte, model entity.Model) {
	var payload map[string]any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return
	}

	usageRaw, ok := payload["usage"].(map[string]any)
	if !ok {
		return
	}

	state.promptTokens = intFromAny(usageRaw["prompt_tokens"])
	state.completionTokens = intFromAny(usageRaw["completion_tokens"])
	state.totalTokens = intFromAny(usageRaw["total_tokens"])
	state.costAmount, state.billableAmount = amountsForUsage(state.promptTokens, state.completionTokens, model)
	metadata["has_usage"] = true
	metadata["cost_amount"] = state.costAmount
	metadata["billable_amount"] = state.billableAmount
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	default:
		return 0
	}
}

func extractErrorMessage(responseBody []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return "upstream returned a non-success response"
	}

	errorValue, ok := payload["error"].(map[string]any)
	if !ok {
		return "upstream returned a non-success response"
	}

	message, ok := errorValue["message"].(string)
	if !ok || strings.TrimSpace(message) == "" {
		return "upstream returned a non-success response"
	}

	return strings.TrimSpace(message)
}

func clientKeyAllowsModel(clientKey entity.ClientAPIKey, modelPublicName string) bool {
	if len(clientKey.AllowedModels) == 0 {
		return true
	}

	target := strings.TrimSpace(modelPublicName)
	for _, allowed := range clientKey.AllowedModels {
		if strings.EqualFold(strings.TrimSpace(allowed), target) {
			return true
		}
	}

	return false
}

func clientKeyBudgetExceeded(clientKey entity.ClientAPIKey) (bool, string) {
	if clientKey.CostUsage == nil {
		return false, ""
	}
	if clientKey.CostUsage.IsDailyCostLimited {
		return true, "client daily cost budget exceeded"
	}
	if clientKey.CostUsage.IsMonthlyCostLimited {
		return true, "client monthly cost budget exceeded"
	}
	return false, ""
}

func amountsForUsage(promptTokens int, completionTokens int, model entity.Model) (float64, float64) {
	if promptTokens <= 0 && completionTokens <= 0 {
		return 0, 0
	}

	costAmount := (float64(promptTokens)*model.CostInputPer1M + float64(completionTokens)*model.CostOutputPer1M) / 1_000_000
	billableAmount := (float64(promptTokens)*model.SaleInputPer1M + float64(completionTokens)*model.SaleOutputPer1M) / 1_000_000
	return costAmount, billableAmount
}
