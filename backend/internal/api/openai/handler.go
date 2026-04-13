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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/repository"
)

const maxLoggedPayloadBytes = 64 * 1024

type Handler struct {
	store      repository.PublicModelStore
	logWriter  repository.RequestLogWriter
	httpClient *http.Client
	counters   sync.Map
}

type chatLogState struct {
	traceID          string
	requestType      string
	modelPublicName  string
	upstreamModel    string
	providerID       int64
	providerKeyID    int64
	clientIP         string
	requestMethod    string
	requestPath      string
	httpStatus       int
	success          bool
	latencyMS        int
	promptTokens     int
	completionTokens int
	totalTokens      int
	estimatedCost    float64
	errorType        string
	errorMessage     string
	requestPayload   json.RawMessage
	responsePayload  json.RawMessage
	metadata         map[string]any
}

func NewHandler(
	store repository.PublicModelStore,
	logWriter repository.RequestLogWriter,
	httpClient *http.Client,
) *Handler {
	client := httpClient
	if client == nil {
		client = &http.Client{}
	}

	return &Handler{
		store:      store,
		logWriter:  logWriter,
		httpClient: client,
	}
}

func (h *Handler) ListModels(c *gin.Context) {
	items := make([]gin.H, 0)
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
	startedAt := time.Now()
	logState := chatLogState{
		traceID:       strings.TrimSpace(c.Writer.Header().Get("X-Request-Id")),
		requestType:   "chat_completions",
		clientIP:      c.ClientIP(),
		requestMethod: c.Request.Method,
		requestPath:   c.FullPath(),
		metadata: map[string]any{
			"stream": false,
		},
	}

	defer func() {
		h.writeRequestLog(c.Request.Context(), startedAt, logState)
	}()

	if h.store == nil {
		logState.httpStatus = http.StatusServiceUnavailable
		logState.errorType = "service_unavailable"
		logState.errorMessage = "model store is not configured"
		logState.responsePayload = wrappedPayloadJSON(map[string]any{
			"error": map[string]any{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logState.httpStatus = http.StatusBadRequest
		logState.errorType = "invalid_request"
		logState.errorMessage = "failed to read request body"
		logState.responsePayload = wrappedPayloadJSON(map[string]any{
			"error": map[string]any{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		return
	}

	logState.requestPayload = payloadForStorage(body, "application/json", false)

	payload, publicModel, err := parseChatCompletionPayload(body)
	if err != nil {
		logState.httpStatus = http.StatusBadRequest
		logState.errorType = "invalid_request"
		logState.errorMessage = err.Error()
		logState.responsePayload = wrappedPayloadJSON(map[string]any{
			"error": map[string]any{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		return
	}
	logState.modelPublicName = publicModel

	streamRequested := extractStreamFlag(payload)
	logState.metadata["stream"] = streamRequested
	if streamRequested {
		ensureStreamOptionsIncludeUsage(payload)
		logState.metadata["stream_include_usage"] = true
	}

	route, err := h.store.ResolveModelRoute(c.Request.Context(), publicModel)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logState.httpStatus = http.StatusNotFound
			logState.errorType = "not_found"
			logState.errorMessage = "model not found or unavailable"
			logState.responsePayload = wrappedPayloadJSON(map[string]any{
				"error": map[string]any{
					"message": logState.errorMessage,
					"type":    logState.errorType,
				},
			})
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": logState.errorMessage,
					"type":    logState.errorType,
				},
			})
			return
		}

		logState.httpStatus = http.StatusInternalServerError
		logState.errorType = "database_error"
		logState.errorMessage = err.Error()
		logState.responsePayload = wrappedPayloadJSON(map[string]any{
			"error": map[string]any{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		return
	}

	logState.upstreamModel = route.Model.UpstreamModel
	logState.providerID = route.Provider.ID
	logState.metadata["provider_slug"] = route.Provider.Slug
	logState.metadata["provider_type"] = route.Provider.ProviderType
	logState.metadata["route_strategy"] = route.Model.RouteStrategy

	selectedKey, err := h.selectKey(route)
	if err != nil {
		logState.httpStatus = http.StatusServiceUnavailable
		logState.errorType = "routing_error"
		logState.errorMessage = err.Error()
		logState.responsePayload = wrappedPayloadJSON(map[string]any{
			"error": map[string]any{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		return
	}

	logState.providerKeyID = selectedKey.ID
	logState.metadata["provider_key_name"] = selectedKey.Name

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
		logState.httpStatus = http.StatusInternalServerError
		logState.errorType = "internal_error"
		logState.errorMessage = "failed to encode upstream payload"
		logState.responsePayload = wrappedPayloadJSON(map[string]any{
			"error": map[string]any{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		return
	}

	timeout := 120 * time.Second
	if route.Model.TimeoutSeconds > 0 {
		timeout = time.Duration(route.Model.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	upstreamRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		buildUpstreamURL(route.Provider.BaseURL, "/chat/completions"),
		bytes.NewReader(rewrittenBody),
	)
	if err != nil {
		logState.httpStatus = http.StatusInternalServerError
		logState.errorType = "internal_error"
		logState.errorMessage = "failed to create upstream request"
		logState.responsePayload = wrappedPayloadJSON(map[string]any{
			"error": map[string]any{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		return
	}

	upstreamRequest.Header.Set("Authorization", "Bearer "+selectedKey.APIKey)
	upstreamRequest.Header.Set("Content-Type", contentTypeOrDefault(c.GetHeader("Content-Type"), "application/json"))
	upstreamRequest.Header.Set("Accept", contentTypeOrDefault(c.GetHeader("Accept"), "application/json"))
	copyOptionalHeader(c, upstreamRequest, "OpenAI-Beta")

	response, err := h.httpClient.Do(upstreamRequest)
	if err != nil {
		logState.httpStatus = http.StatusBadGateway
		logState.errorType = "upstream_error"
		logState.errorMessage = err.Error()
		logState.responsePayload = wrappedPayloadJSON(map[string]any{
			"error": map[string]any{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"message": logState.errorMessage,
				"type":    logState.errorType,
			},
		})
		return
	}
	defer response.Body.Close()

	responseContentType := response.Header.Get("Content-Type")
	logState.httpStatus = response.StatusCode
	logState.success = response.StatusCode >= 200 && response.StatusCode < 300
	logState.metadata["response_content_type"] = responseContentType

	copyResponseHeaders(c.Writer.Header(), response.Header)
	c.Status(response.StatusCode)

	if isStreamingResponse(responseContentType) {
		logState.metadata["stream_response"] = true
		preview, truncated, streamErr := streamUpstreamResponse(c, response.Body, logState.metadata, &logState)
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
		c.Status(http.StatusBadGateway)
		_, _ = c.Writer.Write(logState.responsePayload)
		return
	}

	logState.responsePayload = payloadForStorage(responseBody, responseContentType, streamRequested)
	if !logState.success && logState.errorType == "" {
		logState.errorType = "upstream_response_error"
		logState.errorMessage = extractErrorMessage(responseBody)
	}

	applyUsage(logState.metadata, &logState, responseBody)

	if _, err := c.Writer.Write(responseBody); err != nil {
		logState.success = false
		logState.errorType = "client_write_error"
		logState.errorMessage = err.Error()
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

func (h *Handler) selectKey(route entity.ModelRoute) (entity.ProviderKey, error) {
	if len(route.Keys) == 0 {
		return entity.ProviderKey{}, errors.New("no active provider key is available for the requested model")
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
		return keys[index], nil
	}

	return keys[0], nil
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
		ClientIP:         state.clientIP,
		RequestMethod:    state.requestMethod,
		RequestPath:      state.requestPath,
		HTTPStatus:       state.httpStatus,
		Success:          state.success,
		LatencyMS:        latencyMS,
		PromptTokens:     state.promptTokens,
		CompletionTokens: state.completionTokens,
		TotalTokens:      state.totalTokens,
		EstimatedCost:    state.estimatedCost,
		ErrorType:        state.errorType,
		ErrorMessage:     state.errorMessage,
		RequestPayload:   state.requestPayload,
		ResponsePayload:  state.responsePayload,
		Metadata:         metadataBytes,
	})
}

func buildUpstreamURL(baseURL string, path string) string {
	if strings.TrimSpace(baseURL) == "" {
		return ""
	}

	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(path, "/")
	return parsed.String()
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

func streamUpstreamResponse(c *gin.Context, body io.Reader, metadata map[string]any, state *chatLogState) ([]byte, bool, error) {
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
			extractUsageFromStreamBuffer(parser, metadata, state)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return preview.Bytes(), truncated, nil
			}

			return preview.Bytes(), truncated, err
		}
	}
}

func extractUsageFromStreamBuffer(buffer *bytes.Buffer, metadata map[string]any, state *chatLogState) {
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

			applyUsage(metadata, state, payload)
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

func applyUsage(metadata map[string]any, state *chatLogState, responseBody []byte) {
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
	metadata["has_usage"] = true
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
