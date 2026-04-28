package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/middleware"
)

const anthropicVersionDefault = "2023-06-01"

func (h *Handler) Messages(c *gin.Context) {
	h.handleMessages(c, chatCompletionOptions{
		RequestType: "messages",
	})
}

func (h *Handler) AdminDebugMessages(c *gin.Context) {
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

	h.handleMessages(c, chatCompletionOptions{
		RequestBody:      request.Payload,
		RequestType:      "messages_debug",
		RouteStrategy:    routeStrategy,
		ProviderKeyID:    request.ProviderKeyID,
		IgnoreKeyHealth:  request.ProviderKeyID > 0,
		EmitDebugHeaders: true,
	})
}

func (h *Handler) handleMessages(c *gin.Context, options chatCompletionOptions) {
	startedAt := time.Now()
	logState := chatLogState{
		traceID:       strings.TrimSpace(c.Writer.Header().Get("X-Request-Id")),
		requestType:   firstNonEmpty(options.RequestType, "messages"),
		clientIP:      c.ClientIP(),
		requestMethod: c.Request.Method,
		requestPath:   c.FullPath(),
		metadata: map[string]any{
			"stream":        false,
			"provider_type": "anthropic",
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
			"type": "error",
			"error": map[string]any{
				"type":    errorType,
				"message": message,
			},
		})
		h.writeDebugHeaders(c.Writer.Header(), logState, options)
		c.JSON(statusCode, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    errorType,
				"message": message,
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

	payload, publicModel, err := parseAnthropicMessagesPayload(body)
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

	streamRequested := extractStreamFlag(payload)
	logState.metadata["stream"] = streamRequested

	route, err := h.store.ResolveModelRoute(c.Request.Context(), publicModel)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(http.StatusNotFound, "not_found", "model not found or unavailable")
			return
		}
		writeJSONError(http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	if !strings.EqualFold(strings.TrimSpace(route.Provider.ProviderType), "anthropic") {
		writeJSONError(http.StatusBadRequest, "invalid_provider_type", "selected model is not configured for anthropic messages")
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
	if _, exists := payload["max_tokens"]; !exists {
		writeJSONError(http.StatusBadRequest, "invalid_request", "max_tokens is required for anthropic messages")
		return
	}
	if _, exists := payload["temperature"]; !exists && route.Model.Temperature > 0 {
		payload["temperature"] = route.Model.Temperature
	}
	if clientKey, ok := middleware.ClientAPIKeyFromContext(c); ok && clientKey.UserID > 0 {
		requiredReserve := estimateAnthropicReserveAmount(payload, route.Model)
		logState.reservedAmount = requiredReserve
		logState.metadata["required_wallet_reserve"] = requiredReserve
		if requiredReserve > 0 && clientKey.UserWalletBalance < requiredReserve {
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

	version, betaHeaders := anthropicHeaders(route.Provider.ExtraConfig)
	failoverCount := 0
	retryCount := 0
	attemptMetadata := make([]map[string]any, 0, len(attempts))
	logState.metadata["routing_attempts"] = attemptMetadata
	logState.metadata["anthropic_version"] = version
	if len(betaHeaders) > 0 {
		logState.metadata["anthropic_beta"] = betaHeaders
	}

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
			buildUpstreamURL(route.Provider.BaseURL, "/v1/messages"),
			bytes.NewReader(rewrittenBody),
		)
		if err != nil {
			writeJSONError(http.StatusInternalServerError, "internal_error", "failed to create upstream request")
			return
		}
		upstreamRequest.Header.Set("x-api-key", attempt.Key.APIKey)
		upstreamRequest.Header.Set("anthropic-version", version)
		upstreamRequest.Header.Set("Content-Type", contentTypeOrDefault(c.GetHeader("Content-Type"), "application/json"))
		upstreamRequest.Header.Set("Accept", contentTypeOrDefault(c.GetHeader("Accept"), "application/json"))
		copyOptionalHeader(c, upstreamRequest, "anthropic-beta")
		for _, betaHeader := range betaHeaders {
			upstreamRequest.Header.Add("anthropic-beta", betaHeader)
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
			attemptRecord["error_type"] = firstNonEmpty(extractAnthropicErrorType(responseBody), "upstream_response_error")
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
			preview, truncated, streamErr := streamAnthropicUpstreamResponse(c, response.Body, logState.metadata, &logState, route.Model)
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
			logState.errorType = firstNonEmpty(extractAnthropicErrorType(responseBody), "upstream_response_error")
			logState.errorMessage = extractErrorMessage(responseBody)
		}
		applyAnthropicUsage(logState.metadata, &logState, responseBody, route.Model)

		if _, err := c.Writer.Write(responseBody); err != nil {
			logState.success = false
			logState.errorType = "client_write_error"
			logState.errorMessage = err.Error()
		}
		return
	}
}

func estimateAnthropicReserveAmount(payload map[string]any, model entity.Model) float64 {
	promptTokens := estimatePromptTokens(payload["messages"])
	completionTokens := extractPositiveInt(payload["max_tokens"])
	if completionTokens <= 0 {
		completionTokens = model.MaxTokens
	}
	if completionTokens <= 0 {
		completionTokens = 1024
	}
	_, billableAmount := amountsForUsage(promptTokens, completionTokens, model)
	return applyReservePolicy(billableAmount, model)
}

func parseAnthropicMessagesPayload(body []byte) (map[string]any, string, error) {
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

	rawMessages, ok := payload["messages"]
	if !ok {
		return nil, "", errors.New("messages is required")
	}
	messageList, ok := rawMessages.([]any)
	if !ok || len(messageList) == 0 {
		return nil, "", errors.New("messages must be a non-empty array")
	}

	return payload, strings.TrimSpace(modelName), nil
}

func anthropicHeaders(extraConfig json.RawMessage) (string, []string) {
	version := anthropicVersionDefault
	if len(bytes.TrimSpace(extraConfig)) == 0 {
		return version, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(extraConfig, &payload); err != nil {
		return version, nil
	}

	if configuredVersion, ok := payload["anthropic_version"].(string); ok && strings.TrimSpace(configuredVersion) != "" {
		version = strings.TrimSpace(configuredVersion)
	}

	betas := make([]string, 0)
	switch raw := payload["anthropic_beta"].(type) {
	case string:
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			betas = append(betas, trimmed)
		}
	case []any:
		for _, item := range raw {
			if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
				betas = append(betas, strings.TrimSpace(value))
			}
		}
	}

	return version, betas
}

func streamAnthropicUpstreamResponse(c *gin.Context, body io.Reader, metadata map[string]any, state *chatLogState, model entity.Model) ([]byte, bool, error) {
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
			extractAnthropicUsageFromStreamBuffer(parser, metadata, state, model)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return preview.Bytes(), truncated, nil
			}
			return preview.Bytes(), truncated, err
		}
	}
}

