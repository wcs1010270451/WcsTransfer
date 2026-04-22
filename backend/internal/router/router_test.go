package router

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wcstransfer/backend/internal/config"
	"wcstransfer/backend/internal/entity"
	"wcstransfer/backend/internal/platform"
	adminauthsvc "wcstransfer/backend/internal/service/adminauth"
)

type stubStore struct {
	providers       []entity.Provider
	tenants         []entity.Tenant
	tenantUsers     []entity.TenantUser
	walletLedger    []entity.TenantWalletLedgerEntry
	clientKeys      []entity.ClientAPIKey
	providerKeys    []entity.ProviderKey
	models          []entity.Model
	requestLogs     []entity.RequestLog
	logDetails      map[int64]entity.RequestLogDetail
	createdLogs     []entity.CreateRequestLogInput
	adminActionLogs []entity.CreateAdminActionLogInput
	reconciliation  []entity.TenantBillingReconciliation
	dashboard       entity.DashboardStats
}

func (s *stubStore) ListProviders(context.Context) ([]entity.Provider, error) {
	return s.providers, nil
}

func (s *stubStore) CreateProvider(_ context.Context, input entity.CreateProviderInput) (entity.Provider, error) {
	item := entity.Provider{
		ID:           int64(len(s.providers) + 1),
		Name:         input.Name,
		Slug:         input.Slug,
		ProviderType: input.ProviderType,
		BaseURL:      input.BaseURL,
		Status:       input.Status,
		Description:  input.Description,
		ExtraConfig:  input.ExtraConfig,
	}
	s.providers = append(s.providers, item)
	return item, nil
}

func (s *stubStore) UpdateProvider(_ context.Context, input entity.UpdateProviderInput) (entity.Provider, error) {
	for index, item := range s.providers {
		if item.ID == input.ID {
			item.Name = input.Name
			item.Slug = input.Slug
			item.ProviderType = input.ProviderType
			item.BaseURL = input.BaseURL
			item.Status = input.Status
			item.Description = input.Description
			item.ExtraConfig = input.ExtraConfig
			s.providers[index] = item
			return item, nil
		}
	}

	return entity.Provider{}, context.Canceled
}

func (s *stubStore) ListTenants(context.Context) ([]entity.Tenant, error) {
	return s.tenants, nil
}

func (s *stubStore) CreateTenant(_ context.Context, input entity.CreateTenantInput) (entity.Tenant, error) {
	item := entity.Tenant{
		ID:                  int64(len(s.tenants) + 1),
		Name:                input.Name,
		Slug:                input.Slug,
		Status:              input.Status,
		MaxClientKeys:       input.MaxClientKeys,
		MinAvailableBalance: input.MinAvailableBalance,
		Notes:               input.Notes,
	}
	s.tenants = append([]entity.Tenant{item}, s.tenants...)
	return item, nil
}

func (s *stubStore) UpdateTenant(_ context.Context, input entity.UpdateTenantInput) (entity.Tenant, error) {
	for index, item := range s.tenants {
		if item.ID == input.ID {
			item.Name = input.Name
			item.Slug = input.Slug
			item.Status = input.Status
			item.MaxClientKeys = input.MaxClientKeys
			item.MinAvailableBalance = input.MinAvailableBalance
			item.Notes = input.Notes
			s.tenants[index] = item
			return item, nil
		}
	}

	return entity.Tenant{}, context.Canceled
}

