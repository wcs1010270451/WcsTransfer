package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/middleware"
)

const geminiAPIVersionDefault = "v1beta"

func (h *Handler) GeminiGenerateContent(c *gin.Context) {
	h.handleGeminiGenerateContent(c, geminiOptions{
		RequestType: "gemini_generate_content",
		Method:      "generateContent",
		Stream:      false,
	})
}

func (h *Handler) GeminiStreamGenerateContent(c *gin.Context) {
	h.handleGeminiGenerateContent(c, geminiOptions{
		RequestType: "gemini_stream_generate_content",
		Method:      "streamGenerateContent",
		Stream:      true,
	})
}

func (h *Handler) AdminDebugGeminiGenerateContent(c *gin.Context) {
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

	h.handleGeminiGenerateContent(c, geminiOptions{
		RequestBody:      request.Payload,
		RequestType:      "gemini_generate_content_debug",
		Method:           "generateContent",
		Stream:           false,
		RouteStrategy:    routeStrategy,
		ProviderKeyID:    request.ProviderKeyID,
		IgnoreKeyHealth:  request.ProviderKeyID > 0,
		EmitDebugHeaders: true,
	})
}

func (h *Handler) AdminDebugGeminiStreamGenerateContent(c *gin.Context) {
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

	h.handleGeminiGenerateContent(c, geminiOptions{
		RequestBody:      request.Payload,
		RequestType:      "gemini_stream_generate_content_debug",
		Method:           "streamGenerateContent",
		Stream:           true,
		RouteStrategy:    routeStrategy,
		ProviderKeyID:    request.ProviderKeyID,
		IgnoreKeyHealth:  request.ProviderKeyID > 0,
		EmitDebugHeaders: true,
	})
}

type geminiOptions struct {
	RequestBody      []byte
	RequestType      string
	Method           string
	Stream           bool
	RouteStrategy    string
	ProviderKeyID    int64
	IgnoreKeyHealth  bool
	EmitDebugHeaders bool
}

