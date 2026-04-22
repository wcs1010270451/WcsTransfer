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
	ListTenants(ctx context.Context) ([]entity.Tenant, error)
	CreateTenant(ctx context.Context, input entity.CreateTenantInput) (entity.Tenant, error)
	UpdateTenant(ctx context.Context, input entity.UpdateTenantInput) (entity.Tenant, error)
	ListTenantUsers(ctx context.Context, tenantID int64) ([]entity.TenantUser, error)
	CreateTenantUser(ctx context.Context, input entity.CreateTenantUserInput) (entity.TenantUser, error)
	UpdateTenantUserStatus(ctx context.Context, input entity.UpdateTenantUserStatusInput) (entity.TenantUser, error)
	ResetTenantUserPassword(ctx context.Context, input entity.ResetTenantUserPasswordInput) error
	AdjustTenantWallet(ctx context.Context, input entity.TenantWalletAdjustmentInput) (entity.Tenant, error)
	CorrectTenantWallet(ctx context.Context, input entity.TenantWalletCorrectionInput) (entity.Tenant, error)
	ListTenantWalletLedger(ctx context.Context, tenantID int64, page int, pageSize int) (entity.TenantWalletLedgerPage, error)
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
	ExportTenantRequestLogs(ctx context.Context, tenantID int64, input entity.ListRequestLogsInput) ([]entity.RequestLog, error)
	CreateAdminActionLog(ctx context.Context, input entity.CreateAdminActionLogInput) error
	GetTenantBillingReconciliation(ctx context.Context) ([]entity.TenantBillingReconciliation, error)
	GetProviderRequestAnomalies(ctx context.Context, since time.Time, minRequests int, rateLimitedThreshold float64, serverErrorThreshold float64) ([]entity.ProviderRequestAnomaly, error)
	GetTenantWalletBlockAnomalies(ctx context.Context, since time.Time, walletBlockThreshold int, reserveBlockThreshold int) ([]entity.TenantWalletBlockAnomaly, error)
	GetTenantBillingDebitAnomalies(ctx context.Context, since time.Time, minCount int, minBillableAmount float64) ([]entity.TenantBillingDebitAnomaly, error)
}

type AdminAuthStore interface {
	AuthenticateAdminUser(ctx context.Context, username string, password string) (entity.AdminUser, error)
	UpdateAdminUserLastLogin(ctx context.Context, userID int64) error
	GetAdminUserByID(ctx context.Context, userID int64) (entity.AdminUser, error)
}

type TenantAuthStore interface {
	RegisterTenantUser(ctx context.Context, input entity.RegisterTenantUserInput) (entity.TenantUser, error)
	AuthenticateTenantUser(ctx context.Context, email string, password string) (entity.TenantUser, error)
	UpdateTenantUserLastLogin(ctx context.Context, userID int64) error
	GetTenantUserByID(ctx context.Context, userID int64) (entity.TenantUser, error)
	GetTenantByID(ctx context.Context, tenantID int64) (entity.Tenant, error)
}

type TenantClientKeyStore interface {
	ListTenantClientAPIKeys(ctx context.Context, tenantID int64) ([]entity.ClientAPIKey, error)
	CreateTenantClientAPIKey(ctx context.Context, input entity.CreateClientAPIKeyInput) (entity.ClientAPIKey, error)
	DisableTenantClientAPIKey(ctx context.Context, tenantID int64, id int64) (entity.ClientAPIKey, error)
	GetTenantPortalStats(ctx context.Context, tenantID int64) (entity.TenantPortalStats, error)
	ListTenantWalletLedger(ctx context.Context, tenantID int64, page int, pageSize int) (entity.TenantWalletLedgerPage, error)
	ListTenantRequestLogs(ctx context.Context, tenantID int64, input entity.ListRequestLogsInput) (entity.RequestLogPage, error)
	GetTenantRequestLog(ctx context.Context, tenantID int64, id int64) (entity.RequestLogDetail, error)
	ExportTenantRequestLogs(ctx context.Context, tenantID int64, input entity.ListRequestLogsInput) ([]entity.RequestLog, error)
	ListTenantModels(ctx context.Context, tenantID int64) ([]entity.Model, error)
}

type RequestLogWriter interface {
	CreateRequestLog(ctx context.Context, input entity.CreateRequestLogInput) (int64, error)
	DeductTenantWalletUsage(ctx context.Context, input entity.TenantWalletUsageDebitInput) error
}

type PublicModelStore interface {
	ListEnabledModels(ctx context.Context) ([]entity.Model, error)
	ResolveModelRoute(ctx context.Context, publicName string) (entity.ModelRoute, error)
}

type ClientAuthStore interface {
	AuthenticateClientAPIKey(ctx context.Context, rawKey string) (entity.ClientAPIKey, error)
}
