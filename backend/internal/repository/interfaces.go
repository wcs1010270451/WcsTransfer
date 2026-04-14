package repository

import (
	"context"

	"wcstransfer/backend/internal/entity"
)

type AdminStore interface {
	ListProviders(ctx context.Context) ([]entity.Provider, error)
	CreateProvider(ctx context.Context, input entity.CreateProviderInput) (entity.Provider, error)
	UpdateProvider(ctx context.Context, input entity.UpdateProviderInput) (entity.Provider, error)
	ListClientAPIKeys(ctx context.Context) ([]entity.ClientAPIKey, error)
	CreateClientAPIKey(ctx context.Context, input entity.CreateClientAPIKeyInput) (entity.ClientAPIKey, error)
	UpdateClientAPIKey(ctx context.Context, input entity.UpdateClientAPIKeyInput) (entity.ClientAPIKey, error)
	ListProviderKeys(ctx context.Context) ([]entity.ProviderKey, error)
	CreateProviderKey(ctx context.Context, input entity.CreateProviderKeyInput) (entity.ProviderKey, error)
	UpdateProviderKey(ctx context.Context, input entity.UpdateProviderKeyInput) (entity.ProviderKey, error)
	ListModels(ctx context.Context) ([]entity.Model, error)
	CreateModel(ctx context.Context, input entity.CreateModelInput) (entity.Model, error)
	UpdateModel(ctx context.Context, input entity.UpdateModelInput) (entity.Model, error)
	ListRequestLogs(ctx context.Context, input entity.ListRequestLogsInput) (entity.RequestLogPage, error)
	GetRequestLog(ctx context.Context, id int64) (entity.RequestLogDetail, error)
	ExportRequestLogs(ctx context.Context, input entity.ListRequestLogsInput) ([]entity.RequestLog, error)
	GetDashboardStats(ctx context.Context) (entity.DashboardStats, error)
}

type RequestLogWriter interface {
	CreateRequestLog(ctx context.Context, input entity.CreateRequestLogInput) error
}

type PublicModelStore interface {
	ListEnabledModels(ctx context.Context) ([]entity.Model, error)
	ResolveModelRoute(ctx context.Context, publicName string) (entity.ModelRoute, error)
}

type ClientAuthStore interface {
	AuthenticateClientAPIKey(ctx context.Context, rawKey string) (entity.ClientAPIKey, error)
}
