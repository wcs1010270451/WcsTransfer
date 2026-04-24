package router

import (
	"github.com/gin-gonic/gin"

	"wcstransfer/backend/internal/api/admin"
	adminauthapi "wcstransfer/backend/internal/api/adminauth"
	"wcstransfer/backend/internal/api/openai"
	"wcstransfer/backend/internal/api/system"
	"wcstransfer/backend/internal/api/tenant"
	"wcstransfer/backend/internal/config"
	"wcstransfer/backend/internal/middleware"
	"wcstransfer/backend/internal/platform"
	"wcstransfer/backend/internal/repository"
	repopostgres "wcstransfer/backend/internal/repository/postgres"
	adminauthsvc "wcstransfer/backend/internal/service/adminauth"
	"wcstransfer/backend/internal/service/clientquota"
	"wcstransfer/backend/internal/service/keyhealth"
	"wcstransfer/backend/internal/service/tenantauth"
)

type Stores struct {
	Admin      repository.AdminStore
	AdminAuth  repository.AdminAuthStore
	Auth       repository.ClientAuthStore
	Log        repository.RequestLogWriter
	Public     repository.PublicModelStore
	TenantAuth repository.TenantAuthStore
	TenantKeys repository.TenantClientKeyStore
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
	tenantTokenService := tenantauth.New(cfg.AuthTokenSecret)
	tenantHandler := tenant.NewHandler(resolvedStores.TenantAuth, resolvedStores.TenantKeys, tenantTokenService)
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
		authGroup.POST("/login", tenantHandler.Login)
	}

	adminAuthGroup := engine.Group("/admin/auth")
	adminAuthGroup.Use(middleware.NoStore())
	{
		adminAuthGroup.POST("/login", adminAuthHandler.Login)
	}

	portalGroup := engine.Group("/portal")
	portalGroup.Use(middleware.NoStore())
	portalGroup.Use(middleware.TenantUserAuth(tenantTokenService))
	{
		portalGroup.GET("/me", tenantHandler.Me)
		portalGroup.GET("/models", tenantHandler.Models)
		portalGroup.GET("/stats", tenantHandler.Stats)
		portalGroup.GET("/wallet/ledger", tenantHandler.WalletLedger)
		portalGroup.GET("/billing/export", tenantHandler.ExportBilling)
		portalGroup.GET("/logs", tenantHandler.Logs)
		portalGroup.GET("/logs/:id", tenantHandler.LogDetail)
		portalGroup.GET("/client-keys", tenantHandler.ListClientKeys)
		portalGroup.POST("/client-keys", tenantHandler.CreateClientKey)
		portalGroup.POST("/client-keys/:id/disable", tenantHandler.DisableClientKey)
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
	}

	adminGroup := engine.Group("/admin")
	adminGroup.Use(middleware.NoStore())
	adminGroup.Use(middleware.AdminAuth(adminTokenService))
	{
		adminGroup.GET("/me", adminAuthHandler.Me)
		adminGroup.GET("/providers", adminHandler.ListProviders)
		adminGroup.POST("/providers", adminHandler.CreateProvider)
		adminGroup.PUT("/providers/:id", adminHandler.UpdateProvider)
		adminGroup.GET("/tenants", adminHandler.ListTenants)
		adminGroup.POST("/tenants", adminHandler.CreateTenant)
		adminGroup.PUT("/tenants/:id", adminHandler.UpdateTenant)
		adminGroup.GET("/tenants/:id/users", adminHandler.ListTenantUsers)
		adminGroup.POST("/tenants/:id/users", adminHandler.CreateTenantUser)
		adminGroup.PUT("/tenants/:id/users/:userId/status", adminHandler.UpdateTenantUserStatus)
		adminGroup.POST("/tenants/:id/users/:userId/reset-password", adminHandler.ResetTenantUserPassword)
		adminGroup.POST("/tenants/:id/wallet/adjust", adminHandler.AdjustTenantWallet)
		adminGroup.POST("/tenants/:id/wallet/correct", adminHandler.CorrectTenantWallet)
		adminGroup.GET("/tenants/:id/wallet/ledger", adminHandler.ListTenantWalletLedger)
		adminGroup.GET("/tenants/:id/billing/export", adminHandler.ExportTenantBilling)
		adminGroup.GET("/client-keys", adminHandler.ListClientAPIKeys)
		adminGroup.POST("/client-keys", adminHandler.CreateClientAPIKey)
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
		adminGroup.GET("/reconciliation/tenants", adminHandler.GetTenantBillingReconciliation)
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
		resolved.TenantAuth = store
		resolved.TenantKeys = store
	}

	return resolved
}