func (s *stubStore) ListTenantUsers(_ context.Context, tenantID int64) ([]entity.TenantUser, error) {
	items := make([]entity.TenantUser, 0)
	for _, item := range s.tenantUsers {
		if item.TenantID == tenantID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *stubStore) CreateTenantUser(_ context.Context, input entity.CreateTenantUserInput) (entity.TenantUser, error) {
	item := entity.TenantUser{
		ID:       int64(len(s.tenantUsers) + 1),
		TenantID: input.TenantID,
		Email:    input.Email,
		FullName: input.FullName,
		Status:   input.Status,
	}
	for _, tenant := range s.tenants {
		if tenant.ID == input.TenantID {
			item.TenantName = tenant.Name
			break
		}
	}
	s.tenantUsers = append([]entity.TenantUser{item}, s.tenantUsers...)
	return item, nil
}

func (s *stubStore) UpdateTenantUserStatus(_ context.Context, input entity.UpdateTenantUserStatusInput) (entity.TenantUser, error) {
	for index, item := range s.tenantUsers {
		if item.ID == input.UserID && item.TenantID == input.TenantID {
			item.Status = input.Status
			s.tenantUsers[index] = item
			return item, nil
		}
	}
	return entity.TenantUser{}, context.Canceled
}

func (s *stubStore) ResetTenantUserPassword(_ context.Context, input entity.ResetTenantUserPasswordInput) error {
	for _, item := range s.tenantUsers {
		if item.ID == input.UserID && item.TenantID == input.TenantID {
			return nil
		}
	}
	return context.Canceled
}

func (s *stubStore) AdjustTenantWallet(_ context.Context, input entity.TenantWalletAdjustmentInput) (entity.Tenant, error) {
	for index, item := range s.tenants {
		if item.ID == input.TenantID {
			before := item.WalletBalance
			item.WalletBalance += input.Amount
			s.tenants[index] = item
			s.walletLedger = append([]entity.TenantWalletLedgerEntry{{
				ID:             int64(len(s.walletLedger) + 1),
				TenantID:       item.ID,
				Direction:      "credit",
				Amount:         input.Amount,
				BalanceBefore:  before,
				BalanceAfter:   item.WalletBalance,
				Note:           input.Note,
				OperatorType:   "admin",
				OperatorUserID: input.OperatorID,
				CreatedAt:      time.Now(),
			}}, s.walletLedger...)
			return item, nil
		}
	}
	return entity.Tenant{}, context.Canceled
}

func (s *stubStore) CorrectTenantWallet(_ context.Context, input entity.TenantWalletCorrectionInput) (entity.Tenant, error) {
	for index, item := range s.tenants {
		if item.ID == input.TenantID {
			before := item.WalletBalance
			amount := input.Amount
			direction := "credit"
			if amount < 0 {
				direction = "debit"
				if before+amount < 0 {
					return entity.Tenant{}, context.Canceled
				}
			}
			item.WalletBalance += amount
			s.tenants[index] = item
			s.walletLedger = append([]entity.TenantWalletLedgerEntry{{
				ID:             int64(len(s.walletLedger) + 1),
				TenantID:       item.ID,
				Direction:      direction,
				Amount:         math.Abs(amount),
				BalanceBefore:  before,
				BalanceAfter:   item.WalletBalance,
				Note:           input.Note,
				OperatorType:   "admin",
				OperatorUserID: input.OperatorID,
				CreatedAt:      time.Now(),
			}}, s.walletLedger...)
			return item, nil
		}
	}
	return entity.Tenant{}, context.Canceled
}

func (s *stubStore) ListTenantWalletLedger(_ context.Context, tenantID int64, page int, pageSize int) (entity.TenantWalletLedgerPage, error) {
	filtered := make([]entity.TenantWalletLedgerEntry, 0)
	for _, item := range s.walletLedger {
		if item.TenantID == tenantID {
			filtered = append(filtered, item)
		}
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return entity.TenantWalletLedgerPage{
		Items:    filtered[start:end],
		Total:    int64(len(filtered)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *stubStore) ListClientAPIKeys(context.Context) ([]entity.ClientAPIKey, error) {
	return s.clientKeys, nil
}

func (s *stubStore) CreateClientAPIKey(_ context.Context, input entity.CreateClientAPIKeyInput) (entity.ClientAPIKey, error) {
	item := entity.ClientAPIKey{
		ID:                int64(len(s.clientKeys) + 1),
		Name:              input.Name,
		MaskedKey:         "wcs_live_abc...1234",
		PlainAPIKey:       "wcs_live_plain_test_key",
		Status:            input.Status,
		Description:       input.Description,
		RPMLimit:          input.RPMLimit,
		DailyRequestLimit: input.DailyRequestLimit,
		DailyTokenLimit:   input.DailyTokenLimit,
		DailyCostLimit:    input.DailyCostLimit,
		MonthlyCostLimit:  input.MonthlyCostLimit,
		WarningThreshold:  input.WarningThreshold,
		AllowedModelIDs:   input.AllowedModelIDs,
		ExpiresAt:         input.ExpiresAt,
	}
	item.AllowedModels = s.allowedModelNames(input.AllowedModelIDs)
	s.clientKeys = append(s.clientKeys, item)
	return item, nil
}

func (s *stubStore) UpdateClientAPIKey(_ context.Context, input entity.UpdateClientAPIKeyInput) (entity.ClientAPIKey, error) {
	for index, item := range s.clientKeys {
		if item.ID == input.ID {
			item.Name = input.Name
			item.Status = input.Status
			item.Description = input.Description
			item.RPMLimit = input.RPMLimit
			item.DailyRequestLimit = input.DailyRequestLimit
			item.DailyTokenLimit = input.DailyTokenLimit
			item.DailyCostLimit = input.DailyCostLimit
			item.MonthlyCostLimit = input.MonthlyCostLimit
			item.WarningThreshold = input.WarningThreshold
			item.AllowedModelIDs = input.AllowedModelIDs
			item.AllowedModels = s.allowedModelNames(input.AllowedModelIDs)
			item.ExpiresAt = input.ExpiresAt
			s.clientKeys[index] = item
			return item, nil
		}
	}

	return entity.ClientAPIKey{}, context.Canceled
}

func (s *stubStore) ListProviderKeys(context.Context) ([]entity.ProviderKey, error) {
	return s.providerKeys, nil
}

func (s *stubStore) CreateProviderKey(_ context.Context, input entity.CreateProviderKeyInput) (entity.ProviderKey, error) {
	item := entity.ProviderKey{
		ID:           int64(len(s.providerKeys) + 1),
		ProviderID:   input.ProviderID,
		ProviderName: "stub-provider",
		Name:         input.Name,
		Status:       input.Status,
		Weight:       input.Weight,
		Priority:     input.Priority,
		RPMLimit:     input.RPMLimit,
		TPMLimit:     input.TPMLimit,
		MaskedAPIKey: "sk-t***1234",
	}
	s.providerKeys = append(s.providerKeys, item)
	return item, nil
}

func (s *stubStore) UpdateProviderKey(_ context.Context, input entity.UpdateProviderKeyInput) (entity.ProviderKey, error) {
	for index, item := range s.providerKeys {
		if item.ID == input.ID {
			item.ProviderID = input.ProviderID
			item.Name = input.Name
			item.Status = input.Status
			item.Weight = input.Weight
			item.Priority = input.Priority
			item.RPMLimit = input.RPMLimit
			item.TPMLimit = input.TPMLimit
			if input.APIKey != nil && strings.TrimSpace(*input.APIKey) != "" {
				item.MaskedAPIKey = "sk-u***date"
			}
			s.providerKeys[index] = item
			return item, nil
		}
	}

	return entity.ProviderKey{}, context.Canceled
}

func (s *stubStore) ListModels(context.Context) ([]entity.Model, error) {
	return s.models, nil
}

func (s *stubStore) CreateModel(_ context.Context, input entity.CreateModelInput) (entity.Model, error) {
	item := entity.Model{
		ID:              int64(len(s.models) + 1),
		PublicName:      input.PublicName,
		ProviderID:      input.ProviderID,
		ProviderName:    "stub-provider",
		UpstreamModel:   input.UpstreamModel,
		RouteStrategy:   input.RouteStrategy,
		IsEnabled:       input.IsEnabled,
		MaxTokens:       input.MaxTokens,
		Temperature:     input.Temperature,
		TimeoutSeconds:  input.TimeoutSeconds,
		CostInputPer1M:  input.CostInputPer1M,
		CostOutputPer1M: input.CostOutputPer1M,
		SaleInputPer1M:  input.SaleInputPer1M,
		SaleOutputPer1M: input.SaleOutputPer1M,
		Metadata:        input.Metadata,
	}
	s.models = append(s.models, item)
	return item, nil
}

func (s *stubStore) UpdateModel(_ context.Context, input entity.UpdateModelInput) (entity.Model, error) {
	for index, item := range s.models {
		if item.ID == input.ID {
			item.PublicName = input.PublicName
			item.ProviderID = input.ProviderID
			item.ProviderName = "stub-provider"
			item.UpstreamModel = input.UpstreamModel
			item.RouteStrategy = input.RouteStrategy
			item.IsEnabled = input.IsEnabled
			item.MaxTokens = input.MaxTokens
			item.Temperature = input.Temperature
			item.TimeoutSeconds = input.TimeoutSeconds
			item.CostInputPer1M = input.CostInputPer1M
			item.CostOutputPer1M = input.CostOutputPer1M
			item.SaleInputPer1M = input.SaleInputPer1M
			item.SaleOutputPer1M = input.SaleOutputPer1M
			item.Metadata = input.Metadata
			s.models[index] = item
			return item, nil
		}
	}

	return entity.Model{}, context.Canceled
}

func (s *stubStore) ListEnabledModels(context.Context) ([]entity.Model, error) {
	items := make([]entity.Model, 0)
	for _, item := range s.models {
		if item.IsEnabled {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *stubStore) ResolveModelRoute(_ context.Context, publicName string) (entity.ModelRoute, error) {
	for _, item := range s.models {
		if item.PublicName == publicName && item.IsEnabled {
			return entity.ModelRoute{
				Model: item,
				Provider: entity.Provider{
					ID:      item.ProviderID,
					Name:    item.ProviderName,
					BaseURL: "https://example.com/v1",
					Status:  "active",
				},
				Keys: []entity.ProviderKey{
					{
						ID:         1,
						ProviderID: item.ProviderID,
						Name:       "default",
						APIKey:     "sk-test-secret",
						Status:     "active",
						Priority:   100,
						Weight:     100,
					},
				},
			}, nil
		}
	}

	return entity.ModelRoute{}, context.Canceled
}

func (s *stubStore) AuthenticateClientAPIKey(_ context.Context, rawKey string) (entity.ClientAPIKey, error) {
	trimmed := strings.TrimSpace(rawKey)
	for _, item := range s.clientKeys {
		if item.PlainAPIKey == trimmed && item.Status == "active" {
			if len(item.AllowedModels) == 0 && len(item.AllowedModelIDs) > 0 {
				item.AllowedModels = s.allowedModelNames(item.AllowedModelIDs)
			}
			if item.TenantID > 0 {
				for _, tenant := range s.tenants {
					if tenant.ID == item.TenantID {
						item.TenantWalletBalance = tenant.WalletBalance
						item.TenantMinAvailableBalance = tenant.MinAvailableBalance
						break
					}
				}
			}
			return item, nil
		}
	}

	return entity.ClientAPIKey{}, context.Canceled
}

func (s *stubStore) RegisterTenantUser(_ context.Context, input entity.RegisterTenantUserInput) (entity.TenantUser, error) {
	tenant := entity.Tenant{
		ID:                  int64(len(s.tenants) + 1),
		Name:                input.TenantName,
		Slug:                input.TenantSlug,
		Status:              "pending",
		MaxClientKeys:       0,
		MinAvailableBalance: 0.01,
	}
	s.tenants = append(s.tenants, tenant)
	return entity.TenantUser{
		ID:         int64(len(s.clientKeys) + len(s.tenants)),
		TenantID:   tenant.ID,
		TenantName: tenant.Name,
		Email:      input.Email,
		FullName:   input.FullName,
		Status:     "active",
	}, nil
}

func (s *stubStore) AuthenticateTenantUser(_ context.Context, email string, _ string) (entity.TenantUser, error) {
	trimmed := strings.TrimSpace(email)
	for _, tenant := range s.tenants {
		return entity.TenantUser{
			ID:         1,
			TenantID:   tenant.ID,
			TenantName: tenant.Name,
			Email:      trimmed,
			FullName:   "Stub User",
			Status:     "active",
		}, nil
	}
	return entity.TenantUser{}, context.Canceled
}

func (s *stubStore) UpdateTenantUserLastLogin(_ context.Context, _ int64) error {
	return nil
}

func (s *stubStore) GetTenantUserByID(_ context.Context, userID int64) (entity.TenantUser, error) {
	if len(s.tenants) == 0 {
		return entity.TenantUser{}, context.Canceled
	}
	return entity.TenantUser{
		ID:         userID,
		TenantID:   s.tenants[0].ID,
		TenantName: s.tenants[0].Name,
		Email:      "stub@example.com",
		FullName:   "Stub User",
		Status:     "active",
	}, nil
}

func (s *stubStore) GetTenantByID(_ context.Context, tenantID int64) (entity.Tenant, error) {
	for _, item := range s.tenants {
		if item.ID == tenantID {
			return item, nil
		}
	}
	return entity.Tenant{}, context.Canceled
}

func (s *stubStore) ListRequestLogs(_ context.Context, input entity.ListRequestLogsInput) (entity.RequestLogPage, error) {
	filtered := make([]entity.RequestLog, 0)
	for _, item := range s.requestLogs {
		if input.ProviderID > 0 && item.ProviderID != input.ProviderID {
			continue
		}
		if input.ModelPublicName != "" && item.ModelPublicName != input.ModelPublicName {
			continue
		}
		if input.Success != nil && item.Success != *input.Success {
			continue
		}
		if input.HTTPStatus > 0 && item.HTTPStatus != input.HTTPStatus {
			continue
		}
		if input.TraceID != "" && !strings.Contains(item.TraceID, input.TraceID) {
			continue
		}
		filtered = append(filtered, item)
	}

	page := input.Page
	if page <= 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	return entity.RequestLogPage{
		Items:    filtered[start:end],
		Total:    int64(len(filtered)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *stubStore) allowedModelNames(ids []int64) []string {
	if len(ids) == 0 {
		return nil
	}

	items := make([]string, 0, len(ids))
	for _, id := range ids {
		for _, model := range s.models {
			if model.ID == id {
				items = append(items, model.PublicName)
				break
			}
		}
	}
	return items
}

func (s *stubStore) GetRequestLog(_ context.Context, id int64) (entity.RequestLogDetail, error) {
	if item, ok := s.logDetails[id]; ok {
		return item, nil
	}

	return entity.RequestLogDetail{}, context.Canceled
}

func (s *stubStore) ExportRequestLogs(_ context.Context, input entity.ListRequestLogsInput) ([]entity.RequestLog, error) {
	page, err := s.ListRequestLogs(context.Background(), input)
	if err != nil {
		return nil, err
	}

	return page.Items, nil
}

func (s *stubStore) ExportTenantRequestLogs(_ context.Context, tenantID int64, input entity.ListRequestLogsInput) ([]entity.RequestLog, error) {
	input.TenantID = tenantID
	return s.ExportRequestLogs(context.Background(), input)
}

func (s *stubStore) CreateRequestLog(_ context.Context, input entity.CreateRequestLogInput) (int64, error) {
	s.createdLogs = append(s.createdLogs, input)
	return int64(len(s.createdLogs)), nil
}

func (s *stubStore) DeductTenantWalletUsage(_ context.Context, input entity.TenantWalletUsageDebitInput) error {
	for index, item := range s.clientKeys {
		if item.ID == input.ClientAPIKeyID {
			before := item.TenantWalletBalance
			item.TenantWalletBalance -= input.BillableAmount
			if item.TenantWalletBalance < 0 {
				item.TenantWalletBalance = 0
			}
			s.clientKeys[index] = item
			if item.TenantID > 0 {
				s.walletLedger = append([]entity.TenantWalletLedgerEntry{{
					ID:              int64(len(s.walletLedger) + 1),
					TenantID:        item.TenantID,
					Direction:       "debit",
					Amount:          input.BillableAmount,
					BalanceBefore:   before,
					BalanceAfter:    item.TenantWalletBalance,
					Note:            input.Note,
					OperatorType:    "system",
					RequestLogID:    input.RequestLogID,
					TraceID:         input.TraceID,
					ModelPublicName: input.ModelPublicName,
					TotalTokens:     int64(input.TotalTokens),
					CostAmount:      input.CostAmount,
					BillableAmount:  input.BillableAmount,
					CreatedAt:       time.Now(),
				}}, s.walletLedger...)
			}
			return nil
		}
	}
	return nil
}

func (s *stubStore) GetDashboardStats(context.Context) (entity.DashboardStats, error) {
	return s.dashboard, nil
}

func (s *stubStore) GetTenantBillingReconciliation(context.Context) ([]entity.TenantBillingReconciliation, error) {
	return s.reconciliation, nil
}

func (s *stubStore) GetProviderRequestAnomalies(context.Context, time.Time, int, float64, float64) ([]entity.ProviderRequestAnomaly, error) {
	return nil, nil
}

func (s *stubStore) GetTenantWalletBlockAnomalies(context.Context, time.Time, int, int) ([]entity.TenantWalletBlockAnomaly, error) {
	return nil, nil
}

func (s *stubStore) GetTenantBillingDebitAnomalies(context.Context, time.Time, int, float64) ([]entity.TenantBillingDebitAnomaly, error) {
	return nil, nil
}

func (s *stubStore) CreateAdminActionLog(_ context.Context, input entity.CreateAdminActionLogInput) error {
	s.adminActionLogs = append(s.adminActionLogs, input)
	return nil
}

func (s *stubStore) ListTenantClientAPIKeys(_ context.Context, tenantID int64) ([]entity.ClientAPIKey, error) {
	items := make([]entity.ClientAPIKey, 0)
	for _, item := range s.clientKeys {
		if item.TenantID == tenantID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *stubStore) CreateTenantClientAPIKey(_ context.Context, input entity.CreateClientAPIKeyInput) (entity.ClientAPIKey, error) {
	return s.CreateClientAPIKey(context.Background(), input)
}

func (s *stubStore) DisableTenantClientAPIKey(_ context.Context, tenantID int64, id int64) (entity.ClientAPIKey, error) {
	for index, item := range s.clientKeys {
		if item.ID == id && item.TenantID == tenantID {
			item.Status = "disabled"
			s.clientKeys[index] = item
			return item, nil
		}
	}
	return entity.ClientAPIKey{}, context.Canceled
}

func TestPublicRoutes(t *testing.T) {
	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{
		clientKeys: []entity.ClientAPIKey{
			{ID: 7, Name: "integration-client", PlainAPIKey: "wcs_live_proxy_test", Status: "active"},
		},
		models: []entity.Model{
			{PublicName: "gpt-4o-mini", ProviderName: "stub-provider", IsEnabled: true},
		},
	}

	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	testCases := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "healthz", method: http.MethodGet, path: "/healthz", status: http.StatusOK},
		{name: "version", method: http.MethodGet, path: "/version", status: http.StatusOK},
		{name: "models", method: http.MethodGet, path: "/v1/models", status: http.StatusOK},
		{name: "chat completions invalid request", method: http.MethodPost, path: "/v1/chat/completions", status: http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, tc.path, nil)

			engine.ServeHTTP(recorder, request)

			if recorder.Code != tc.status {
				t.Fatalf("expected status %d, got %d", tc.status, recorder.Code)
			}
		})
	}
}

func TestAdminRoutesRequireTokenWhenConfigured(t *testing.T) {
	cfg := config.Config{
		AppName:         "wcstransfer-gateway",
		Env:             "test",
		GinMode:         "test",
		HTTPPort:        "8080",
		AuthTokenSecret: "admin-auth-secret-for-test",
	}

	store := &stubStore{}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/providers", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestAdminRoutesAllowTokenWhenProvided(t *testing.T) {
	cfg := config.Config{
		AppName:         "wcstransfer-gateway",
		Env:             "test",
		GinMode:         "test",
		HTTPPort:        "8080",
		AuthTokenSecret: "admin-auth-secret-for-test",
	}

	store := &stubStore{
		providers: []entity.Provider{
			{ID: 1, Name: "OpenAI Compatible", Slug: "openai-compatible"},
		},
	}

	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})
	tokenService := adminauthsvc.New(cfg.AuthTokenSecret)
	token, err := tokenService.IssueToken(42, "ops-admin", "Ops Admin", time.Hour)
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/providers", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestAdminLogDetailRoute(t *testing.T) {
	cfg := config.Config{
		AppName:         "wcstransfer-gateway",
		Env:             "test",
		GinMode:         "test",
		HTTPPort:        "8080",
		AuthTokenSecret: "admin-auth-secret-for-test",
	}

	store := &stubStore{
		logDetails: map[int64]entity.RequestLogDetail{
			13: {
				RequestLog: entity.RequestLog{
					ID:               13,
					TraceID:          "trace-13",
					RequestType:      "chat.completions",
					ModelPublicName:  "qwen-max",
					HTTPStatus:       http.StatusBadRequest,
					Success:          false,
					PromptTokens:     12,
					CompletionTokens: 0,
					TotalTokens:      12,
					ReservedAmount:   0.25,
					CostAmount:       0.01,
					BillableAmount:   0.02,
				},
				RequestPayload:  json.RawMessage(`{"model":"qwen-max"}`),
				ResponsePayload: json.RawMessage(`{"error":"bad request"}`),
				Metadata:        json.RawMessage(`{"source":"test"}`),
			},
		},
	}

	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})
	tokenService := adminauthsvc.New(cfg.AuthTokenSecret)
	token, err := tokenService.IssueToken(42, "ops-admin", "Ops Admin", time.Hour)
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/logs/13", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "\"reserved_amount\":0.25") {
		t.Fatalf("expected reserved_amount in response, got %s", recorder.Body.String())
	}
}

