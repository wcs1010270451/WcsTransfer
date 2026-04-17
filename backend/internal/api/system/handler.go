package system

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"wcstransfer/backend/internal/buildinfo"
	"wcstransfer/backend/internal/config"
	"wcstransfer/backend/internal/platform"
)

type Handler struct {
	config       config.Config
	dependencies *platform.Dependencies
}

func NewHandler(cfg config.Config, deps *platform.Dependencies) *Handler {
	return &Handler{
		config:       cfg,
		dependencies: deps,
	}
}

func (h *Handler) Healthz(c *gin.Context) {
	checks := map[string]platform.CheckResult{}
	if h.dependencies != nil {
		checks = h.dependencies.Health(c.Request.Context())
	}

	status := "ok"
	code := http.StatusOK
	for _, check := range checks {
		if check.Status == "down" {
			status = "degraded"
			code = http.StatusServiceUnavailable
			break
		}
	}

	c.JSON(code, gin.H{
		"status":       status,
		"service":      h.config.AppName,
		"environment":  h.config.Env,
		"dependencies": checks,
		"time":         time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":     h.config.AppName,
		"environment": h.config.Env,
		"version":     buildinfo.Version,
		"commit":      buildinfo.Commit,
		"build_time":  buildinfo.BuildTime,
	})
}

