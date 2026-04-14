package router

import (
	"github.com/gin-gonic/gin"

	"wcstransfer/backend/internal/api/admin"
	"wcstransfer/backend/internal/api/openai"
	"wcstransfer/backend/internal/api/system"
	"wcstransfer/backend/internal/config"
	"wcstransfer/backend/internal/middleware"
	"wcstransfer/backend/internal/platform"
	"wcstransfer/backend/internal/repository"
	repopostgres "wcstransfer/backend/internal/repository/postgres"
	"wcstransfer/backend/internal/service/clientquota"
	"wcstransfer/backend/internal/service/keyhealth"
)

type Stores struct {
	Admin  repository.AdminStore
	Auth   repository.ClientAuthStore
	Log    repository.RequestLogWriter
	Public repository.PublicModelStore
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
	openAIHandler := openai.NewHandler(resolvedStores.Public, resolvedStores.Log, nil, tracker, quota)
	adminHandler := admin.NewHandler(resolvedStores.Admin, tracker)

	engine.GET("/healthz", systemHandler.Healthz)
	engine.GET("/version", systemHandler.Version)

	v1 := engine.Group("/v1")
	v1.Use(middleware.PublicAPIAuth(resolvedStores.Auth))
	v1.Use(middleware.PublicAPIQuota(quota))
	{
		v1.GET("/models", openAIHandler.ListModels)
		v1.POST("/chat/completions", openAIHandler.ChatCompletions)
	}

	adminGroup := engine.Group("/admin")
	adminGroup.Use(middleware.AdminAuth(cfg.AdminToken))
	{
		adminGroup.GET("/providers", adminHandler.ListProviders)
		adminGroup.POST("/providers", adminHandler.CreateProvider)
		adminGroup.PUT("/providers/:id", adminHandler.UpdateProvider)
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
		adminGroup.POST("/debug/chat/completions", openAIHandler.AdminDebugChatCompletions)
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
		resolved.Auth = store
		resolved.Log = store
		resolved.Public = store
	}

	return resolved
}