func (h *Handler) handleGeminiGenerateContent(c *gin.Context, options geminiOptions) {
	startedAt := time.Now()
	logState := chatLogState{
		traceID:       strings.TrimSpace(c.Writer.Header().Get("X-Request-Id")),
		requestType:   firstNonEmpty(options.RequestType, "gemini_generate_content"),
		clientIP:      c.ClientIP(),
		requestMethod: c.Request.Method,
		requestPath:   c.FullPath(),
		metadata: map[string]any{
			"stream":        options.Stream,
			"provider_type": "gemini",
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
		h.writeDebugHeaders(c.Writer.Header(), logState, chatCompletionOptions{EmitDebugHeaders: options.EmitDebugHeaders})
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

	payload, publicModel, err := parseGeminiGenerateContentPayload(body)
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
	if !strings.EqualFold(strings.TrimSpace(route.Provider.ProviderType), "gemini") {
		writeJSONError(http.StatusBadRequest, "invalid_provider_type", "selected model is not configured for gemini native api")
		return
	}

	logState.upstreamModel = route.Model.UpstreamModel
	logState.providerID = route.Provider.ID
	logState.metadata["provider_slug"] = route.Provider.Slug
	logState.metadata["route_strategy"] = route.Model.RouteStrategy
	logState.metadata["cost_input_per_1m"] = route.Model.CostInputPer1M
	logState.metadata["cost_output_per_1m"] = route.Model.CostOutputPer1M
	logState.metadata["sale_input_per_1m"] = route.Model.SaleInputPer1M
	logState.metadata["sale_output_per_1m"] = route.Model.SaleOutputPer1M
	logState.metadata["reserve_multiplier"] = route.Model.ReserveMultiplier
	logState.metadata["reserve_min_amount"] = route.Model.ReserveMinAmount

	route, err = applyChatRouteOptions(route, chatCompletionOptions{
		RouteStrategy:   options.RouteStrategy,
		ProviderKeyID:   options.ProviderKeyID,
		IgnoreKeyHealth: options.IgnoreKeyHealth,
	}, logState.metadata)
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

	delete(payload, "model")
	applyGeminiGenerationDefaults(payload, route.Model)
	if clientKey, ok := middleware.ClientAPIKeyFromContext(c); ok && clientKey.TenantID > 0 {
		requiredReserve := estimateGeminiReserveAmount(payload, route.Model)
		logState.reservedAmount = requiredReserve
		logState.metadata["required_wallet_reserve"] = requiredReserve
		if requiredReserve > 0 && clientKey.TenantWalletBalance < requiredReserve {
			writeJSONError(
				http.StatusPaymentRequired,
				"wallet_reserve_insufficient",
				fmt.Sprintf("tenant wallet balance is below the required reserve %.4f USD", requiredReserve),
			)
			return
		}
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

	apiVersion := geminiAPIVersion(route.Provider.ExtraConfig)
	logState.metadata["gemini_api_version"] = apiVersion

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
			buildGeminiUpstreamURL(route.Provider.BaseURL, apiVersion, route.Model.UpstreamModel, options.Method, options.Stream),
			bytes.NewReader(rewrittenBody),
		)
		if err != nil {
			writeJSONError(http.StatusInternalServerError, "internal_error", "failed to create upstream request")
			return
		}
		upstreamRequest.Header.Set("x-goog-api-key", attempt.Key.APIKey)
		upstreamRequest.Header.Set("Content-Type", contentTypeOrDefault(c.GetHeader("Content-Type"), "application/json"))
		if options.Stream {
			upstreamRequest.Header.Set("Accept", contentTypeOrDefault(c.GetHeader("Accept"), "text/event-stream"))
		} else {
			upstreamRequest.Header.Set("Accept", contentTypeOrDefault(c.GetHeader("Accept"), "application/json"))
		}

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

		h.writeDebugHeaders(c.Writer.Header(), logState, chatCompletionOptions{EmitDebugHeaders: options.EmitDebugHeaders})
		copyResponseHeaders(c.Writer.Header(), response.Header)
		c.Status(response.StatusCode)

		if isStreamingResponse(responseContentType) {
			defer response.Body.Close()
			logState.metadata["stream_response"] = true
			preview, truncated, streamErr := streamGeminiUpstreamResponse(c, response.Body, logState.metadata, &logState, route.Model)
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
			writeJSONError(http.StatusBadGateway, "upstream_read_error", readErr.Error())
			return
		}

		logState.responsePayload = payloadForStorage(responseBody, responseContentType, false)
		if !logState.success && logState.errorType == "" {
			logState.errorType = "upstream_response_error"
			logState.errorMessage = extractErrorMessage(responseBody)
		}
		applyGeminiUsage(logState.metadata, &logState, responseBody, route.Model)

		if _, err := c.Writer.Write(responseBody); err != nil {
			logState.success = false
			logState.errorType = "client_write_error"
			logState.errorMessage = err.Error()
		}
		return
	}
}

func parseGeminiGenerateContentPayload(body []byte) (map[string]any, string, error) {
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
	rawContents, ok := payload["contents"]
	if !ok {
		return nil, "", errors.New("contents is required")
	}
	switch typed := rawContents.(type) {
	case []any:
		if len(typed) == 0 {
			return nil, "", errors.New("contents must be a non-empty array")
		}
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil, "", errors.New("contents must be a non-empty string or array")
		}
	default:
		return nil, "", errors.New("contents must be a non-empty string or array")
	}

	return payload, strings.TrimSpace(modelName), nil
}

func applyGeminiGenerationDefaults(payload map[string]any, model entity.Model) {
	config := ensureGeminiGenerationConfig(payload)
	if _, exists := config["temperature"]; !exists && model.Temperature > 0 {
		config["temperature"] = model.Temperature
	}
	if _, exists := config["maxOutputTokens"]; !exists && model.MaxTokens > 0 {
		config["maxOutputTokens"] = model.MaxTokens
	}
	payload["generationConfig"] = config
}

