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

// Stores 封装了系统所需的所有数据访问接口
type Stores struct {
	Admin     repository.AdminStore
	AdminAuth repository.AdminAuthStore
	Auth      repository.ClientAuthStore
	Log       repository.RequestLogWriter
	Public    repository.PublicModelStore
	UserAuth  repository.UserAuthStore
	UserKeys  repository.UserClientKeyStore
}

// New 初始化并返回 Gin 路由引擎，配置所有路由分组和中间件
func New(cfg config.Config, deps *platform.Dependencies, stores *Stores) *gin.Engine {
	gin.SetMode(cfg.GinMode)

	engine := gin.New()
	// 基础中间件：CORS、日志、恢复、请求 ID
	engine.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestID())

	// 依赖组装与兜底
	resolvedStores := resolveStores(deps, stores)
	tracker := keyhealth.NewTracker() // 上游密钥健康检查追踪
	quota := clientquota.New(nil)     // 租户额度/频率控制器
	
	// 如果未注入 Worker（如单元测试中），则使用同步 Worker 确保测试可预测性
	workerPool := platform.NewSyncWorker()
	if deps != nil {
		if deps.Redis != nil {
			quota = clientquota.New(deps.Redis)
		}
		if deps.Worker != nil {
			workerPool = deps.Worker
		}
	}
	
	systemHandler := system.NewHandler(cfg, deps)
	adminTokenService := adminauthsvc.New(cfg.AuthTokenSecret)
	openAIHandler := openai.NewHandler(cfg, resolvedStores.Public, resolvedStores.Log, nil, workerPool, tracker, quota)
	adminHandler := admin.NewHandler(resolvedStores.Admin, tracker, quota)
	adminAuthHandler := adminauthapi.NewHandler(resolvedStores.AdminAuth, adminTokenService)
	userTokenService := userauth.New(cfg.AuthTokenSecret)
	userHandler := tenant.NewHandler(resolvedStores.UserAuth, resolvedStores.UserKeys, userTokenService)
	
	enableDocs := cfg.EnableDocs || cfg.Env == "test"
	enableAdminDebug := cfg.EnableAdminDebug || cfg.Env == "test"

	// --- 基础公开接口 ---
	engine.GET("/healthz", systemHandler.Healthz) // 健康检查
	engine.GET("/version", systemHandler.Version) // 版本信息
	if enableDocs {
		// API 文档支持
		engine.GET("/openapi.json", systemHandler.OpenAPI)
		engine.GET("/docs", systemHandler.SwaggerUI)
		engine.GET("/redoc", systemHandler.ReDoc)
	}

	// --- 认证分组 ---
	// 租户门户登录
	authGroup := engine.Group("/portal/auth")
	authGroup.Use(middleware.NoStore())
	{
		authGroup.POST("/login", userHandler.Login)
	}

	// 管理端登录
	adminAuthGroup := engine.Group("/admin/auth")
	adminAuthGroup.Use(middleware.NoStore())
	{
		adminAuthGroup.POST("/login", adminAuthHandler.Login)
	}

	// --- 租户门户分组 (Portal) ---
	portalGroup := engine.Group("/portal")
	portalGroup.Use(middleware.NoStore())
	portalGroup.Use(middleware.TenantUserAuth(userTokenService)) // 租户 JWT 鉴权
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
		
		// 门户在线调试工具：模拟真实 API 调用逻辑
		portalGroup.POST("/client-keys/:id/debug/chat/completions", func(c *gin.Context) {
			// (内部逻辑已包含权限检查、模型解析、协议转换等)
			claims, ok := middleware.UserClaimsFromContext(c)
			if !ok {
				apierror.Write(c, apierror.CodeUnauthorized, "未授权访问")
				return
			}
			keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				apierror.Write(c, apierror.CodeInvalidParam, "无效的密钥 ID")
				return
			}
			// 校验密钥所有权
			key, err := resolvedStores.UserKeys.GetUserClientAPIKeyByID(c.Request.Context(), claims.Sub, keyID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					apierror.Write(c, apierror.CodeNotFound, "未找到该密钥")
				} else {
					apierror.Write(c, apierror.CodeInternalError, "系统内部错误")
				}
				return
			}
			if key.Status != "active" {
				apierror.Write(c, apierror.CodeInvalidState, "该密钥已被禁用")
				return
			}

			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				apierror.Write(c, apierror.CodeInvalidBody, "无法读取请求体")
				return
			}

			var req struct {
				Model        string `json:"model"`
				ProviderType string `json:"provider_type"`
				Stream       bool   `json:"stream"`
			}
			if json.Unmarshal(bodyBytes, &req) != nil || req.Model == "" {
				apierror.Write(c, apierror.CodeInvalidParam, "必须提供模型名称")
				return
			}
			if req.ProviderType == "" {
				apierror.Write(c, apierror.CodeInvalidParam, "必须提供供应商类型")
				return
			}

			// 解析路由与可用密钥
			route, err := resolvedStores.Public.ResolveModelRoute(c.Request.Context(), req.Model)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					apierror.Write(c, apierror.CodeNotFound, "模型未找到或当前不可用")
				} else {
					apierror.Write(c, apierror.CodeInternalError, "解析模型路由失败")
				}
				return
			}
			
			// VertexAI 走 ADC 模式，不需要配置 provider key
			isVertexAI := strings.EqualFold(strings.TrimSpace(route.Provider.ProviderType), "vertexai")
			if len(route.Keys) == 0 && !isVertexAI {
				apierror.Write(c, apierror.CodeRoutingError, "该模型当前没有可用的上游密钥")
				return
			}

			// 校验供应商类型匹配逻辑
			normalizeType := func(pt string) string {
				pt = strings.ToLower(strings.TrimSpace(pt))
				if pt == "vertexai" { return "gemini" }
				if pt == "openai_compatible" { return "openai" }
				return pt
			}
			if normalizeType(req.ProviderType) != normalizeType(route.Provider.ProviderType) {
				apierror.Write(c, apierror.CodeInvalidParam,
					fmt.Sprintf("模型 %s 属于 %s 供应商，请切换调试类型后重试", req.Model, route.Provider.ProviderType))
				return
			}

			// 设置上下文并执行代理调用
			middleware.SetClientAPIKeyInContext(c, key)
			switch normalizeType(route.Provider.ProviderType) {
			case "anthropic":
				// Anthropic 协议适配
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
				// OpenAI messages 格式转 Gemini contents 格式
				var full map[string]any
				if json.Unmarshal(bodyBytes, &full) == nil {
					if msgs, ok := full["messages"].([]any); ok {
						contents := make([]any, 0, len(msgs))
						for _, msg := range msgs {
							m, ok := msg.(map[string]any)
							if !ok { continue }
							role, _ := m["role"].(string)
							if strings.EqualFold(role, "assistant") { role = "model" }
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
			default: // openai / compatible
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				openAIHandler.ChatCompletions(c)
			}
		})
	}

	// --- 公开代理 API 分组 (v1) ---
	v1 := engine.Group("/v1")
	v1.Use(middleware.PublicAPIAuth(resolvedStores.Auth, resolvedStores.Log)) // API Key 鉴权
	v1.Use(middleware.PublicAPIQuota(quota))                                // 额度与频率限制
	{
		v1.GET("/models", openAIHandler.ListModels)
		v1.POST("/chat/completions", openAIHandler.ChatCompletions)
		v1.POST("/embeddings", openAIHandler.Embeddings)
		v1.POST("/messages", openAIHandler.Messages)
		v1.POST("/gemini/generate-content", openAIHandler.GeminiGenerateContent)
		v1.POST("/gemini/stream-generate-content", openAIHandler.GeminiStreamGenerateContent)
		// 支持 Google Gemini 原生格式: /v1/models/{model}:generateContent
		v1.POST("/models/*action", openAIHandler.GeminiNativeAPI)
	}

	// Google Gemini 原生 v1beta 版本适配
	v1beta := engine.Group("/v1beta")
	v1beta.Use(middleware.PublicAPIAuth(resolvedStores.Auth, resolvedStores.Log))
	v1beta.Use(middleware.PublicAPIQuota(quota))
	{
		v1beta.POST("/models/*action", openAIHandler.GeminiNativeAPI)
	}

	// --- 管理端全权管理分组 (Admin) ---
	adminGroup := engine.Group("/admin")
	adminGroup.Use(middleware.NoStore())
	adminGroup.Use(middleware.AdminAuth(adminTokenService)) // 管理员 JWT 鉴权
	{
		adminGroup.GET("/me", adminAuthHandler.Me)
		
		// 提供商管理
		adminGroup.GET("/providers", adminHandler.ListProviders)
		adminGroup.POST("/providers", adminHandler.CreateProvider)
		adminGroup.PUT("/providers/:id", adminHandler.UpdateProvider)
		
		// 用户与钱包管理
		adminGroup.GET("/users", adminHandler.ListUsers)
		adminGroup.POST("/users", adminHandler.CreateUser)
		adminGroup.PUT("/users/:id/status", adminHandler.UpdateUserStatus)
		adminGroup.POST("/users/:id/reset-password", adminHandler.ResetUserPassword)
		adminGroup.POST("/users/:id/wallet/adjust", adminHandler.AdjustUserWallet)   // 充值
		adminGroup.POST("/users/:id/wallet/correct", adminHandler.CorrectUserWallet) // 修正
		adminGroup.GET("/users/:id/wallet/ledger", adminHandler.ListUserWalletLedger)
		adminGroup.GET("/users/:id/billing/export", adminHandler.ExportUserBilling)
		
		// 租户 API Key 管理
		adminGroup.GET("/client-keys", adminHandler.ListClientAPIKeys)
		adminGroup.PUT("/client-keys/:id", adminHandler.UpdateClientAPIKey)
		
		// 上游密钥池管理
		adminGroup.GET("/keys", adminHandler.ListProviderKeys)
		adminGroup.POST("/keys", adminHandler.CreateProviderKey)
		adminGroup.PUT("/keys/:id", adminHandler.UpdateProviderKey)
		
		// 模型映射与策略管理
		adminGroup.GET("/models", adminHandler.ListModels)
		adminGroup.POST("/models", adminHandler.CreateModel)
		adminGroup.PUT("/models/:id", adminHandler.UpdateModel)
		
		// 审计日志与统计
		adminGroup.GET("/logs", adminHandler.ListLogs)
		adminGroup.GET("/logs/export", adminHandler.ExportLogs)
		adminGroup.GET("/logs/:id", adminHandler.GetLogDetail)
		adminGroup.GET("/stats", adminHandler.GetStats)
		
		// 对账功能
		adminGroup.GET("/reconciliation/users", adminHandler.GetUserBillingReconciliation)
		
		// 管理端在线调试
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
