package entity

import (
	"encoding/json"
	"time"
)

type Provider struct {
	ID           int64           `json:"id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	ProviderType string          `json:"provider_type"`
	BaseURL      string          `json:"base_url"`
	Status       string          `json:"status"`
	Description  string          `json:"description"`
	ExtraConfig  json.RawMessage `json:"extra_config"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type ProviderKey struct {
	ID               int64      `json:"id"`
	ProviderID       int64      `json:"provider_id"`
	ProviderName     string     `json:"provider_name"`
	Name             string     `json:"name"`
	APIKey           string     `json:"-"`
	Status           string     `json:"status"`
	HealthStatus     string     `json:"health_status"`
	CooldownReason   string     `json:"cooldown_reason,omitempty"`
	CooldownUntil    *time.Time `json:"cooldown_until,omitempty"`
	Weight           int        `json:"weight"`
	Priority         int        `json:"priority"`
	RPMLimit         int        `json:"rpm_limit"`
	TPMLimit         int64      `json:"tpm_limit"`
	CurrentRPM       int        `json:"current_rpm"`
	CurrentTPM       int64      `json:"current_tpm"`
	MaskedAPIKey     string     `json:"masked_api_key"`
	LastErrorMessage string     `json:"last_error_message"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type Model struct {
	ID               int64           `json:"id"`
	PublicName       string          `json:"public_name"`
	ProviderID       int64           `json:"provider_id"`
	ProviderName     string          `json:"provider_name"`
	UpstreamModel    string          `json:"upstream_model"`
	RouteStrategy    string          `json:"route_strategy"`
	IsEnabled        bool            `json:"is_enabled"`
	MaxTokens        int             `json:"max_tokens"`
	Temperature      float64         `json:"temperature"`
	TimeoutSeconds   int             `json:"timeout_seconds"`
	CostInputPer1M   float64         `json:"cost_input_per_1m"`
	CostOutputPer1M  float64         `json:"cost_output_per_1m"`
	SaleInputPer1M   float64         `json:"sale_input_per_1m"`
	SaleOutputPer1M  float64         `json:"sale_output_per_1m"`
	Metadata         json.RawMessage `json:"metadata"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type ModelRoute struct {
	Model    Model         `json:"model"`
	Provider Provider      `json:"provider"`
	Keys     []ProviderKey `json:"keys"`
}

type ClientAPIKey struct {
	ID                  int64             `json:"id"`
	TenantID            int64             `json:"tenant_id"`
	TenantName          string            `json:"tenant_name,omitempty"`
	TenantWalletBalance float64           `json:"tenant_wallet_balance,omitempty"`
	Name                string            `json:"name"`
	MaskedKey           string            `json:"masked_key"`
	PlainAPIKey         string            `json:"plain_api_key,omitempty"`
	Status              string            `json:"status"`
	Description         string            `json:"description"`
	RPMLimit            int               `json:"rpm_limit"`
	DailyRequestLimit   int               `json:"daily_request_limit"`
	DailyTokenLimit     int               `json:"daily_token_limit"`
	DailyCostLimit      float64           `json:"daily_cost_limit"`
	MonthlyCostLimit    float64           `json:"monthly_cost_limit"`
	WarningThreshold    float64           `json:"warning_threshold"`
	AllowedModelIDs     []int64           `json:"allowed_model_ids"`
	AllowedModels       []string          `json:"allowed_models"`
	Usage               *ClientQuotaUsage `json:"usage,omitempty"`
	CostUsage           *ClientCostUsage  `json:"cost_usage,omitempty"`
	ExpiresAt           *time.Time        `json:"expires_at,omitempty"`
	LastUsedAt          *time.Time        `json:"last_used_at,omitempty"`
	LastErrorAt         *time.Time        `json:"last_error_at,omitempty"`
	LastErrorMessage    string            `json:"last_error_message"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

type Tenant struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Status        string    `json:"status"`
	MaxClientKeys int       `json:"max_client_keys"`
	WalletBalance float64   `json:"wallet_balance"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type TenantWalletLedgerEntry struct {
	ID            int64     `json:"id"`
	TenantID      int64     `json:"tenant_id"`
	Direction     string    `json:"direction"`
	Amount        float64   `json:"amount"`
	BalanceBefore float64   `json:"balance_before"`
	BalanceAfter  float64   `json:"balance_after"`
	Note          string    `json:"note"`
	OperatorType  string    `json:"operator_type"`
	OperatorUserID int64    `json:"operator_user_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type TenantWalletLedgerPage struct {
	Items    []TenantWalletLedgerEntry `json:"items"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

type TenantWalletAdjustmentInput struct {
	TenantID   int64
	Amount     float64
	Note       string
	OperatorID int64
}

type TenantUser struct {
	ID          int64      `json:"id"`
	TenantID    int64      `json:"tenant_id"`
	TenantName  string     `json:"tenant_name,omitempty"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TenantLoginResult struct {
	User  TenantUser `json:"user"`
	Token string     `json:"token"`
}

type ClientQuotaUsage struct {
	CurrentRPM               int64      `json:"current_rpm"`
	DailyRequestsUsed        int64      `json:"daily_requests_used"`
	DailyTokensUsed          int64      `json:"daily_tokens_used"`
	RPMRemaining             int64      `json:"rpm_remaining"`
	DailyRequestsRemaining   int64      `json:"daily_requests_remaining"`
	DailyTokensRemaining     int64      `json:"daily_tokens_remaining"`
	RPMUsagePercent          float64    `json:"rpm_usage_percent"`
	DailyRequestUsagePercent float64    `json:"daily_request_usage_percent"`
	DailyTokenUsagePercent   float64    `json:"daily_token_usage_percent"`
	RPMResetAt               *time.Time `json:"rpm_reset_at,omitempty"`
	DailyResetAt             *time.Time `json:"daily_reset_at,omitempty"`
	IsRPMLimited             bool       `json:"is_rpm_limited"`
	IsDailyRequestLimited    bool       `json:"is_daily_request_limited"`
	IsDailyTokenLimited      bool       `json:"is_daily_token_limited"`
}

type ClientCostUsage struct {
	DailyCostUsed           float64    `json:"daily_cost_used"`
	MonthlyCostUsed         float64    `json:"monthly_cost_used"`
	DailyCostRemaining      float64    `json:"daily_cost_remaining"`
	MonthlyCostRemaining    float64    `json:"monthly_cost_remaining"`
	DailyCostUsagePercent   float64    `json:"daily_cost_usage_percent"`
	MonthlyCostUsagePercent float64    `json:"monthly_cost_usage_percent"`
	DailyResetAt            *time.Time `json:"daily_reset_at,omitempty"`
	MonthlyResetAt          *time.Time `json:"monthly_reset_at,omitempty"`
	IsDailyCostLimited      bool       `json:"is_daily_cost_limited"`
	IsMonthlyCostLimited    bool       `json:"is_monthly_cost_limited"`
	IsWarningTriggered      bool       `json:"is_warning_triggered"`
}

type RequestLog struct {
	ID               int64     `json:"id"`
	TraceID          string    `json:"trace_id"`
	RequestType      string    `json:"request_type"`
	ModelPublicName  string    `json:"model_public_name"`
	UpstreamModel    string    `json:"upstream_model"`
	ProviderID       int64     `json:"provider_id"`
	ProviderName     string    `json:"provider_name"`
	ProviderKeyID    int64     `json:"provider_key_id"`
	ProviderKeyName  string    `json:"provider_key_name"`
	ClientAPIKeyID   int64     `json:"client_api_key_id"`
	ClientAPIKeyName string    `json:"client_api_key_name"`
	ClientIP         string    `json:"client_ip"`
	RequestMethod    string    `json:"request_method"`
	RequestPath      string    `json:"request_path"`
	HTTPStatus       int       `json:"http_status"`
	Success          bool      `json:"success"`
	LatencyMS        int       `json:"latency_ms"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CostAmount       float64   `json:"cost_amount"`
	BillableAmount   float64   `json:"billable_amount"`
	ErrorType        string    `json:"error_type"`
	ErrorMessage     string    `json:"error_message"`
	CreatedAt        time.Time `json:"created_at"`
}

type RequestLogDetail struct {
	RequestLog
	RequestPayload  json.RawMessage `json:"request_payload"`
	ResponsePayload json.RawMessage `json:"response_payload"`
	Metadata        json.RawMessage `json:"metadata"`
}

type ListRequestLogsInput struct {
	Page            int
	PageSize        int
	TenantID        int64
	ProviderID      int64
	ModelPublicName string
	Success         *bool
	HTTPStatus      int
	TraceID         string
	CreatedFrom     *time.Time
	CreatedTo       *time.Time
}

type RequestLogPage struct {
	Items    []RequestLog `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

type TenantPortalStats struct {
	RequestCount     int64   `json:"request_count"`
	SuccessCount     int64   `json:"success_count"`
	FailedCount      int64   `json:"failed_count"`
	SuccessRate      float64 `json:"success_rate"`
	AverageLatencyMS float64 `json:"average_latency_ms"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	CostAmount       float64 `json:"cost_amount"`
	BillableAmount   float64 `json:"billable_amount"`
	ClientKeyCount   int64   `json:"client_key_count"`
	ActiveClientKeys int64   `json:"active_client_keys"`
}

type DashboardStats struct {
	WindowHours          int                    `json:"window_hours"`
	ProviderCount        int                    `json:"provider_count"`
	KeyCount             int                    `json:"key_count"`
	ActiveKeyCount       int                    `json:"active_key_count"`
	ClientKeyCount       int                    `json:"client_key_count"`
	ActiveClientKeyCount int                    `json:"active_client_key_count"`
	ModelCount           int                    `json:"model_count"`
	EnabledModelCount    int                    `json:"enabled_model_count"`
	RequestCount         int64                  `json:"request_count"`
	SuccessCount         int64                  `json:"success_count"`
	FailedCount          int64                  `json:"failed_count"`
	SuccessRate          float64                `json:"success_rate"`
	AverageLatencyMS     float64                `json:"average_latency_ms"`
	PromptTokens         int64                  `json:"prompt_tokens"`
	CompletionTokens     int64                  `json:"completion_tokens"`
	TotalTokens          int64                  `json:"total_tokens"`
	CostAmount           float64                `json:"cost_amount"`
	BillableAmount       float64                `json:"billable_amount"`
	TopModels            []ModelUsageStat       `json:"top_models"`
	TopProviders         []ProviderUsageStat    `json:"top_providers"`
	TopClients           []ClientUsageStat      `json:"top_clients"`
	QuotaPressure        []ClientQuotaPressure  `json:"quota_pressure"`
	BudgetPressure       []ClientBudgetPressure `json:"budget_pressure"`
}

type ModelUsageStat struct {
	ModelPublicName  string  `json:"model_public_name"`
	RequestCount     int64   `json:"request_count"`
	SuccessCount     int64   `json:"success_count"`
	FailedCount      int64   `json:"failed_count"`
	SuccessRate      float64 `json:"success_rate"`
	AverageLatencyMS float64 `json:"average_latency_ms"`
	TotalTokens      int64   `json:"total_tokens"`
}

type ProviderUsageStat struct {
	ProviderID       int64   `json:"provider_id"`
	ProviderName     string  `json:"provider_name"`
	RequestCount     int64   `json:"request_count"`
	SuccessCount     int64   `json:"success_count"`
	FailedCount      int64   `json:"failed_count"`
	SuccessRate      float64 `json:"success_rate"`
	AverageLatencyMS float64 `json:"average_latency_ms"`
	TotalTokens      int64   `json:"total_tokens"`
}

type ClientUsageStat struct {
	ClientAPIKeyID   int64   `json:"client_api_key_id"`
	ClientAPIKeyName string  `json:"client_api_key_name"`
	RequestCount     int64   `json:"request_count"`
	SuccessCount     int64   `json:"success_count"`
	FailedCount      int64   `json:"failed_count"`
	SuccessRate      float64 `json:"success_rate"`
	AverageLatencyMS float64 `json:"average_latency_ms"`
	TotalTokens      int64   `json:"total_tokens"`
	CostAmount       float64 `json:"cost_amount"`
	BillableAmount   float64 `json:"billable_amount"`
}

type ClientQuotaPressure struct {
	ClientAPIKeyID           int64    `json:"client_api_key_id"`
	ClientAPIKeyName         string   `json:"client_api_key_name"`
	HighestUsagePercent      float64  `json:"highest_usage_percent"`
	RPMUsagePercent          float64  `json:"rpm_usage_percent"`
	DailyRequestUsagePercent float64  `json:"daily_request_usage_percent"`
	DailyTokenUsagePercent   float64  `json:"daily_token_usage_percent"`
	LimitedDimensions        []string `json:"limited_dimensions"`
}

type ClientBudgetPressure struct {
	ClientAPIKeyID          int64    `json:"client_api_key_id"`
	ClientAPIKeyName        string   `json:"client_api_key_name"`
	HighestUsagePercent     float64  `json:"highest_usage_percent"`
	DailyCostUsagePercent   float64  `json:"daily_cost_usage_percent"`
	MonthlyCostUsagePercent float64  `json:"monthly_cost_usage_percent"`
	IsWarningTriggered      bool     `json:"is_warning_triggered"`
	LimitedDimensions       []string `json:"limited_dimensions"`
}

type CreateRequestLogInput struct {
	TraceID          string
	RequestType      string
	ModelPublicName  string
	UpstreamModel    string
	ProviderID       int64
	ProviderKeyID    int64
	ClientAPIKeyID   int64
	ClientIP         string
	RequestMethod    string
	RequestPath      string
	HTTPStatus       int
	Success          bool
	LatencyMS        int
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostAmount       float64
	BillableAmount   float64
	ErrorType        string
	ErrorMessage     string
	RequestPayload   json.RawMessage
	ResponsePayload  json.RawMessage
	Metadata         json.RawMessage
}

type CreateClientAPIKeyInput struct {
	TenantID          int64
	CreatedByUserID   int64
	Name              string
	Status            string
	Description       string
	RPMLimit          int
	DailyRequestLimit int
	DailyTokenLimit   int
	DailyCostLimit    float64
	MonthlyCostLimit  float64
	WarningThreshold  float64
	AllowedModelIDs   []int64
	ExpiresAt         *time.Time
}

type UpdateClientAPIKeyInput struct {
	ID                int64
	TenantID          int64
	Name              string
	Status            string
	Description       string
	RPMLimit          int
	DailyRequestLimit int
	DailyTokenLimit   int
	DailyCostLimit    float64
	MonthlyCostLimit  float64
	WarningThreshold  float64
	AllowedModelIDs   []int64
	ExpiresAt         *time.Time
}

type RegisterTenantUserInput struct {
	TenantName string
	TenantSlug string
	Email      string
	Password   string
	FullName   string
}

type UpdateTenantInput struct {
	ID            int64
	Name          string
	Slug          string
	Status        string
	MaxClientKeys int
	Notes         string
}

type TenantClientKeyInput struct {
	Name        string
	Description string
	ExpiresAt   *time.Time
}

type CreateProviderInput struct {
	Name         string
	Slug         string
	ProviderType string
	BaseURL      string
	Status       string
	Description  string
	ExtraConfig  json.RawMessage
}

type UpdateProviderInput struct {
	ID           int64
	Name         string
	Slug         string
	ProviderType string
	BaseURL      string
	Status       string
	Description  string
	ExtraConfig  json.RawMessage
}

type CreateProviderKeyInput struct {
	ProviderID int64
	Name       string
	APIKey     string
	Status     string
	Weight     int
	Priority   int
	RPMLimit   int
	TPMLimit   int64
}

type UpdateProviderKeyInput struct {
	ID         int64
	ProviderID int64
	Name       string
	APIKey     *string
	Status     string
	Weight     int
	Priority   int
	RPMLimit   int
	TPMLimit   int64
}

type CreateModelInput struct {
	PublicName      string
	ProviderID      int64
	UpstreamModel   string
	RouteStrategy   string
	IsEnabled       bool
	MaxTokens       int
	Temperature     float64
	TimeoutSeconds  int
	CostInputPer1M  float64
	CostOutputPer1M float64
	SaleInputPer1M  float64
	SaleOutputPer1M float64
	Metadata        json.RawMessage
}

type UpdateModelInput struct {
	ID              int64
	PublicName      string
	ProviderID      int64
	UpstreamModel   string
	RouteStrategy   string
	IsEnabled       bool
	MaxTokens       int
	Temperature     float64
	TimeoutSeconds  int
	CostInputPer1M  float64
	CostOutputPer1M float64
	SaleInputPer1M  float64
	SaleOutputPer1M float64
	Metadata        json.RawMessage
}