func (h *Handler) OpenAPI(c *gin.Context) {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := c.GetHeader("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	serverURL := scheme + "://" + c.Request.Host

	clientBearer := gin.H{
		"type":         "http",
		"scheme":       "bearer",
		"bearerFormat": "API Key",
		"description":  "Client API key used for /v1 routes.",
	}
	adminBearer := gin.H{
		"type":         "http",
		"scheme":       "bearer",
		"bearerFormat": "Admin Token",
		"description":  "Admin bearer token used for /admin routes.",
	}

	errorSchema := gin.H{
		"type": "object",
		"properties": gin.H{
			"error": gin.H{
				"type": "object",
				"properties": gin.H{
					"type":    gin.H{"type": "string"},
					"message": gin.H{"type": "string"},
				},
			},
		},
	}
	providerSchema := gin.H{
		"type": "object",
		"properties": gin.H{
			"id":            gin.H{"type": "integer"},
			"name":          gin.H{"type": "string"},
			"slug":          gin.H{"type": "string"},
			"provider_type": gin.H{"type": "string"},
			"base_url":      gin.H{"type": "string"},
			"status":        gin.H{"type": "string"},
			"description":   gin.H{"type": "string"},
			"extra_config":  gin.H{"type": "object"},
			"created_at":    gin.H{"type": "string", "format": "date-time"},
			"updated_at":    gin.H{"type": "string", "format": "date-time"},
		},
	}
	clientKeySchema := gin.H{
		"type": "object",
		"properties": gin.H{
			"id":                   gin.H{"type": "integer"},
			"name":                 gin.H{"type": "string"},
			"masked_key":           gin.H{"type": "string"},
			"status":               gin.H{"type": "string"},
			"description":          gin.H{"type": "string"},
			"rpm_limit":            gin.H{"type": "integer"},
			"daily_request_limit":  gin.H{"type": "integer"},
			"daily_token_limit":    gin.H{"type": "integer"},
			"daily_cost_limit":     gin.H{"type": "number"},
			"monthly_cost_limit":   gin.H{"type": "number"},
			"warning_threshold":    gin.H{"type": "number"},
			"allowed_model_ids":    gin.H{"type": "array", "items": gin.H{"type": "integer"}},
			"current_rpm":          gin.H{"type": "integer"},
			"daily_request_usage":  gin.H{"type": "integer"},
			"daily_token_usage":    gin.H{"type": "integer"},
			"daily_cost_usage":     gin.H{"type": "number"},
			"monthly_cost_usage":   gin.H{"type": "number"},
			"quota_health_status":  gin.H{"type": "string"},
			"quota_health_message": gin.H{"type": "string"},
			"expires_at":           gin.H{"type": "string", "format": "date-time"},
			"created_at":           gin.H{"type": "string", "format": "date-time"},
			"updated_at":           gin.H{"type": "string", "format": "date-time"},
		},
	}
	providerKeySchema := gin.H{
		"type": "object",
		"properties": gin.H{
			"id":              gin.H{"type": "integer"},
			"provider_id":     gin.H{"type": "integer"},
			"provider_name":   gin.H{"type": "string"},
			"name":            gin.H{"type": "string"},
			"masked_key":      gin.H{"type": "string"},
			"status":          gin.H{"type": "string"},
			"weight":          gin.H{"type": "integer"},
			"priority":        gin.H{"type": "integer"},
			"rpm_limit":       gin.H{"type": "integer"},
			"tpm_limit":       gin.H{"type": "integer"},
			"health_status":   gin.H{"type": "string"},
			"cooldown_reason": gin.H{"type": "string"},
			"cooldown_until":  gin.H{"type": "string", "format": "date-time"},
			"created_at":      gin.H{"type": "string", "format": "date-time"},
			"updated_at":      gin.H{"type": "string", "format": "date-time"},
		},
	}
	modelSchema := gin.H{
		"type": "object",
		"properties": gin.H{
			"id":                 gin.H{"type": "integer"},
			"public_name":        gin.H{"type": "string"},
			"provider_id":        gin.H{"type": "integer"},
			"provider_name":      gin.H{"type": "string"},
			"upstream_model":     gin.H{"type": "string"},
			"route_strategy":     gin.H{"type": "string"},
			"is_enabled":         gin.H{"type": "boolean"},
			"max_tokens":         gin.H{"type": "integer"},
			"temperature":        gin.H{"type": "number"},
			"timeout_seconds":    gin.H{"type": "integer"},
			"input_cost_per_1m":  gin.H{"type": "number"},
			"output_cost_per_1m": gin.H{"type": "number"},
			"metadata":           gin.H{"type": "object"},
			"created_at":         gin.H{"type": "string", "format": "date-time"},
			"updated_at":         gin.H{"type": "string", "format": "date-time"},
		},
	}
	logSchema := gin.H{
		"type": "object",
		"properties": gin.H{
			"id":                gin.H{"type": "integer"},
			"trace_id":          gin.H{"type": "string"},
			"provider_id":       gin.H{"type": "integer"},
			"provider_name":     gin.H{"type": "string"},
			"provider_key_id":   gin.H{"type": "integer"},
			"provider_key_name": gin.H{"type": "string"},
			"client_key_id":     gin.H{"type": "integer"},
			"client_key_name":   gin.H{"type": "string"},
			"model_public_name": gin.H{"type": "string"},
			"upstream_model":    gin.H{"type": "string"},
			"success":           gin.H{"type": "boolean"},
			"http_status":       gin.H{"type": "integer"},
			"latency_ms":        gin.H{"type": "integer"},
			"prompt_tokens":     gin.H{"type": "integer"},
			"completion_tokens": gin.H{"type": "integer"},
			"total_tokens":      gin.H{"type": "integer"},
			"estimated_cost":    gin.H{"type": "number"},
			"error_type":        gin.H{"type": "string"},
			"error_message":     gin.H{"type": "string"},
			"created_at":        gin.H{"type": "string", "format": "date-time"},
		},
	}
	statsSchema := gin.H{
		"type": "object",
		"properties": gin.H{
			"provider_count":          gin.H{"type": "integer"},
			"provider_key_count":      gin.H{"type": "integer"},
			"model_count":             gin.H{"type": "integer"},
			"client_key_count":        gin.H{"type": "integer"},
			"active_client_key_count": gin.H{"type": "integer"},
			"request_count_24h":       gin.H{"type": "integer"},
			"success_rate_24h":        gin.H{"type": "number"},
			"avg_latency_ms_24h":      gin.H{"type": "number"},
			"prompt_tokens_24h":       gin.H{"type": "integer"},
			"completion_tokens_24h":   gin.H{"type": "integer"},
			"total_tokens_24h":        gin.H{"type": "integer"},
			"estimated_cost_24h":      gin.H{"type": "number"},
			"top_models":              gin.H{"type": "array", "items": gin.H{"type": "object"}},
			"top_providers":           gin.H{"type": "array", "items": gin.H{"type": "object"}},
			"top_clients":             gin.H{"type": "array", "items": gin.H{"type": "object"}},
			"quota_pressure":          gin.H{"type": "array", "items": gin.H{"type": "object"}},
			"recent_requests":         gin.H{"type": "array", "items": gin.H{"$ref": "#/components/schemas/RequestLog"}},
		},
	}

	listEnvelope := func(ref string) gin.H {
		return gin.H{
			"type": "object",
			"properties": gin.H{
				"items": gin.H{
					"type":  "array",
					"items": gin.H{"$ref": ref},
				},
				"total": gin.H{"type": "integer"},
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"openapi": "3.1.0",
		"info": gin.H{
			"title":       "WcsTransfer Gateway API",
			"version":     buildinfo.Version,
			"description": "OpenAI-compatible gateway API for chat completions and embeddings.",
		},
		"servers": []gin.H{
			{"url": serverURL},
		},
		"components": gin.H{
			"securitySchemes": gin.H{
				"bearerAuth": clientBearer,
				"adminAuth":  adminBearer,
			},
			"schemas": gin.H{
				"Error":           errorSchema,
				"Provider":        providerSchema,
				"ClientAPIKey":    clientKeySchema,
				"ProviderKey":     providerKeySchema,
				"Model":           modelSchema,
				"RequestLog":      logSchema,
				"AdminStats":      statsSchema,
				"ProviderList":    listEnvelope("#/components/schemas/Provider"),
				"ClientKeyList":   listEnvelope("#/components/schemas/ClientAPIKey"),
				"ProviderKeyList": listEnvelope("#/components/schemas/ProviderKey"),
				"ModelList":       listEnvelope("#/components/schemas/Model"),
				"RequestLogList":  listEnvelope("#/components/schemas/RequestLog"),
			},
		},
		"security": []gin.H{{"bearerAuth": []string{}}},
		"paths": gin.H{
			"/healthz": gin.H{
				"get": gin.H{
					"summary":   "Health check",
					"responses": gin.H{"200": gin.H{"description": "Service health"}},
				},
			},
			"/version": gin.H{
				"get": gin.H{
					"summary":   "Build and version metadata",
					"responses": gin.H{"200": gin.H{"description": "Build metadata"}},
				},
			},
			"/v1/models": gin.H{
				"get": gin.H{
					"summary":  "List models visible to the current client key",
					"security": []gin.H{{"bearerAuth": []string{}}},
					"responses": gin.H{
						"200": gin.H{"description": "Model list", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/ModelList"}}}},
						"401": gin.H{"description": "Unauthorized", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Error"}}}},
					},
				},
			},
			"/v1/chat/completions": gin.H{
				"post": gin.H{
					"summary":  "Create a chat completion",
					"security": []gin.H{{"bearerAuth": []string{}}},
					"requestBody": gin.H{
						"required": true,
						"content": gin.H{
							"application/json": gin.H{
								"schema": gin.H{
									"type": "object",
									"properties": gin.H{
										"model": gin.H{"type": "string"},
										"messages": gin.H{
											"type":  "array",
											"items": gin.H{"type": "object"},
										},
										"stream": gin.H{"type": "boolean"},
									},
									"required": []string{"model", "messages"},
								},
							},
						},
					},
					"responses": gin.H{
						"200": gin.H{"description": "Chat completion response"},
						"401": gin.H{"description": "Unauthorized", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Error"}}}},
						"403": gin.H{"description": "Model forbidden", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Error"}}}},
						"429": gin.H{"description": "Quota or budget exceeded", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Error"}}}},
					},
				},
			},
			"/v1/embeddings": gin.H{
				"post": gin.H{
					"summary":  "Create embeddings",
					"security": []gin.H{{"bearerAuth": []string{}}},
					"requestBody": gin.H{
						"required": true,
						"content": gin.H{
							"application/json": gin.H{
								"schema": gin.H{
									"type": "object",
									"properties": gin.H{
										"model": gin.H{"type": "string"},
										"input": gin.H{},
									},
									"required": []string{"model", "input"},
								},
							},
						},
					},
					"responses": gin.H{
						"200": gin.H{"description": "Embeddings response"},
						"401": gin.H{"description": "Unauthorized", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Error"}}}},
						"403": gin.H{"description": "Model forbidden", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Error"}}}},
						"429": gin.H{"description": "Quota or budget exceeded", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Error"}}}},
					},
				},
			},
			"/admin/providers": gin.H{
				"get": gin.H{
					"summary":   "List providers",
					"security":  []gin.H{{"adminAuth": []string{}}},
					"responses": gin.H{"200": gin.H{"description": "Provider list", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/ProviderList"}}}}},
				},
				"post": gin.H{
					"summary":  "Create provider",
					"security": []gin.H{{"adminAuth": []string{}}},
					"requestBody": gin.H{
						"required": true,
						"content": gin.H{
							"application/json": gin.H{
								"schema": gin.H{
									"type":     "object",
									"required": []string{"name", "slug", "base_url"},
									"properties": gin.H{
										"name":          gin.H{"type": "string"},
										"slug":          gin.H{"type": "string"},
										"provider_type": gin.H{"type": "string"},
										"base_url":      gin.H{"type": "string"},
										"status":        gin.H{"type": "string"},
										"description":   gin.H{"type": "string"},
										"extra_config":  gin.H{"type": "object"},
									},
								},
							},
						},
					},
					"responses": gin.H{"201": gin.H{"description": "Created provider", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Provider"}}}}},
				},
			},
			"/admin/providers/{id}": gin.H{
				"put": gin.H{
					"summary":  "Update provider",
					"security": []gin.H{{"adminAuth": []string{}}},
					"parameters": []gin.H{
						{"name": "id", "in": "path", "required": true, "schema": gin.H{"type": "integer"}},
					},
					"requestBody": gin.H{
						"required": true,
						"content":  gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}}},
					},
					"responses": gin.H{"200": gin.H{"description": "Updated provider", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Provider"}}}}},
				},
			},
			"/admin/client-keys": gin.H{
				"get": gin.H{
					"summary":   "List client API keys",
					"security":  []gin.H{{"adminAuth": []string{}}},
					"responses": gin.H{"200": gin.H{"description": "Client key list", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/ClientKeyList"}}}}},
				},
				"post": gin.H{
					"summary":  "Create client API key",
					"security": []gin.H{{"adminAuth": []string{}}},
					"requestBody": gin.H{
						"required": true,
						"content":  gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}}},
					},
					"responses": gin.H{"201": gin.H{"description": "Created client key"}},
				},
			},
			"/admin/client-keys/{id}": gin.H{
				"put": gin.H{
					"summary":  "Update client API key",
					"security": []gin.H{{"adminAuth": []string{}}},
					"parameters": []gin.H{
						{"name": "id", "in": "path", "required": true, "schema": gin.H{"type": "integer"}},
					},
					"requestBody": gin.H{
						"required": true,
						"content":  gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}}},
					},
					"responses": gin.H{"200": gin.H{"description": "Updated client key", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/ClientAPIKey"}}}}},
				},
			},
			"/admin/keys": gin.H{
				"get": gin.H{
					"summary":   "List provider keys",
					"security":  []gin.H{{"adminAuth": []string{}}},
					"responses": gin.H{"200": gin.H{"description": "Provider key list", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/ProviderKeyList"}}}}},
				},
				"post": gin.H{
					"summary":  "Create provider key",
					"security": []gin.H{{"adminAuth": []string{}}},
					"requestBody": gin.H{
						"required": true,
						"content":  gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}}},
					},
					"responses": gin.H{"201": gin.H{"description": "Created provider key"}},
				},
			},
			"/admin/keys/{id}": gin.H{
				"put": gin.H{
					"summary":  "Update provider key",
					"security": []gin.H{{"adminAuth": []string{}}},
					"parameters": []gin.H{
						{"name": "id", "in": "path", "required": true, "schema": gin.H{"type": "integer"}},
					},
					"requestBody": gin.H{
						"required": true,
						"content":  gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}}},
					},
					"responses": gin.H{"200": gin.H{"description": "Updated provider key", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/ProviderKey"}}}}},
				},
			},
			"/admin/models": gin.H{
				"get": gin.H{
					"summary":   "List models",
					"security":  []gin.H{{"adminAuth": []string{}}},
					"responses": gin.H{"200": gin.H{"description": "Model list", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/ModelList"}}}}},
				},
				"post": gin.H{
					"summary":  "Create model mapping",
					"security": []gin.H{{"adminAuth": []string{}}},
					"requestBody": gin.H{
						"required": true,
						"content":  gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}}},
					},
					"responses": gin.H{"201": gin.H{"description": "Created model", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Model"}}}}},
				},
			},
			"/admin/models/{id}": gin.H{
				"put": gin.H{
					"summary":  "Update model mapping",
					"security": []gin.H{{"adminAuth": []string{}}},
					"parameters": []gin.H{
						{"name": "id", "in": "path", "required": true, "schema": gin.H{"type": "integer"}},
					},
					"requestBody": gin.H{
						"required": true,
						"content":  gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}}},
					},
					"responses": gin.H{"200": gin.H{"description": "Updated model", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Model"}}}}},
				},
			},
			"/admin/logs": gin.H{
				"get": gin.H{
					"summary":  "List request logs",
					"security": []gin.H{{"adminAuth": []string{}}},
					"parameters": []gin.H{
						{"name": "page", "in": "query", "schema": gin.H{"type": "integer"}},
						{"name": "page_size", "in": "query", "schema": gin.H{"type": "integer"}},
						{"name": "provider_id", "in": "query", "schema": gin.H{"type": "integer"}},
						{"name": "model_public_name", "in": "query", "schema": gin.H{"type": "string"}},
						{"name": "success", "in": "query", "schema": gin.H{"type": "boolean"}},
						{"name": "http_status", "in": "query", "schema": gin.H{"type": "integer"}},
						{"name": "trace_id", "in": "query", "schema": gin.H{"type": "string"}},
						{"name": "created_from", "in": "query", "schema": gin.H{"type": "string", "format": "date-time"}},
						{"name": "created_to", "in": "query", "schema": gin.H{"type": "string", "format": "date-time"}},
					},
					"responses": gin.H{"200": gin.H{"description": "Request log list", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/RequestLogList"}}}}},
				},
			},
			"/admin/logs/{id}": gin.H{
				"get": gin.H{
					"summary":  "Get request log detail",
					"security": []gin.H{{"adminAuth": []string{}}},
					"parameters": []gin.H{
						{"name": "id", "in": "path", "required": true, "schema": gin.H{"type": "integer"}},
					},
					"responses": gin.H{"200": gin.H{"description": "Log detail", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/RequestLog"}}}}},
				},
			},
			"/admin/stats": gin.H{
				"get": gin.H{
					"summary":   "Get dashboard stats",
					"security":  []gin.H{{"adminAuth": []string{}}},
					"responses": gin.H{"200": gin.H{"description": "Aggregated dashboard metrics", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/AdminStats"}}}}},
				},
			},
			"/admin/debug/chat/completions": gin.H{
				"post": gin.H{
					"summary":  "Run admin debug chat completion",
					"security": []gin.H{{"adminAuth": []string{}}},
					"requestBody": gin.H{
						"required": true,
						"content":  gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}}},
					},
					"responses": gin.H{"200": gin.H{"description": "Debug chat response"}},
				},
			},
			"/admin/debug/embeddings": gin.H{
				"post": gin.H{
					"summary":  "Run admin debug embeddings",
					"security": []gin.H{{"adminAuth": []string{}}},
					"requestBody": gin.H{
						"required": true,
						"content":  gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}}},
					},
					"responses": gin.H{"200": gin.H{"description": "Debug embeddings response"}},
				},
			},
		},
	})
}

func (h *Handler) SwaggerUI(c *gin.Context) {
	html := `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>WcsTransfer API Docs</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css" />
    <style>
      html, body {
        margin: 0;
        padding: 0;
        background: #f5f7fb;
      }
      #swagger-ui {
        max-width: 1280px;
        margin: 0 auto;
      }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js" crossorigin></script>
    <script>
      window.onload = function () {
        window.ui = SwaggerUIBundle({
          url: "/openapi.json",
          dom_id: "#swagger-ui",
          deepLinking: true,
          presets: [SwaggerUIBundle.presets.apis],
          layout: "BaseLayout"
        });
      };
    </script>
  </body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func (h *Handler) ReDoc(c *gin.Context) {
	html := `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>WcsTransfer API Reference</title>
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <redoc spec-url="/openapi.json"></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
  </body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