func TestAdminTenantBillingReconciliationRoute(t *testing.T) {
	cfg := config.Config{
		AppName:         "wcstransfer-gateway",
		Env:             "test",
		GinMode:         "test",
		HTTPPort:        "8080",
		AuthTokenSecret: "admin-auth-secret-for-test",
	}

	store := &stubStore{
		reconciliation: []entity.TenantBillingReconciliation{
			{
				TenantID:           1,
				TenantName:         "tenant-a",
				WalletBalance:      10,
				LedgerCreditAmount: 12,
				LedgerDebitAmount:  2,
				LedgerNetAmount:    10,
				LogBillableAmount:  2,
				LogCostAmount:      1.5,
				IsWalletBalanced:   true,
				IsBillingBalanced:  true,
			},
		},
	}

	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})
	tokenService := adminauthsvc.New(cfg.AuthTokenSecret)
	token, err := tokenService.IssueToken(42, "ops-admin", "Ops Admin", time.Hour)
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/reconciliation/tenants", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "\"tenant_name\":\"tenant-a\"") {
		t.Fatalf("expected reconciliation payload, got %s", recorder.Body.String())
	}
}

func TestAdminWalletAdjustRecordsOperatorAndAuditLog(t *testing.T) {
	cfg := config.Config{
		AppName:         "wcstransfer-gateway",
		Env:             "test",
		GinMode:         "test",
		HTTPPort:        "8080",
		AuthTokenSecret: "admin-auth-secret-for-test",
	}

	store := &stubStore{
		tenants: []entity.Tenant{
			{ID: 1, Name: "Tenant A", Slug: "tenant-a", Status: "active", WalletBalance: 5},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	tokenService := adminauthsvc.New(cfg.AuthTokenSecret)
	token, err := tokenService.IssueToken(42, "ops-admin", "Ops Admin", time.Hour)
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}

	body := bytes.NewBufferString(`{"amount":10,"note":"manual top-up"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/tenants/1/wallet/adjust", body)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if len(store.walletLedger) != 1 {
		t.Fatalf("expected 1 wallet ledger entry, got %d", len(store.walletLedger))
	}
	if store.walletLedger[0].OperatorUserID != 42 {
		t.Fatalf("expected operator user id 42, got %d", store.walletLedger[0].OperatorUserID)
	}
	if len(store.adminActionLogs) != 1 {
		t.Fatalf("expected 1 admin action log, got %d", len(store.adminActionLogs))
	}
	if store.adminActionLogs[0].Action != "tenant.wallet.credit" {
		t.Fatalf("unexpected admin audit action: %+v", store.adminActionLogs[0])
	}
	if store.adminActionLogs[0].AdminUserID != 42 || store.adminActionLogs[0].AdminUsername != "ops-admin" {
		t.Fatalf("unexpected admin audit actor: %+v", store.adminActionLogs[0])
	}
	if store.adminActionLogs[0].RequestPath != "/admin/tenants/:id/wallet/adjust" {
		t.Fatalf("unexpected admin audit request path: %+v", store.adminActionLogs[0])
	}
}

func TestAdminWalletCorrectionRecordsOperatorAndAuditLog(t *testing.T) {
	cfg := config.Config{
		AppName:         "wcstransfer-gateway",
		Env:             "test",
		GinMode:         "test",
		HTTPPort:        "8080",
		AuthTokenSecret: "admin-auth-secret-for-test",
	}

	store := &stubStore{
		tenants: []entity.Tenant{
			{ID: 1, Name: "Tenant A", Slug: "tenant-a", Status: "active", WalletBalance: 5},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	tokenService := adminauthsvc.New(cfg.AuthTokenSecret)
	token, err := tokenService.IssueToken(42, "ops-admin", "Ops Admin", time.Hour)
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}

	body := bytes.NewBufferString(`{"amount":-1.25,"note":"manual reconciliation fix"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/tenants/1/wallet/correct", body)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if len(store.walletLedger) != 1 {
		t.Fatalf("expected 1 wallet ledger entry, got %d", len(store.walletLedger))
	}
	if store.walletLedger[0].Direction != "debit" || store.walletLedger[0].Amount != 1.25 {
		t.Fatalf("unexpected wallet ledger correction entry: %+v", store.walletLedger[0])
	}
	if store.walletLedger[0].OperatorUserID != 42 {
		t.Fatalf("expected operator user id 42, got %d", store.walletLedger[0].OperatorUserID)
	}
	if len(store.adminActionLogs) != 1 {
		t.Fatalf("expected 1 admin action log, got %d", len(store.adminActionLogs))
	}
	if store.adminActionLogs[0].Action != "tenant.wallet.reconcile" {
		t.Fatalf("unexpected admin audit action: %+v", store.adminActionLogs[0])
	}
	if store.adminActionLogs[0].RequestPath != "/admin/tenants/:id/wallet/correct" {
		t.Fatalf("unexpected admin audit request path: %+v", store.adminActionLogs[0])
	}
}

func TestPublicRoutesRequireClientAPIKeyWhenAuthStoreConfigured(t *testing.T) {
	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  store,
		Auth:   store,
		Public: store,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestPublicRoutesAllowClientAPIKeyWhenProvided(t *testing.T) {
	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{
		clientKeys: []entity.ClientAPIKey{
			{ID: 9, Name: "web-app", PlainAPIKey: "wcs_live_test_public", Status: "active"},
		},
		models: []entity.Model{
			{PublicName: "gpt-4o-mini", ProviderName: "stub-provider", IsEnabled: true},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  store,
		Auth:   store,
		Public: store,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	request.Header.Set("Authorization", "Bearer wcs_live_test_public")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestPublicModelsFilteredByClientAuthorization(t *testing.T) {
	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{
		models: []entity.Model{
			{ID: 1, PublicName: "gpt-4o-mini", ProviderName: "stub-provider", IsEnabled: true},
			{ID: 2, PublicName: "qwen-plus", ProviderName: "stub-provider", IsEnabled: true},
		},
		clientKeys: []entity.ClientAPIKey{
			{ID: 9, Name: "restricted-client", PlainAPIKey: "wcs_live_restricted", Status: "active", AllowedModelIDs: []int64{2}},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  store,
		Auth:   store,
		Public: store,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	request.Header.Set("Authorization", "Bearer wcs_live_restricted")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].ID != "qwen-plus" {
		t.Fatalf("expected only qwen-plus in model list, got %#v", payload.Data)
	}
}

func TestChatCompletionsRejectsUnauthorizedModel(t *testing.T) {
	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{
		models: []entity.Model{
			{ID: 1, PublicName: "gpt-4o-mini", ProviderName: "stub-provider", IsEnabled: true},
			{ID: 2, PublicName: "qwen-plus", ProviderName: "stub-provider", IsEnabled: true},
		},
		clientKeys: []entity.ClientAPIKey{
			{ID: 9, Name: "restricted-client", PlainAPIKey: "wcs_live_restricted", Status: "active", AllowedModelIDs: []int64{2}},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  store,
		Auth:   store,
		Public: store,
		Log:    store,
	})

	body, err := json.Marshal(map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]any{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer wcs_live_restricted")
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
	if len(store.createdLogs) != 1 {
		t.Fatalf("expected one request log, got %d", len(store.createdLogs))
	}
	if store.createdLogs[0].ErrorType != "model_forbidden" {
		t.Fatalf("expected model_forbidden log type, got %q", store.createdLogs[0].ErrorType)
	}
}

func TestChatCompletionsRejectsExceededBudget(t *testing.T) {
	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{
		models: []entity.Model{
			{ID: 1, PublicName: "gpt-4o-mini", ProviderName: "stub-provider", IsEnabled: true},
		},
		clientKeys: []entity.ClientAPIKey{
			{
				ID:             9,
				Name:           "budgeted-client",
				PlainAPIKey:    "wcs_live_budgeted",
				Status:         "active",
				DailyCostLimit: 1.5,
				CostUsage: &entity.ClientCostUsage{
					DailyCostUsed:      1.6,
					DailyCostRemaining: 0,
					IsDailyCostLimited: true,
				},
			},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  store,
		Auth:   store,
		Public: store,
		Log:    store,
	})

	body, err := json.Marshal(map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]any{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer wcs_live_budgeted")
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, recorder.Code)
	}
	if len(store.createdLogs) != 1 {
		t.Fatalf("expected one request log, got %d", len(store.createdLogs))
	}
	if store.createdLogs[0].ErrorType != "budget_exceeded" {
		t.Fatalf("expected budget_exceeded log type, got %q", store.createdLogs[0].ErrorType)
	}
}

func TestCreateProviderRoute(t *testing.T) {
	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	body, err := json.Marshal(map[string]any{
		"name":          "OpenAI Compatible",
		"slug":          "openai-compatible",
		"provider_type": "openai_compatible",
		"base_url":      "https://example.com/v1",
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/providers", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
}

func TestAdminStatsRoute(t *testing.T) {
	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{
		dashboard: entity.DashboardStats{
			WindowHours:       24,
			ProviderCount:     2,
			KeyCount:          3,
			ActiveKeyCount:    2,
			ModelCount:        4,
			EnabledModelCount: 3,
			RequestCount:      10,
			SuccessCount:      8,
			FailedCount:       2,
			SuccessRate:       80,
			AverageLatencyMS:  456.7,
			PromptTokens:      100,
			CompletionTokens:  200,
			TotalTokens:       300,
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "\"request_count\":10") {
		t.Fatalf("expected request_count in response body, got %s", recorder.Body.String())
	}
}

func TestUpdateProviderRoute(t *testing.T) {
	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		providers: []entity.Provider{
			{ID: 1, Name: "Old", Slug: "old", ProviderType: "openai_compatible", BaseURL: "https://old.example/v1", Status: "active"},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	body, err := json.Marshal(map[string]any{
		"name":          "New Name",
		"slug":          "new-name",
		"provider_type": "openai_compatible",
		"base_url":      "https://new.example/v1",
		"status":        "disabled",
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/providers/1", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if store.providers[0].Status != "disabled" {
		t.Fatalf("expected provider status to update, got %+v", store.providers[0])
	}
}

func TestUpdateKeyRoute(t *testing.T) {
	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		providerKeys: []entity.ProviderKey{
			{ID: 1, ProviderID: 1, ProviderName: "stub-provider", Name: "default", Status: "active", Weight: 100, Priority: 100, MaskedAPIKey: "sk-t***1234"},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	body, err := json.Marshal(map[string]any{
		"provider_id": 1,
		"name":        "backup",
		"status":      "disabled",
		"weight":      80,
		"priority":    20,
		"rpm_limit":   60,
		"tpm_limit":   1000,
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/keys/1", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if store.providerKeys[0].Status != "disabled" || store.providerKeys[0].Name != "backup" {
		t.Fatalf("expected key to update, got %+v", store.providerKeys[0])
	}
}

func TestUpdateModelRoute(t *testing.T) {
	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		clientKeys: []entity.ClientAPIKey{
			{ID: 7, Name: "integration-client", PlainAPIKey: "wcs_live_proxy_test", Status: "active"},
		},
		models: []entity.Model{
			{ID: 1, PublicName: "gpt-4o-mini", ProviderID: 1, ProviderName: "stub-provider", UpstreamModel: "gpt-4o-mini", RouteStrategy: "fixed", IsEnabled: true},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	body, err := json.Marshal(map[string]any{
		"public_name":     "qwen-plus",
		"provider_id":     1,
		"upstream_model":  "qwen-plus-2025-11-25",
		"route_strategy":  "failover",
		"is_enabled":      false,
		"max_tokens":      2048,
		"temperature":     0.2,
		"timeout_seconds": 90,
		"metadata":        map[string]any{"tier": "gold"},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/models/1", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if store.models[0].PublicName != "qwen-plus" || store.models[0].IsEnabled {
		t.Fatalf("expected model to update, got %+v", store.models[0])
	}
}

func TestListLogsRouteWithPaginationAndFilters(t *testing.T) {
	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		requestLogs: []entity.RequestLog{
			{ID: 1, TraceID: "trace-a", ModelPublicName: "qwen-plus", ProviderID: 1, ProviderName: "Bailian", HTTPStatus: 200, Success: true, CreatedAt: time.Now()},
			{ID: 2, TraceID: "trace-b", ModelPublicName: "gpt-4o-mini", ProviderID: 2, ProviderName: "OpenAI", HTTPStatus: 500, Success: false, CreatedAt: time.Now()},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/logs?page=1&page_size=10&provider_id=1&success=true", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "\"total\":1") || !strings.Contains(recorder.Body.String(), "\"trace_id\":\"trace-a\"") {
		t.Fatalf("unexpected logs response: %s", recorder.Body.String())
	}
}

func TestGetLogDetailRoute(t *testing.T) {
	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		logDetails: map[int64]entity.RequestLogDetail{
			1: {
				RequestLog: entity.RequestLog{
					ID:              1,
					TraceID:         "trace-a",
					ModelPublicName: "qwen-plus",
					ProviderName:    "Bailian",
				},
				RequestPayload:  json.RawMessage(`{"model":"qwen-plus"}`),
				ResponsePayload: json.RawMessage(`{"id":"chatcmpl-1"}`),
				Metadata:        json.RawMessage(`{"stream":false}`),
			},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/logs/1", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "\"request_payload\":{\"model\":\"qwen-plus\"}") {
		t.Fatalf("unexpected log detail response: %s", recorder.Body.String())
	}
}

func TestExportLogsRoute(t *testing.T) {
	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	success := true
	store := &stubStore{
		requestLogs: []entity.RequestLog{
			{ID: 1, TraceID: "trace-a", ModelPublicName: "qwen-plus", ProviderName: "Bailian", ProviderKeyName: "primary", HTTPStatus: 200, Success: true, CreatedAt: time.Now()},
			{ID: 2, TraceID: "trace-b", ModelPublicName: "gpt-4o-mini", ProviderName: "OpenAI", ProviderKeyName: "backup", HTTPStatus: 500, Success: false, CreatedAt: time.Now()},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{Admin: store, Public: store})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/logs/export?success=true", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "trace-a") || strings.Contains(recorder.Body.String(), "trace-b") {
		t.Fatalf("unexpected csv export response: %s", recorder.Body.String())
	}
	if success == false {
		t.Fatalf("success filter should remain true")
	}
}

func TestChatCompletionsProxyRoute(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test-secret" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream body: %v", err)
		}

		if got := payload["model"]; got != "gpt-4o-mini-upstream" {
			t.Fatalf("unexpected upstream model: %v", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
		})
	}))
	defer upstream.Close()

	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{
		clientKeys: []entity.ClientAPIKey{
			{ID: 7, Name: "integration-client", PlainAPIKey: "wcs_live_proxy_test", Status: "active"},
		},
		models: []entity.Model{
			{
				ID:             1,
				PublicName:     "gpt-4o-mini",
				ProviderID:     1,
				ProviderName:   "stub-provider",
				UpstreamModel:  "gpt-4o-mini-upstream",
				RouteStrategy:  "fixed",
				IsEnabled:      true,
				TimeoutSeconds: 30,
			},
		},
		providerKeys: []entity.ProviderKey{
			{ID: 1, ProviderID: 1, ProviderName: "stub-provider", Name: "primary", Status: "active", Weight: 100, Priority: 10, MaskedAPIKey: "sk-p***ary"},
			{ID: 2, ProviderID: 1, ProviderName: "stub-provider", Name: "backup", Status: "active", Weight: 100, Priority: 20, MaskedAPIKey: "sk-b***kup"},
		},
	}

	body, err := json.Marshal(map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	// Swap in an upstream-aware store route and HTTP client through the handler constructor path.
	store.models[0].ProviderName = "stub-provider"
	routeStore := &stubStoreWithUpstream{
		base:     store,
		upstream: upstream.URL + "/v1",
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  routeStore,
		Auth:   routeStore,
		Log:    routeStore,
		Public: routeStore,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer wcs_live_proxy_test")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !upstreamCalled {
		t.Fatalf("expected upstream server to be called")
	}
	if len(store.createdLogs) != 1 {
		t.Fatalf("expected 1 created log, got %d", len(store.createdLogs))
	}
	if store.createdLogs[0].ModelPublicName != "gpt-4o-mini" {
		t.Fatalf("unexpected logged model: %s", store.createdLogs[0].ModelPublicName)
	}
	if !store.createdLogs[0].Success {
		t.Fatalf("expected success log entry")
	}
	if store.createdLogs[0].ClientAPIKeyID != 7 {
		t.Fatalf("expected client api key id to be logged, got %+v", store.createdLogs[0])
	}
}

func TestChatCompletionsRejectsInsufficientWalletReserve(t *testing.T) {
	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{
		tenants: []entity.Tenant{
			{ID: 1, Name: "tenant-a", Slug: "tenant-a", Status: "active", MaxClientKeys: 1, WalletBalance: 0.05, MinAvailableBalance: 0.01},
		},
		clientKeys: []entity.ClientAPIKey{
			{ID: 7, TenantID: 1, Name: "tenant-client", PlainAPIKey: "wcs_live_low_balance", Status: "active"},
		},
		models: []entity.Model{
			{
				ID:              1,
				PublicName:      "gpt-4o-mini",
				ProviderID:      1,
				ProviderName:    "stub-provider",
				UpstreamModel:   "gpt-4o-mini-upstream",
				RouteStrategy:   "fixed",
				IsEnabled:       true,
				MaxTokens:       200000,
				SaleOutputPer1M: 10,
			},
		},
		providerKeys: []entity.ProviderKey{
			{ID: 1, ProviderID: 1, ProviderName: "stub-provider", Name: "primary", Status: "active", Weight: 100, Priority: 10, MaskedAPIKey: "sk-p***ary"},
		},
	}

	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  store,
		Auth:   store,
		Log:    store,
		Public: store,
	})

	body, err := json.Marshal(map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer wcs_live_low_balance")
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusPaymentRequired {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusPaymentRequired, recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "wallet_reserve_insufficient") {
		t.Fatalf("expected wallet_reserve_insufficient, got %s", recorder.Body.String())
	}
}

func TestChatCompletionsLogsAmounts(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-cost",
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "done"}},
			},
			"usage": map[string]any{
				"prompt_tokens":     1000,
				"completion_tokens": 2000,
				"total_tokens":      3000,
			},
		})
	}))
	defer upstream.Close()

	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		clientKeys: []entity.ClientAPIKey{
			{ID: 7, Name: "integration-client", PlainAPIKey: "wcs_live_proxy_test", Status: "active"},
		},
		models: []entity.Model{
			{
				ID:              1,
				PublicName:      "gpt-4o-mini",
				ProviderID:      1,
				ProviderName:    "stub-provider",
				UpstreamModel:   "gpt-4o-mini-upstream",
				RouteStrategy:   "fixed",
				IsEnabled:       true,
				TimeoutSeconds:  30,
				CostInputPer1M:  0.15,
				CostOutputPer1M: 0.60,
				SaleInputPer1M:  0.30,
				SaleOutputPer1M: 1.20,
			},
		},
	}

	routeStore := &stubStoreWithUpstream{
		base:     store,
		upstream: upstream.URL + "/v1",
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  routeStore,
		Auth:   routeStore,
		Log:    routeStore,
		Public: routeStore,
	})

	body, err := json.Marshal(map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]any{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer wcs_live_proxy_test")
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(store.createdLogs) != 1 {
		t.Fatalf("expected 1 created log, got %d", len(store.createdLogs))
	}
	if store.createdLogs[0].CostAmount <= 0 {
		t.Fatalf("expected positive cost amount, got %+v", store.createdLogs[0])
	}
	if store.createdLogs[0].BillableAmount <= 0 {
		t.Fatalf("expected positive billable amount, got %+v", store.createdLogs[0])
	}
}

func TestEmbeddingsProxyRoute(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream body: %v", err)
		}
		if got := payload["model"]; got != "text-embedding-upstream" {
			t.Fatalf("unexpected upstream model: %v", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"object": "embedding", "index": 0, "embedding": []float64{0.1, 0.2, 0.3}},
			},
			"usage": map[string]any{
				"prompt_tokens": 250,
				"total_tokens":  250,
			},
		})
	}))
	defer upstream.Close()

	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		clientKeys: []entity.ClientAPIKey{
			{ID: 7, Name: "integration-client", PlainAPIKey: "wcs_live_proxy_test", Status: "active"},
		},
		models: []entity.Model{
			{
				ID:              1,
				PublicName:      "text-embedding-3-small",
				ProviderID:      1,
				ProviderName:    "stub-provider",
				UpstreamModel:   "text-embedding-upstream",
				RouteStrategy:   "fixed",
				IsEnabled:       true,
				TimeoutSeconds:  30,
				CostInputPer1M:  0.02,
				CostOutputPer1M: 0,
				SaleInputPer1M:  0.02,
				SaleOutputPer1M: 0,
			},
		},
	}
	routeStore := &stubStoreWithUpstream{
		base:     store,
		upstream: upstream.URL + "/v1",
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  routeStore,
		Auth:   routeStore,
		Log:    routeStore,
		Public: routeStore,
	})

	body, err := json.Marshal(map[string]any{
		"model": "text-embedding-3-small",
		"input": "hello embeddings",
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer wcs_live_proxy_test")
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !upstreamCalled {
		t.Fatalf("expected upstream server to be called")
	}
	if len(store.createdLogs) != 1 {
		t.Fatalf("expected 1 created log, got %d", len(store.createdLogs))
	}
	if store.createdLogs[0].RequestType != "embeddings" {
		t.Fatalf("unexpected log request type: %+v", store.createdLogs[0])
	}
	if store.createdLogs[0].TotalTokens != 250 {
		t.Fatalf("expected total_tokens 250, got %+v", store.createdLogs[0])
	}
}

func TestChatCompletionsStreamProxyRoute(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestPayload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&requestPayload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		streamOptions, ok := requestPayload["stream_options"].(map[string]any)
		if !ok || streamOptions["include_usage"] != true {
			t.Fatalf("expected stream_options.include_usage=true, got %v", requestPayload["stream_options"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"id\":\"chunk-1\"}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = w.Write([]byte("data: {\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":34,\"total_tokens\":46}}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	cfg := config.Config{
		AppName:  "wcstransfer-gateway",
		Env:      "test",
		GinMode:  "test",
		HTTPPort: "8080",
	}

	store := &stubStore{
		models: []entity.Model{
			{
				ID:             1,
				PublicName:     "gpt-4o-mini",
				ProviderID:     1,
				ProviderName:   "stub-provider",
				UpstreamModel:  "gpt-4o-mini-upstream",
				RouteStrategy:  "fixed",
				IsEnabled:      true,
				TimeoutSeconds: 30,
			},
		},
		providerKeys: []entity.ProviderKey{
			{ID: 1, ProviderID: 1, ProviderName: "stub-provider", Name: "primary", Status: "active", Weight: 100, Priority: 10, MaskedAPIKey: "sk-p***ary"},
			{ID: 2, ProviderID: 1, ProviderName: "stub-provider", Name: "backup", Status: "active", Weight: 100, Priority: 20, MaskedAPIKey: "sk-b***kup"},
		},
	}

	body, err := json.Marshal(map[string]any{
		"model":  "gpt-4o-mini",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	routeStore := &stubStoreWithUpstream{
		base:     store,
		upstream: upstream.URL + "/v1",
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  routeStore,
		Log:    routeStore,
		Public: routeStore,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "data: {\"id\":\"chunk-1\"}") {
		t.Fatalf("expected streamed chunk in response body, got %s", recorder.Body.String())
	}
	if len(store.createdLogs) != 1 {
		t.Fatalf("expected 1 created log, got %d", len(store.createdLogs))
	}
	if !strings.Contains(string(store.createdLogs[0].Metadata), "\"stream_response\":true") {
		t.Fatalf("expected stream_response metadata, got %s", string(store.createdLogs[0].Metadata))
	}
	if store.createdLogs[0].PromptTokens != 12 || store.createdLogs[0].CompletionTokens != 34 || store.createdLogs[0].TotalTokens != 46 {
		t.Fatalf("unexpected logged usage: %+v", store.createdLogs[0])
	}
}

type stubStoreWithUpstream struct {
	base         *stubStore
	upstream     string
	keys         []entity.ProviderKey
	providerType string
	extraConfig  json.RawMessage
}

func (s *stubStoreWithUpstream) ListProviders(ctx context.Context) ([]entity.Provider, error) {
	return s.base.ListProviders(ctx)
}

func (s *stubStoreWithUpstream) CreateProvider(ctx context.Context, input entity.CreateProviderInput) (entity.Provider, error) {
	return s.base.CreateProvider(ctx, input)
}

func (s *stubStoreWithUpstream) UpdateProvider(ctx context.Context, input entity.UpdateProviderInput) (entity.Provider, error) {
	return s.base.UpdateProvider(ctx, input)
}

func (s *stubStoreWithUpstream) ListTenants(ctx context.Context) ([]entity.Tenant, error) {
	return s.base.ListTenants(ctx)
}

func (s *stubStoreWithUpstream) CreateTenant(ctx context.Context, input entity.CreateTenantInput) (entity.Tenant, error) {
	return s.base.CreateTenant(ctx, input)
}

func (s *stubStoreWithUpstream) UpdateTenant(ctx context.Context, input entity.UpdateTenantInput) (entity.Tenant, error) {
	return s.base.UpdateTenant(ctx, input)
}

func (s *stubStoreWithUpstream) ListTenantUsers(ctx context.Context, tenantID int64) ([]entity.TenantUser, error) {
	return s.base.ListTenantUsers(ctx, tenantID)
}

func (s *stubStoreWithUpstream) CreateTenantUser(ctx context.Context, input entity.CreateTenantUserInput) (entity.TenantUser, error) {
	return s.base.CreateTenantUser(ctx, input)
}

func (s *stubStoreWithUpstream) UpdateTenantUserStatus(ctx context.Context, input entity.UpdateTenantUserStatusInput) (entity.TenantUser, error) {
	return s.base.UpdateTenantUserStatus(ctx, input)
}

func (s *stubStoreWithUpstream) ResetTenantUserPassword(ctx context.Context, input entity.ResetTenantUserPasswordInput) error {
	return s.base.ResetTenantUserPassword(ctx, input)
}

func (s *stubStoreWithUpstream) AdjustTenantWallet(ctx context.Context, input entity.TenantWalletAdjustmentInput) (entity.Tenant, error) {
	return s.base.AdjustTenantWallet(ctx, input)
}

func (s *stubStoreWithUpstream) CorrectTenantWallet(ctx context.Context, input entity.TenantWalletCorrectionInput) (entity.Tenant, error) {
	return s.base.CorrectTenantWallet(ctx, input)
}

func (s *stubStoreWithUpstream) ListTenantWalletLedger(ctx context.Context, tenantID int64, page int, pageSize int) (entity.TenantWalletLedgerPage, error) {
	return s.base.ListTenantWalletLedger(ctx, tenantID, page, pageSize)
}

func (s *stubStoreWithUpstream) ListClientAPIKeys(ctx context.Context) ([]entity.ClientAPIKey, error) {
	return s.base.ListClientAPIKeys(ctx)
}

func (s *stubStoreWithUpstream) CreateClientAPIKey(ctx context.Context, input entity.CreateClientAPIKeyInput) (entity.ClientAPIKey, error) {
	return s.base.CreateClientAPIKey(ctx, input)
}

func (s *stubStoreWithUpstream) UpdateClientAPIKey(ctx context.Context, input entity.UpdateClientAPIKeyInput) (entity.ClientAPIKey, error) {
	return s.base.UpdateClientAPIKey(ctx, input)
}

func (s *stubStoreWithUpstream) ListProviderKeys(ctx context.Context) ([]entity.ProviderKey, error) {
	return s.base.ListProviderKeys(ctx)
}

func (s *stubStoreWithUpstream) CreateProviderKey(ctx context.Context, input entity.CreateProviderKeyInput) (entity.ProviderKey, error) {
	return s.base.CreateProviderKey(ctx, input)
}

func (s *stubStoreWithUpstream) UpdateProviderKey(ctx context.Context, input entity.UpdateProviderKeyInput) (entity.ProviderKey, error) {
	return s.base.UpdateProviderKey(ctx, input)
}

func (s *stubStoreWithUpstream) ListModels(ctx context.Context) ([]entity.Model, error) {
	return s.base.ListModels(ctx)
}

func (s *stubStoreWithUpstream) CreateModel(ctx context.Context, input entity.CreateModelInput) (entity.Model, error) {
	return s.base.CreateModel(ctx, input)
}

func (s *stubStoreWithUpstream) UpdateModel(ctx context.Context, input entity.UpdateModelInput) (entity.Model, error) {
	return s.base.UpdateModel(ctx, input)
}

func (s *stubStoreWithUpstream) ListEnabledModels(ctx context.Context) ([]entity.Model, error) {
	return s.base.ListEnabledModels(ctx)
}

func (s *stubStoreWithUpstream) ResolveModelRoute(_ context.Context, publicName string) (entity.ModelRoute, error) {
	for _, item := range s.base.models {
		if item.PublicName == publicName && item.IsEnabled {
			keys := s.keys
			if len(keys) == 0 {
				keys = []entity.ProviderKey{
					{
						ID:         1,
						ProviderID: item.ProviderID,
						Name:       "default",
						APIKey:     "sk-test-secret",
						Status:     "active",
						Priority:   100,
						Weight:     100,
					},
				}
			}

			return entity.ModelRoute{
				Model: item,
				Provider: entity.Provider{
					ID:           item.ProviderID,
					Name:         item.ProviderName,
					BaseURL:      s.upstream,
					Status:       "active",
					ProviderType: defaultTestProviderType(s.providerType),
					ExtraConfig:  s.extraConfig,
				},
				Keys: keys,
			}, nil
		}
	}

	return entity.ModelRoute{}, context.Canceled
}

func (s *stubStoreWithUpstream) AuthenticateClientAPIKey(ctx context.Context, rawKey string) (entity.ClientAPIKey, error) {
	return s.base.AuthenticateClientAPIKey(ctx, rawKey)
}

func (s *stubStoreWithUpstream) RegisterTenantUser(ctx context.Context, input entity.RegisterTenantUserInput) (entity.TenantUser, error) {
	return s.base.RegisterTenantUser(ctx, input)
}

func (s *stubStoreWithUpstream) AuthenticateTenantUser(ctx context.Context, email string, password string) (entity.TenantUser, error) {
	return s.base.AuthenticateTenantUser(ctx, email, password)
}

func (s *stubStoreWithUpstream) UpdateTenantUserLastLogin(ctx context.Context, userID int64) error {
	return s.base.UpdateTenantUserLastLogin(ctx, userID)
}

func (s *stubStoreWithUpstream) GetTenantUserByID(ctx context.Context, userID int64) (entity.TenantUser, error) {
	return s.base.GetTenantUserByID(ctx, userID)
}

func (s *stubStoreWithUpstream) GetTenantByID(ctx context.Context, tenantID int64) (entity.Tenant, error) {
	return s.base.GetTenantByID(ctx, tenantID)
}

func (s *stubStoreWithUpstream) ListRequestLogs(ctx context.Context, input entity.ListRequestLogsInput) (entity.RequestLogPage, error) {
	return s.base.ListRequestLogs(ctx, input)
}

func (s *stubStoreWithUpstream) GetRequestLog(ctx context.Context, id int64) (entity.RequestLogDetail, error) {
	return s.base.GetRequestLog(ctx, id)
}

func (s *stubStoreWithUpstream) ExportRequestLogs(ctx context.Context, input entity.ListRequestLogsInput) ([]entity.RequestLog, error) {
	return s.base.ExportRequestLogs(ctx, input)
}

func (s *stubStoreWithUpstream) ExportTenantRequestLogs(ctx context.Context, tenantID int64, input entity.ListRequestLogsInput) ([]entity.RequestLog, error) {
	return s.base.ExportTenantRequestLogs(ctx, tenantID, input)
}

func (s *stubStoreWithUpstream) CreateRequestLog(ctx context.Context, input entity.CreateRequestLogInput) (int64, error) {
	return s.base.CreateRequestLog(ctx, input)
}

func (s *stubStoreWithUpstream) DeductTenantWalletUsage(ctx context.Context, input entity.TenantWalletUsageDebitInput) error {
	return s.base.DeductTenantWalletUsage(ctx, input)
}

func (s *stubStoreWithUpstream) GetDashboardStats(ctx context.Context) (entity.DashboardStats, error) {
	return s.base.GetDashboardStats(ctx)
}

func (s *stubStoreWithUpstream) GetTenantBillingReconciliation(ctx context.Context) ([]entity.TenantBillingReconciliation, error) {
	return s.base.GetTenantBillingReconciliation(ctx)
}

func (s *stubStoreWithUpstream) GetProviderRequestAnomalies(ctx context.Context, since time.Time, minRequests int, rateLimitedThreshold float64, serverErrorThreshold float64) ([]entity.ProviderRequestAnomaly, error) {
	return s.base.GetProviderRequestAnomalies(ctx, since, minRequests, rateLimitedThreshold, serverErrorThreshold)
}

func (s *stubStoreWithUpstream) GetTenantWalletBlockAnomalies(ctx context.Context, since time.Time, walletBlockThreshold int, reserveBlockThreshold int) ([]entity.TenantWalletBlockAnomaly, error) {
	return s.base.GetTenantWalletBlockAnomalies(ctx, since, walletBlockThreshold, reserveBlockThreshold)
}

func (s *stubStoreWithUpstream) GetTenantBillingDebitAnomalies(ctx context.Context, since time.Time, minCount int, minBillableAmount float64) ([]entity.TenantBillingDebitAnomaly, error) {
	return s.base.GetTenantBillingDebitAnomalies(ctx, since, minCount, minBillableAmount)
}

func (s *stubStoreWithUpstream) CreateAdminActionLog(ctx context.Context, input entity.CreateAdminActionLogInput) error {
	return s.base.CreateAdminActionLog(ctx, input)
}

func (s *stubStoreWithUpstream) ListTenantClientAPIKeys(ctx context.Context, tenantID int64) ([]entity.ClientAPIKey, error) {
	return s.base.ListTenantClientAPIKeys(ctx, tenantID)
}

func (s *stubStoreWithUpstream) CreateTenantClientAPIKey(ctx context.Context, input entity.CreateClientAPIKeyInput) (entity.ClientAPIKey, error) {
	return s.base.CreateTenantClientAPIKey(ctx, input)
}

func (s *stubStoreWithUpstream) DisableTenantClientAPIKey(ctx context.Context, tenantID int64, id int64) (entity.ClientAPIKey, error) {
	return s.base.DisableTenantClientAPIKey(ctx, tenantID, id)
}

func defaultTestProviderType(value string) string {
	if strings.TrimSpace(value) == "" {
		return "openai_compatible"
	}
	return strings.TrimSpace(value)
}

func TestAnthropicMessagesProxyRoute(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "sk-test-secret" {
			t.Fatalf("unexpected x-api-key header: %s", got)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Fatalf("unexpected anthropic-version header: %s", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream body: %v", err)
		}
		if got := payload["model"]; got != "claude-sonnet-upstream" {
			t.Fatalf("unexpected upstream model: %v", got)
		}
		if got := int(payload["max_tokens"].(float64)); got != 512 {
			t.Fatalf("unexpected max_tokens: %d", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "msg_123",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-upstream",
			"content": []map[string]any{
				{"type": "text", "text": "Hello from Claude"},
			},
			"usage": map[string]any{
				"input_tokens":  45,
				"output_tokens": 67,
			},
		})
	}))
	defer upstream.Close()

	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		models: []entity.Model{
			{
				ID:              1,
				PublicName:      "claude-sonnet-4",
				ProviderID:      1,
				ProviderName:    "anthropic",
				UpstreamModel:   "claude-sonnet-upstream",
				RouteStrategy:   "fixed",
				IsEnabled:       true,
				TimeoutSeconds:  30,
				MaxTokens:       512,
				CostInputPer1M:  3,
				CostOutputPer1M: 15,
				SaleInputPer1M:  3,
				SaleOutputPer1M: 15,
			},
		},
		clientKeys: []entity.ClientAPIKey{
			{ID: 7, Name: "integration-client", PlainAPIKey: "wcs_live_proxy_test", Status: "active"},
		},
	}
	routeStore := &stubStoreWithUpstream{
		base:         store,
		upstream:     upstream.URL,
		providerType: "anthropic",
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  routeStore,
		Auth:   routeStore,
		Log:    routeStore,
		Public: routeStore,
	})

	body, err := json.Marshal(map[string]any{
		"model": "claude-sonnet-4",
		"messages": []map[string]any{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer wcs_live_proxy_test")
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !upstreamCalled {
		t.Fatalf("expected upstream server to be called")
	}
	if len(store.createdLogs) != 1 {
		t.Fatalf("expected 1 created log, got %d", len(store.createdLogs))
	}
	if store.createdLogs[0].RequestType != "messages" {
		t.Fatalf("unexpected log request type: %+v", store.createdLogs[0])
	}
	if store.createdLogs[0].PromptTokens != 45 || store.createdLogs[0].CompletionTokens != 67 || store.createdLogs[0].TotalTokens != 112 {
		t.Fatalf("unexpected usage logged: %+v", store.createdLogs[0])
	}
}

func TestAnthropicMessagesStreamProxyRoute(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("event: message_start\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"usage\":{\"input_tokens\":21}}}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = w.Write([]byte("event: message_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":34}}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = w.Write([]byte("event: message_stop\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		models: []entity.Model{
			{
				ID:              1,
				PublicName:      "claude-sonnet-4",
				ProviderID:      1,
				ProviderName:    "anthropic",
				UpstreamModel:   "claude-sonnet-upstream",
				RouteStrategy:   "fixed",
				IsEnabled:       true,
				TimeoutSeconds:  30,
				MaxTokens:       512,
				CostInputPer1M:  3,
				CostOutputPer1M: 15,
				SaleInputPer1M:  3,
				SaleOutputPer1M: 15,
			},
		},
	}
	routeStore := &stubStoreWithUpstream{
		base:         store,
		upstream:     upstream.URL,
		providerType: "anthropic",
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  routeStore,
		Log:    routeStore,
		Public: routeStore,
	})

	body, err := json.Marshal(map[string]any{
		"model":      "claude-sonnet-4",
		"max_tokens": 128,
		"stream":     true,
		"messages": []map[string]any{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "\"type\":\"message_delta\"") {
		t.Fatalf("expected anthropic stream body, got %s", recorder.Body.String())
	}
	if len(store.createdLogs) != 1 {
		t.Fatalf("expected 1 created log, got %d", len(store.createdLogs))
	}
	if store.createdLogs[0].PromptTokens != 21 || store.createdLogs[0].CompletionTokens != 34 || store.createdLogs[0].TotalTokens != 55 {
		t.Fatalf("unexpected stream usage logged: %+v", store.createdLogs[0])
	}
}

func TestChatCompletionsFailoverToNextKey(t *testing.T) {
	requestCount := 0
	authHeaders := make([]string, 0, 2)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": "rate limited",
					"type":    "rate_limit_error",
				},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-failover",
		})
	}))
	defer upstream.Close()

	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		models: []entity.Model{
			{
				ID:             1,
				PublicName:     "qwen-plus",
				ProviderID:     1,
				ProviderName:   "stub-provider",
				UpstreamModel:  "qwen-plus-upstream",
				RouteStrategy:  "failover",
				IsEnabled:      true,
				TimeoutSeconds: 30,
			},
		},
	}

	body, err := json.Marshal(map[string]any{
		"model": "qwen-plus",
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	routeStore := &stubStoreWithUpstream{
		base:     store,
		upstream: upstream.URL + "/v1",
		keys: []entity.ProviderKey{
			{ID: 1, ProviderID: 1, Name: "primary", APIKey: "sk-primary", Status: "active", Priority: 10, Weight: 100},
			{ID: 2, ProviderID: 1, Name: "backup", APIKey: "sk-backup", Status: "active", Priority: 20, Weight: 100},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  routeStore,
		Log:    routeStore,
		Public: routeStore,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 upstream attempts, got %d", requestCount)
	}
	if len(authHeaders) != 2 || authHeaders[0] != "Bearer sk-primary" || authHeaders[1] != "Bearer sk-backup" {
		t.Fatalf("unexpected auth header sequence: %+v", authHeaders)
	}
	if len(store.createdLogs) != 1 {
		t.Fatalf("expected 1 created log, got %d", len(store.createdLogs))
	}
	if !store.createdLogs[0].Success || store.createdLogs[0].ProviderKeyID != 2 {
		t.Fatalf("expected successful failover log, got %+v", store.createdLogs[0])
	}
	if !strings.Contains(string(store.createdLogs[0].Metadata), "\"failover_count\":1") {
		t.Fatalf("expected failover metadata, got %s", string(store.createdLogs[0].Metadata))
	}
}

func TestChatCompletionsRetryLastKeyOnTransientFailure(t *testing.T) {
	requestCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": "temporary upstream issue",
					"type":    "server_error",
				},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-retry",
		})
	}))
	defer upstream.Close()

	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		models: []entity.Model{
			{
				ID:             1,
				PublicName:     "gpt-4o-mini",
				ProviderID:     1,
				ProviderName:   "stub-provider",
				UpstreamModel:  "gpt-4o-mini-upstream",
				RouteStrategy:  "fixed",
				IsEnabled:      true,
				TimeoutSeconds: 30,
			},
		},
	}

	body, err := json.Marshal(map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	routeStore := &stubStoreWithUpstream{
		base:     store,
		upstream: upstream.URL + "/v1",
		keys: []entity.ProviderKey{
			{ID: 1, ProviderID: 1, Name: "default", APIKey: "sk-test-secret", Status: "active", Priority: 10, Weight: 100},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  routeStore,
		Log:    routeStore,
		Public: routeStore,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 upstream attempts, got %d", requestCount)
	}
	if len(store.createdLogs) != 1 {
		t.Fatalf("expected 1 created log, got %d", len(store.createdLogs))
	}
	if !store.createdLogs[0].Success || store.createdLogs[0].ProviderKeyID != 1 {
		t.Fatalf("expected successful retry log, got %+v", store.createdLogs[0])
	}
	if !strings.Contains(string(store.createdLogs[0].Metadata), "\"retry_count\":1") {
		t.Fatalf("expected retry metadata, got %s", string(store.createdLogs[0].Metadata))
	}
}

func TestChatCompletionsSkipsCoolingDownKeyOnNextRequest(t *testing.T) {
	requestCount := 0
	authHeaders := make([]string, 0, 3)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		if requestCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": "primary rate limited",
					"type":    "rate_limit_error",
				},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-ok",
		})
	}))
	defer upstream.Close()

	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		models: []entity.Model{
			{
				ID:             1,
				PublicName:     "qwen-max",
				ProviderID:     1,
				ProviderName:   "stub-provider",
				UpstreamModel:  "qwen-max-upstream",
				RouteStrategy:  "failover",
				IsEnabled:      true,
				TimeoutSeconds: 30,
			},
		},
		providerKeys: []entity.ProviderKey{
			{ID: 1, ProviderID: 1, ProviderName: "stub-provider", Name: "primary", Status: "active", Weight: 100, Priority: 10, MaskedAPIKey: "sk-p***ary"},
			{ID: 2, ProviderID: 1, ProviderName: "stub-provider", Name: "backup", Status: "active", Weight: 100, Priority: 20, MaskedAPIKey: "sk-b***kup"},
		},
	}

	body, err := json.Marshal(map[string]any{
		"model": "qwen-max",
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	routeStore := &stubStoreWithUpstream{
		base:     store,
		upstream: upstream.URL + "/v1",
		keys: []entity.ProviderKey{
			{ID: 1, ProviderID: 1, Name: "primary", APIKey: "sk-primary", Status: "active", Priority: 10, Weight: 100},
			{ID: 2, ProviderID: 1, Name: "backup", APIKey: "sk-backup", Status: "active", Priority: 20, Weight: 100},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  routeStore,
		Log:    routeStore,
		Public: routeStore,
	})

	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d expected status %d, got %d", i+1, http.StatusOK, recorder.Code)
		}
	}

	if requestCount != 3 {
		t.Fatalf("expected 3 upstream requests, got %d", requestCount)
	}
	expectedHeaders := []string{"Bearer sk-primary", "Bearer sk-backup", "Bearer sk-backup"}
	if strings.Join(authHeaders, ",") != strings.Join(expectedHeaders, ",") {
		t.Fatalf("unexpected auth header sequence: %+v", authHeaders)
	}
	if len(store.createdLogs) != 2 {
		t.Fatalf("expected 2 created logs, got %d", len(store.createdLogs))
	}
	if !strings.Contains(string(store.createdLogs[1].Metadata), "\"temporarily_skipped_keys\"") {
		t.Fatalf("expected skipped key metadata, got %s", string(store.createdLogs[1].Metadata))
	}

	keysRecorder := httptest.NewRecorder()
	keysRequest := httptest.NewRequest(http.MethodGet, "/admin/keys", nil)
	engine.ServeHTTP(keysRecorder, keysRequest)

	if keysRecorder.Code != http.StatusOK {
		t.Fatalf("expected admin keys status %d, got %d", http.StatusOK, keysRecorder.Code)
	}
	if !strings.Contains(keysRecorder.Body.String(), "\"health_status\":\"cooldown\"") || !strings.Contains(keysRecorder.Body.String(), "\"cooldown_reason\":\"rate_limited\"") {
		t.Fatalf("expected cooldown key in admin response, got %s", keysRecorder.Body.String())
	}
}

