package repository

import (
	"context"
	"time"

	"wcstransfer/backend/internal/entity"
)

type AdminStore interface {
	ListProviders(ctx context.Context) ([]entity.Provider, error)
	CreateProvider(ctx context.Context, input entity.CreateProviderInput) (entity.Provider, error)
	UpdateProvider(ctx context.Context, input entity.UpdateProviderInput) (entity.Provider, error)
	ListUsers(ctx context.Context) ([]entity.User, error)
	CreateUser(ctx context.Context, input entity.CreateUserInput) (entity.User, error)
	UpdateUserStatus(ctx context.Context, input entity.UpdateUserStatusInput) (entity.User, error)
	ResetUserPassword(ctx context.Context, input entity.ResetUserPasswordInput) error
	AdjustUserWallet(ctx context.Context, input entity.UserWalletAdjustmentInput) (entity.User, error)
	CorrectUserWallet(ctx context.Context, input entity.UserWalletCorrectionInput) (entity.User, error)
	ListUserWalletLedger(ctx context.Context, userID int64, page int, pageSize int) (entity.WalletLedgerPage, error)
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
	ExportUserRequestLogs(ctx context.Context, userID int64, input entity.ListRequestLogsInput) ([]entity.RequestLog, error)
	GetDashboardStats(ctx context.Context) (entity.DashboardStats, error)
	CreateAdminActionLog(ctx context.Context, input entity.CreateAdminActionLogInput) error
	GetProviderRequestAnomalies(ctx context.Context, since time.Time, minRequests int, rateLimitedThreshold float64, serverErrorThreshold float64) ([]entity.ProviderRequestAnomaly, error)
	GetUserBillingReconciliation(ctx context.Context) ([]entity.UserBillingReconciliation, error)
}

type AdminAuthStore interface {
	AuthenticateAdminUser(ctx context.Context, username string, password string) (entity.AdminUser, error)
	UpdateAdminUserLastLogin(ctx context.Context, userID int64) error
	GetAdminUserByID(ctx context.Context, userID int64) (entity.AdminUser, error)
}

type UserAuthStore interface {
	AuthenticateUser(ctx context.Context, email string, password string) (entity.User, error)
	UpdateUserLastLogin(ctx context.Context, userID int64) error
	GetUserByID(ctx context.Context, userID int64) (entity.User, error)
}

type UserClientKeyStore interface {
	ListUserClientAPIKeys(ctx context.Context, userID int64) ([]entity.ClientAPIKey, error)
	CreateUserClientAPIKey(ctx context.Context, input entity.CreateClientAPIKeyInput) (entity.ClientAPIKey, error)
	DisableUserClientAPIKey(ctx context.Context, userID int64, id int64) (entity.ClientAPIKey, error)
	GetUserPortalStats(ctx context.Context, userID int64) (entity.UserPortalStats, error)
	ListUserWalletLedger(ctx context.Context, userID int64, page int, pageSize int) (entity.WalletLedgerPage, error)
	ListUserRequestLogs(ctx context.Context, userID int64, input entity.ListRequestLogsInput) (entity.RequestLogPage, error)
	GetUserRequestLog(ctx context.Context, userID int64, id int64) (entity.RequestLogDetail, error)
	ExportUserRequestLogs(ctx context.Context, userID int64, input entity.ListRequestLogsInput) ([]entity.RequestLog, error)
	ListModels(ctx context.Context) ([]entity.Model, error)
}

type RequestLogWriter interface {
	CreateRequestLog(ctx context.Context, input entity.CreateRequestLogInput) (int64, error)
	DeductUserWalletUsage(ctx context.Context, input entity.UserWalletUsageDebitInput) error
}

type PublicModelStore interface {
	ListEnabledModels(ctx context.Context) ([]entity.Model, error)
	ResolveModelRoute(ctx context.Context, publicName string) (entity.ModelRoute, error)
}

type ClientAuthStore interface {
	AuthenticateClientAPIKey(ctx context.Context, rawKey string) (entity.ClientAPIKey, error)
}