func extractAnthropicUsageFromStreamBuffer(buffer *bytes.Buffer, metadata map[string]any, state *chatLogState, model entity.Model) {
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
			if len(payload) == 0 {
				continue
			}

			applyAnthropicUsage(metadata, state, payload, model)
		}
	}
}

func applyAnthropicUsage(metadata map[string]any, state *chatLogState, responseBody []byte, model entity.Model) {
	var payload map[string]any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return
	}

	if message, ok := payload["message"].(map[string]any); ok {
		if usageRaw, ok := message["usage"].(map[string]any); ok {
			updateAnthropicUsage(metadata, state, usageRaw, model)
		}
	}

	if usageRaw, ok := payload["usage"].(map[string]any); ok {
		updateAnthropicUsage(metadata, state, usageRaw, model)
	}
}

func updateAnthropicUsage(metadata map[string]any, state *chatLogState, usageRaw map[string]any, model entity.Model) {
	inputTokens := intFromAny(usageRaw["input_tokens"])
	outputTokens := intFromAny(usageRaw["output_tokens"])
	if inputTokens > 0 {
		state.promptTokens = inputTokens
	}
	if outputTokens > 0 {
		state.completionTokens = outputTokens
	}
	state.totalTokens = state.promptTokens + state.completionTokens
	state.costAmount, state.billableAmount = amountsForUsage(state.promptTokens, state.completionTokens, model)
	metadata["has_usage"] = state.totalTokens > 0
	metadata["cost_amount"] = state.costAmount
	metadata["billable_amount"] = state.billableAmount
}

func extractAnthropicErrorType(responseBody []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return ""
	}
	errorValue, ok := payload["error"].(map[string]any)
	if !ok {
		return ""
	}
	errorType, ok := errorValue["type"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(errorType)
}