func ensureGeminiGenerationConfig(payload map[string]any) map[string]any {
	raw, ok := payload["generationConfig"]
	if !ok {
		config := map[string]any{}
		payload["generationConfig"] = config
		return config
	}
	config, ok := raw.(map[string]any)
	if !ok {
		config = map[string]any{}
		payload["generationConfig"] = config
		return config
	}
	return config
}

func estimateGeminiReserveAmount(payload map[string]any, model entity.Model) float64 {
	promptTokens := estimatePromptTokens(payload["contents"])
	completionTokens := 0
	if config, ok := payload["generationConfig"].(map[string]any); ok {
		completionTokens = extractPositiveInt(config["maxOutputTokens"])
	}
	if completionTokens <= 0 {
		completionTokens = model.MaxTokens
	}
	if completionTokens <= 0 {
		completionTokens = 1024
	}
	_, billableAmount := amountsForUsage(promptTokens, completionTokens, model)
	return applyReservePolicy(billableAmount, model)
}

func geminiAPIVersion(extraConfig json.RawMessage) string {
	version := geminiAPIVersionDefault
	if len(bytes.TrimSpace(extraConfig)) == 0 {
		return version
	}

	var payload map[string]any
	if err := json.Unmarshal(extraConfig, &payload); err != nil {
		return version
	}
	if configuredVersion, ok := payload["gemini_api_version"].(string); ok && strings.TrimSpace(configuredVersion) != "" {
		return strings.Trim(strings.TrimSpace(configuredVersion), "/")
	}
	return version
}

func buildGeminiUpstreamURL(baseURL string, apiVersion string, model string, method string, stream bool) string {
	upstreamURL := buildUpstreamURL(baseURL, "/"+strings.Trim(apiVersion, "/")+"/models/"+strings.TrimSpace(model)+":"+strings.TrimSpace(method))
	if !stream {
		return upstreamURL
	}

	parsed, err := url.Parse(upstreamURL)
	if err != nil {
		if strings.Contains(upstreamURL, "?") {
			return upstreamURL + "&alt=sse"
		}
		return upstreamURL + "?alt=sse"
	}
	query := parsed.Query()
	query.Set("alt", "sse")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func streamGeminiUpstreamResponse(c *gin.Context, body io.Reader, metadata map[string]any, state *chatLogState, model entity.Model) ([]byte, bool, error) {
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
			extractGeminiUsageFromStreamBuffer(parser, metadata, state, model)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return preview.Bytes(), truncated, nil
			}
			return preview.Bytes(), truncated, err
		}
	}
}

func extractGeminiUsageFromStreamBuffer(buffer *bytes.Buffer, metadata map[string]any, state *chatLogState, model entity.Model) {
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
			if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
				continue
			}

			applyGeminiUsage(metadata, state, payload, model)
		}
	}
}

func applyGeminiUsage(metadata map[string]any, state *chatLogState, responseBody []byte, model entity.Model) {
	var payload map[string]any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return
	}

	usageRaw, ok := payload["usageMetadata"].(map[string]any)
	if !ok {
		return
	}

	state.promptTokens = intFromAny(usageRaw["promptTokenCount"])
	state.completionTokens = intFromAny(usageRaw["candidatesTokenCount"])
	totalTokens := intFromAny(usageRaw["totalTokenCount"])
	if totalTokens > 0 {
		state.totalTokens = totalTokens
	} else {
		state.totalTokens = state.promptTokens + state.completionTokens
	}
	state.costAmount, state.billableAmount = amountsForUsage(state.promptTokens, state.completionTokens, model)
	metadata["has_usage"] = state.totalTokens > 0
	metadata["cost_amount"] = state.costAmount
	metadata["billable_amount"] = state.billableAmount
}
