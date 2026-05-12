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
	"google.golang.org/genai"

	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/middleware"
)

const defaultVertexAILocation = "us-central1"
const defaultVertexAIModel = "gemini-2.5-pro-preview-05-06"

type vertexAIConfig struct {
	ProjectID string
	Location  string
}

func parseVertexAIConfig(extraConfig json.RawMessage) vertexAIConfig {
	cfg := vertexAIConfig{Location: defaultVertexAILocation}
	if len(bytes.TrimSpace(extraConfig)) == 0 {
		return cfg
	}
	var m map[string]any
	if err := json.Unmarshal(extraConfig, &m); err != nil {
		return cfg
	}
	if v, ok := m["project_id"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.ProjectID = strings.TrimSpace(v)
	}
	if v, ok := m["location"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.Location = strings.TrimSpace(v)
	}
	return cfg
}

// handleWithVertexAISDK is the entry point for Vertex AI + ADC requests.
// It replaces the HTTP-proxy attempt loop entirely.
func (h *Handler) handleWithVertexAISDK(
	c *gin.Context,
	payload map[string]any,
	route entity.ModelRoute,
	logState *chatLogState,
	options geminiOptions,
) {
	timeout := 120 * time.Second
	if route.Model.TimeoutSeconds > 0 {
		timeout = time.Duration(route.Model.TimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	cfg := parseVertexAIConfig(route.Provider.ExtraConfig)
	if cfg.ProjectID == "" {
		h.writeVertexAIError(c, logState, http.StatusBadRequest, "invalid_config",
			"provider extra_config missing project_id for vertexai provider")
		return
	}

	client, err := h.getVertexAIClient(ctx, cfg.ProjectID, cfg.Location)
	if err != nil {
		h.writeVertexAIError(c, logState, http.StatusInternalServerError, "auth_error",
			"failed to initialize Vertex AI client: "+err.Error())
		return
	}

	// Parse contents via JSON round-trip (SDK types share JSON tags with REST API)
	contentsJSON, _ := json.Marshal(payload["contents"])
	var contents []*genai.Content
	if err := json.Unmarshal(contentsJSON, &contents); err != nil {
		h.writeVertexAIError(c, logState, http.StatusBadRequest, "invalid_request",
			"invalid contents format")
		return
	}

	genConfig := buildVertexAIGenConfig(payload)

	modelName := strings.TrimSpace(route.Model.UpstreamModel)
	if modelName == "" {
		modelName = defaultVertexAIModel
	}
	logState.metadata["vertex_project"] = cfg.ProjectID
	logState.metadata["vertex_location"] = cfg.Location

	if options.Stream {
		h.streamVertexAISDKResponse(c, ctx, client, modelName, contents, genConfig, logState, route.Model)
	} else {
		h.nonStreamVertexAISDKResponse(c, ctx, client, modelName, contents, genConfig, logState, route.Model)
	}
}

func (h *Handler) nonStreamVertexAISDKResponse(
	c *gin.Context,
	ctx context.Context,
	client *genai.Client,
	modelName string,
	contents []*genai.Content,
	genConfig *genai.GenerateContentConfig,
	logState *chatLogState,
	model entity.Model,
) {
	resp, err := client.Models.GenerateContent(ctx, modelName, contents, genConfig)
	if err != nil {
		code := vertexAIErrorCode(err)
		h.writeVertexAIError(c, logState, code, "upstream_error", err.Error())
		return
	}

	applyVertexAIUsage(logState, resp.UsageMetadata, model)
	logState.httpStatus = http.StatusOK
	logState.success = true

	// Marshal SDK response — JSON tags match Gemini REST API format
	responseJSON, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		h.writeVertexAIError(c, logState, http.StatusInternalServerError, "internal_error",
			"failed to encode response")
		return
	}

	logState.responsePayload = payloadForStorage(responseJSON, "application/json", false)
	c.Data(http.StatusOK, "application/json; charset=utf-8", responseJSON)
}

func (h *Handler) streamVertexAISDKResponse(
	c *gin.Context,
	ctx context.Context,
	client *genai.Client,
	modelName string,
	contents []*genai.Content,
	genConfig *genai.GenerateContentConfig,
	logState *chatLogState,
	model entity.Model,
) {
	flusher, canFlush := c.Writer.(http.Flusher)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	logState.httpStatus = http.StatusOK
	logState.success = true
	logState.metadata["stream_response"] = true

	preview := bytes.NewBuffer(nil)
	truncated := false
	wroteAny := false

	writeSSEError := func(message string) {
		errEvent := wrappedPayloadJSON(map[string]any{
			"error": map[string]any{"type": "stream_error", "message": message},
		})
		line := append([]byte("data: "), errEvent...)
		line = append(line, '\n', '\n')
		_, _ = c.Writer.Write(line)
		if canFlush {
			flusher.Flush()
		}
	}

	for chunk, err := range client.Models.GenerateContentStream(ctx, modelName, contents, genConfig) {
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logState.success = false
				logState.errorType = "stream_error"
				logState.errorMessage = err.Error()
				writeSSEError(err.Error())
			}
			break
		}

		if chunk.UsageMetadata != nil {
			applyVertexAIUsage(logState, chunk.UsageMetadata, model)
		}

		chunkJSON, marshalErr := json.Marshal(chunk)
		if marshalErr != nil {
			continue
		}

		event := append([]byte("data: "), chunkJSON...)
		event = append(event, '\n', '\n')
		if _, writeErr := c.Writer.Write(event); writeErr != nil {
			logState.success = false
			logState.errorType = "client_write_error"
			logState.errorMessage = writeErr.Error()
			break
		}
		wroteAny = true
		if canFlush {
			flusher.Flush()
		}

		if preview.Len() < maxLoggedPayloadBytes {
			remaining := maxLoggedPayloadBytes - preview.Len()
			if len(chunkJSON) > remaining {
				preview.Write(chunkJSON[:remaining])
				truncated = true
			} else {
				preview.Write(chunkJSON)
			}
		} else {
			truncated = true
		}
	}

	if !wroteAny && logState.errorType == "" {
		writeSSEError("no content returned from upstream")
		logState.success = false
		logState.errorType = "empty_response"
		logState.errorMessage = "no content returned from upstream"
	}

	logState.responsePayload = wrappedPayloadJSON(map[string]any{
		"content_type": "text/event-stream",
		"stream":       true,
		"truncated":    truncated,
		"preview":      preview.String(),
	})
}

