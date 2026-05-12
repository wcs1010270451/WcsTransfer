package router

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"wcstransfer/backend/internal/api/admin"
	adminauthapi "wcstransfer/backend/internal/api/adminauth"
	"wcstransfer/backend/internal/api/openai"
	"wcstransfer/backend/internal/api/system"
	"wcstransfer/backend/internal/api/tenant"
	"wcstransfer/backend/internal/apierror"
	"wcstransfer/backend/internal/config"
	"wcstransfer/backend/internal/middleware"
	"wcstransfer/backend/internal/platform"
	"wcstransfer/backend/internal/repository"
	repopostgres "wcstransfer/backend/internal/repository/postgres"
	adminauthsvc "wcstransfer/backend/internal/service/adminauth"
	"wcstransfer/backend/internal/service/clientquota"
	"wcstransfer/backend/internal/service/keyhealth"
	"wcstransfer/backend/internal/service/userauth"
)

type Stores struct {
	Admin     repository.AdminStore
	AdminAuth repository.AdminAuthStore
	Auth      repository.ClientAuthStore
	Log       repository.RequestLogWriter
	Public    repository.PublicModelStore
	UserAuth  repository.UserAuthStore
	UserKeys  repository.UserClientKeyStore
}

func New(cfg config.Config, deps *platform.Dependencies, stores *Stores) *gin.Engine {
	gin.SetMode(cfg.GinMode)

	engine := gin.New()
	engine.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestID())

	resolvedStores := resolveStores(deps, stores)
	tracker := keyhealth.NewTracker()
	quota := clientquota.New(nil)
	if deps != nil {
		quota = clientquota.New(deps.Redis)
	}
	systemHandler := system.NewHandler(cfg, deps)
	adminTokenService := adminauthsvc.New(cfg.AuthTokenSecret)
	openAIHandler := openai.NewHandler(resolvedStores.Public, resolvedStores.Log, nil, tracker, quota)
	adminHandler := admin.NewHandler(resolvedStores.Admin, tracker, quota)
	adminAuthHandler := adminauthapi.NewHandler(resolvedStores.AdminAuth, adminTokenService)
	userTokenService := userauth.New(cfg.AuthTokenSecret)
	userHandler := tenant.NewHandler(resolvedStores.UserAuth, resolvedStores.UserKeys, userTokenService)
	enableDocs := cfg.EnableDocs || cfg.Env == "test"
	enableAdminDebug := cfg.EnableAdminDebug || cfg.Env == "test"

	engine.GET("/healthz", systemHandler.Healthz)
	engine.GET("/version", systemHandler.Version)
	if enableDocs {
		engine.GET("/openapi.json", systemHandler.OpenAPI)
		engine.GET("/docs", systemHandler.SwaggerUI)
		engine.GET("/redoc", systemHandler.ReDoc)
	}

	authGroup := engine.Group("/portal/auth")
	authGroup.Use(middleware.NoStore())
	{
		authGroup.POST("/login", userHandler.Login)
	}

	adminAuthGroup := engine.Group("/admin/auth")
	adminAuthGroup.Use(middleware.NoStore())
	{
		adminAuthGroup.POST("/login", adminAuthHandler.Login)
	}

	portalGroup := engine.Group("/portal")
	portalGroup.Use(middleware.NoStore())
	portalGroup.Use(middleware.TenantUserAuth(userTokenService))
	{
		portalGroup.GET("/me", userHandler.Me)
		portalGroup.GET("/models", userHandler.Models)
		portalGroup.GET("/stats", userHandler.Stats)
		portalGroup.GET("/stats/daily", userHandler.DailyStats)
		portalGroup.GET("/wallet/ledger", userHandler.WalletLedger)
		portalGroup.GET("/billing/export", userHandler.ExportBilling)
		portalGroup.GET("/logs", userHandler.Logs)
		portalGroup.GET("/logs/:id", userHandler.LogDetail)
		portalGroup.GET("/client-keys", userHandler.ListClientKeys)
		portalGroup.POST("/client-keys", userHandler.CreateClientKey)
		portalGroup.GET("/client-keys/:id/model-stats", userHandler.ClientKeyModelStats)
		portalGroup.PATCH("/client-keys/:id", userHandler.RenameClientKey)
		portalGroup.POST("/client-keys/:id/disable", userHandler.DisableClientKey)
		portalGroup.POST("/client-keys/:id/debug/chat/completions", func(c *gin.Context) {
			claims, ok := middleware.UserClaimsFromContext(c)
			if !ok {
				apierror.Write(c, apierror.CodeUnauthorized, "unauthorized")
				return
			}
			keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				apierror.Write(c, apierror.CodeInvalidParam, "invalid key id")
				return
			}
			key, err := resolvedStores.UserKeys.GetUserClientAPIKeyByID(c.Request.Context(), claims.Sub, keyID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					apierror.Write(c, apierror.CodeNotFound, "key not found")
				} else {
					apierror.Write(c, apierror.CodeInternalError, "internal error")
				}
				return
			}
			if key.Status != "active" {
				apierror.Write(c, apierror.CodeInvalidState, "key is disabled")
				return
			}

			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				apierror.Write(c, apierror.CodeInvalidBody, "failed to read request body")
				return
			}

			var req struct {
				Model        string `json:"model"`
				ProviderType string `json:"provider_type"`
				Stream       bool   `json:"stream"`
			}
			if json.Unmarshal(bodyBytes, &req) != nil || req.Model == "" {
				apierror.Write(c, apierror.CodeInvalidParam, "model is required")
				return
			}
			if req.ProviderType == "" {
				apierror.Write(c, apierror.CodeInvalidParam, "provider_type is required")
				return
			}

			route, err := resolvedStores.Public.ResolveModelRoute(c.Request.Context(), req.Model)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					apierror.Write(c, apierror.CodeNotFound, "model not found or unavailable")
				} else {
					apierror.Write(c, apierror.CodeInternalError, "failed to resolve model")
				}
				return
			}
			// vertexai 走 ADC，不需要配置 provider key
			isVertexAI := strings.EqualFold(strings.TrimSpace(route.Provider.ProviderType), "vertexai")
			if len(route.Keys) == 0 && !isVertexAI {
				apierror.Write(c, apierror.CodeRoutingError, "no active provider key available for this model")
				return
			}

			// 校验用户选择的供应商类型与模型实际类型是否匹配
			// gemini 和 vertex_ai 对用户来说都属于 Gemini 类型
			normalizeType := func(pt string) string {
				pt = strings.ToLower(strings.TrimSpace(pt))
				if pt == "vertexai" {
					return "gemini"
				}
				if pt == "openai_compatible" {
					return "openai"
				}
				return pt
			}
			if normalizeType(req.ProviderType) != normalizeType(route.Provider.ProviderType) {
				apierror.Write(c, apierror.CodeInvalidParam,
					fmt.Sprintf("模型 %s 属于 %s 供应商，请切换调试类型后重试", req.Model, route.Provider.ProviderType))
				return
			}

			middleware.SetClientAPIKeyInContext(c, key)
			switch normalizeType(route.Provider.ProviderType) {
			case "anthropic":
				if route.Model.MaxTokens <= 0 {
					var full map[string]any
					if json.Unmarshal(bodyBytes, &full) == nil {
						if _, has := full["max_tokens"]; !has {
							full["max_tokens"] = 4096
							if patched, merr := json.Marshal(full); merr == nil {
								bodyBytes = patched
							}
						}
					}
				}
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				openAIHandler.Messages(c)
			case "gemini":
				// 将 OpenAI messages 格式转为 Gemini contents 格式
				var full map[string]any
				if json.Unmarshal(bodyBytes, &full) == nil {
					if msgs, ok := full["messages"].([]any); ok {
						contents := make([]any, 0, len(msgs))
						for _, msg := range msgs {
							m, ok := msg.(map[string]any)
							if !ok {
								continue
							}
							role, _ := m["role"].(string)
							if strings.EqualFold(role, "assistant") {
								role = "model"
							}
							text, _ := m["content"].(string)
							contents = append(contents, map[string]any{
								"role":  role,
								"parts": []any{map[string]any{"text": text}},
							})
						}
						full["contents"] = contents
						delete(full, "messages")
					}
					delete(full, "stream")
					delete(full, "provider_type")
					if patched, merr := json.Marshal(full); merr == nil {
						bodyBytes = patched
					}
				}
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				if req.Stream {
					openAIHandler.GeminiStreamGenerateContent(c)
				} else {
					openAIHandler.GeminiGenerateContent(c)
				}
			default: // openai
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				openAIHandler.ChatCompletions(c)
			}
		})
	}

	v1 := engine.Group("/v1")
	v1.Use(middleware.PublicAPIAuth(resolvedStores.Auth, resolvedStores.Log))
	v1.Use(middleware.PublicAPIQuota(quota))
	{
		v1.GET("/models", openAIHandler.ListModels)
		v1.POST("/chat/completions", openAIHandler.ChatCompletions)
		v1.POST("/embeddings", openAIHandler.Embeddings)
		v1.POST("/messages", openAIHandler.Messages)
		v1.POST("/gemini/generate-content", openAIHandler.GeminiGenerateContent)
		v1.POST("/gemini/stream-generate-content", openAIHandler.GeminiStreamGenerateContent)
		// Google Gemini native API format: /v1/models/{model}:generateContent
		v1.POST("/models/*action", openAIHandler.GeminiNativeAPI)
	}

	// Google Gemini native API compat: /v1beta/models/{model}:generateContent
	v1beta := engine.Group("/v1beta")
	v1beta.Use(middleware.PublicAPIAuth(resolvedStores.Auth, resolvedStores.Log))
	v1beta.Use(middleware.PublicAPIQuota(quota))
	{
		v1beta.POST("/models/*action", openAIHandler.GeminiNativeAPI)
	}

	adminGroup := engine.Group("/admin")
	adminGroup.Use(middleware.NoStore())
	adminGroup.Use(middleware.AdminAuth(adminTokenService))
	{
		adminGroup.GET("/me", adminAuthHandler.Me)
		adminGroup.GET("/providers", adminHandler.ListProviders)
		adminGroup.POST("/providers", adminHandler.CreateProvider)
		adminGroup.PUT("/providers/:id", adminHandler.UpdateProvider)
		adminGroup.GET("/users", adminHandler.ListUsers)
		adminGroup.POST("/users", adminHandler.CreateUser)
		adminGroup.PUT("/users/:id/status", adminHandler.UpdateUserStatus)
		adminGroup.POST("/users/:id/reset-password", adminHandler.ResetUserPassword)
		adminGroup.POST("/users/:id/wallet/adjust", adminHandler.AdjustUserWallet)
		adminGroup.POST("/users/:id/wallet/correct", adminHandler.CorrectUserWallet)
		adminGroup.GET("/users/:id/wallet/ledger", adminHandler.ListUserWalletLedger)
		adminGroup.GET("/users/:id/billing/export", adminHandler.ExportUserBilling)
		adminGroup.GET("/client-keys", adminHandler.ListClientAPIKeys)
		adminGroup.PUT("/client-keys/:id", adminHandler.UpdateClientAPIKey)
		adminGroup.GET("/keys", adminHandler.ListProviderKeys)
		adminGroup.POST("/keys", adminHandler.CreateProviderKey)
		adminGroup.PUT("/keys/:id", adminHandler.UpdateProviderKey)
		adminGroup.GET("/models", adminHandler.ListModels)
		adminGroup.POST("/models", adminHandler.CreateModel)
		adminGroup.PUT("/models/:id", adminHandler.UpdateModel)
		adminGroup.GET("/logs", adminHandler.ListLogs)
		adminGroup.GET("/logs/export", adminHandler.ExportLogs)
		adminGroup.GET("/logs/:id", adminHandler.GetLogDetail)
		adminGroup.GET("/stats", adminHandler.GetStats)
		adminGroup.GET("/reconciliation/users", adminHandler.GetUserBillingReconciliation)
		if enableAdminDebug {
			adminGroup.POST("/debug/chat/completions", openAIHandler.AdminDebugChatCompletions)
			adminGroup.POST("/debug/embeddings", openAIHandler.AdminDebugEmbeddings)
			adminGroup.POST("/debug/messages", openAIHandler.AdminDebugMessages)
			adminGroup.POST("/debug/gemini/generate-content", openAIHandler.AdminDebugGeminiGenerateContent)
			adminGroup.POST("/debug/gemini/stream-generate-content", openAIHandler.AdminDebugGeminiStreamGenerateContent)
		}
	}

	return engine
}

func resolveStores(deps *platform.Dependencies, stores *Stores) *Stores {
	if stores != nil {
		return stores
	}

	resolved := &Stores{}
	if deps != nil && deps.Postgres != nil {
		store := repopostgres.NewStore(deps.Postgres)
		resolved.Admin = store
		resolved.AdminAuth = store
		resolved.Auth = store
		resolved.Log = store
		resolved.Public = store
		resolved.UserAuth = store
		resolved.UserKeys = store
	}

	return resolved
}