func TestAdminDebugChatCompletionsUsesSelectedKey(t *testing.T) {
	authHeaders := make([]string, 0, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-debug",
		})
	}))
	defer upstream.Close()

	cfg := config.Config{AppName: "wcstransfer-gateway", Env: "test", GinMode: "test", HTTPPort: "8080"}
	store := &stubStore{
		models: []entity.Model{
			{
				ID:             1,
				PublicName:     "qwen-plus",
				ProviderID:     1,
				ProviderName:   "stub-provider",
				UpstreamModel:  "qwen-plus-upstream",
				RouteStrategy:  "round_robin",
				IsEnabled:      true,
				TimeoutSeconds: 30,
			},
		},
	}

	requestBody, err := json.Marshal(map[string]any{
		"provider_key_id": 2,
		"payload": map[string]any{
			"model": "qwen-plus",
			"messages": []map[string]string{
				{"role": "user", "content": "hello"},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal debug body: %v", err)
	}

	routeStore := &stubStoreWithUpstream{
		base:     store,
		upstream: upstream.URL + "/v1",
		keys: []entity.ProviderKey{
			{ID: 1, ProviderID: 1, Name: "primary", APIKey: "sk-primary", Status: "active", Priority: 10, Weight: 100},
			{ID: 2, ProviderID: 1, Name: "backup", APIKey: "sk-backup", Status: "active", Priority: 20, Weight: 100},
		},
	}
	engine := New(cfg, &platform.Dependencies{}, &Stores{
		Admin:  routeStore,
		Log:    routeStore,
		Public: routeStore,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/debug/chat/completions", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(authHeaders) != 1 || authHeaders[0] != "Bearer sk-backup" {
		t.Fatalf("expected backup key authorization, got %+v", authHeaders)
	}
	if got := recorder.Header().Get("X-Wcs-Debug-Provider-Key-Id"); got != "2" {
		t.Fatalf("expected selected provider key header, got %s", got)
	}
	if got := recorder.Header().Get("X-Wcs-Debug-Route-Strategy"); got != "fixed" {
		t.Fatalf("expected fixed strategy header after key override, got %s", got)
	}
}