func (h *Handler) writeVertexAIError(
	c *gin.Context,
	logState *chatLogState,
	statusCode int,
	errorType string,
	message string,
) {
	logState.httpStatus = statusCode
	logState.errorType = errorType
	logState.errorMessage = message
	logState.responsePayload = wrappedPayloadJSON(map[string]any{
		"error": map[string]any{"message": message, "type": errorType},
	})
	if clientKey, ok := middleware.ClientAPIKeyFromContext(c); ok {
		logState.clientAPIKeyID = clientKey.ID
	}
	c.JSON(statusCode, gin.H{"error": gin.H{"message": message, "type": errorType}})
}

// buildVertexAIGenConfig builds a GenerateContentConfig from the incoming Gemini JSON payload.
// JSON tags in GenerateContentConfig match the Gemini REST API generationConfig fields,
// so we can directly unmarshal generationConfig into it.
func buildVertexAIGenConfig(payload map[string]any) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{}

	if gcRaw, ok := payload["generationConfig"]; ok {
		gcJSON, err := json.Marshal(gcRaw)
		if err == nil {
			_ = json.Unmarshal(gcJSON, config)
		}
	}

	if siRaw, ok := payload["systemInstruction"]; ok {
		siJSON, err := json.Marshal(siRaw)
		if err == nil {
			var si genai.Content
			if err := json.Unmarshal(siJSON, &si); err == nil && len(si.Parts) > 0 {
				config.SystemInstruction = &si
			}
		}
	}

	if toolsRaw, ok := payload["tools"]; ok {
		toolsJSON, err := json.Marshal(toolsRaw)
		if err == nil {
			var tools []*genai.Tool
			if err := json.Unmarshal(toolsJSON, &tools); err == nil {
				config.Tools = tools
			}
		}
	}

	if tcRaw, ok := payload["toolConfig"]; ok {
		tcJSON, err := json.Marshal(tcRaw)
		if err == nil {
			var tc genai.ToolConfig
			if err := json.Unmarshal(tcJSON, &tc); err == nil {
				config.ToolConfig = &tc
			}
		}
	}

	return config
}

func applyVertexAIUsage(logState *chatLogState, usage *genai.GenerateContentResponseUsageMetadata, model entity.Model) {
	if usage == nil {
		return
	}
	logState.promptTokens = int(usage.PromptTokenCount)
	logState.completionTokens = int(usage.CandidatesTokenCount)
	if usage.TotalTokenCount > 0 {
		logState.totalTokens = int(usage.TotalTokenCount)
	} else {
		logState.totalTokens = logState.promptTokens + logState.completionTokens
	}
	logState.costAmount, logState.billableAmount = amountsForUsage(logState.promptTokens, logState.completionTokens, model)
}

func vertexAIErrorCode(err error) int {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "quota") || strings.Contains(msg, "rate") || strings.Contains(msg, "429"):
		return http.StatusTooManyRequests
	case strings.Contains(msg, "permission") || strings.Contains(msg, "403") || strings.Contains(msg, "unauthorized"):
		return http.StatusForbidden
	case strings.Contains(msg, "not found") || strings.Contains(msg, "404"):
		return http.StatusNotFound
	case strings.Contains(msg, "invalid") || strings.Contains(msg, "400"):
		return http.StatusBadRequest
	default:
		return http.StatusBadGateway
	}
}

func vertexAIProviderType(providerType string) bool {
	return strings.EqualFold(strings.TrimSpace(providerType), "vertexai")
}

func estimateVertexAIReserveAmount(payload map[string]any, model entity.Model) float64 {
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

// logVertexAIAttempt sets common metadata for Vertex AI calls (no key rotation).
func logVertexAIAttempt(logState *chatLogState, projectID, location string) {
	logState.metadata["routing_mode"] = "vertex_ai_adc"
	logState.metadata["vertex_project"] = projectID
	logState.metadata["vertex_location"] = location
	logState.metadata["failover_count"] = 0
	logState.metadata["retry_count"] = 0
}

// writeVertexAIRequestLog records the log for a Vertex AI request.
// For Vertex AI, debit still applies the same way as for other providers.
func (h *Handler) writeVertexAIRequestLog(ctx context.Context, startedAt time.Time, logState *chatLogState) {
	h.writeRequestLog(ctx, startedAt, *logState)
}

// GeminiGenerateContent and GeminiStreamGenerateContent route to Vertex AI
// when the resolved provider has type "vertexai", otherwise use the HTTP proxy path.

// Sentinel used to signal Vertex AI path inside handleGeminiGenerateContent.
const providerTypeVertexAI = "vertexai"

// describeVertexAI returns a short description string for logging.
func describeVertexAI(cfg vertexAIConfig) string {
	return fmt.Sprintf("vertexai/%s/%s", cfg.ProjectID, cfg.Location)
}
