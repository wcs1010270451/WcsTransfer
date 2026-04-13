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
	ID               int64     `json:"id"`
	ProviderID       int64     `json:"provider_id"`
	ProviderName     string    `json:"provider_name"`
	Name             string    `json:"name"`
	APIKey           string    `json:"-"`
	Status           string    `json:"status"`
	Weight           int       `json:"weight"`
	Priority         int       `json:"priority"`
	RPMLimit         int       `json:"rpm_limit"`
	TPMLimit         int64     `json:"tpm_limit"`
	CurrentRPM       int       `json:"current_rpm"`
	CurrentTPM       int64     `json:"current_tpm"`
	MaskedAPIKey     string    `json:"masked_api_key"`
	LastErrorMessage string    `json:"last_error_message"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Model struct {
	ID             int64           `json:"id"`
	PublicName     string          `json:"public_name"`
	ProviderID     int64           `json:"provider_id"`
	ProviderName   string          `json:"provider_name"`
	UpstreamModel  string          `json:"upstream_model"`
	RouteStrategy  string          `json:"route_strategy"`
	IsEnabled      bool            `json:"is_enabled"`
	MaxTokens      int             `json:"max_tokens"`
	Temperature    float64         `json:"temperature"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type ModelRoute struct {
	Model    Model         `json:"model"`
	Provider Provider      `json:"provider"`
	Keys     []ProviderKey `json:"keys"`
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
	ClientIP         string    `json:"client_ip"`
	RequestMethod    string    `json:"request_method"`
	RequestPath      string    `json:"request_path"`
	HTTPStatus       int       `json:"http_status"`
	Success          bool      `json:"success"`
	LatencyMS        int       `json:"latency_ms"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	EstimatedCost    float64   `json:"estimated_cost"`
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

type DashboardStats struct {
	WindowHours       int                 `json:"window_hours"`
	ProviderCount     int                 `json:"provider_count"`
	KeyCount          int                 `json:"key_count"`
	ActiveKeyCount    int                 `json:"active_key_count"`
	ModelCount        int                 `json:"model_count"`
	EnabledModelCount int                 `json:"enabled_model_count"`
	RequestCount      int64               `json:"request_count"`
	SuccessCount      int64               `json:"success_count"`
	FailedCount       int64               `json:"failed_count"`
	SuccessRate       float64             `json:"success_rate"`
	AverageLatencyMS  float64             `json:"average_latency_ms"`
	PromptTokens      int64               `json:"prompt_tokens"`
	CompletionTokens  int64               `json:"completion_tokens"`
	TotalTokens       int64               `json:"total_tokens"`
	EstimatedCost     float64             `json:"estimated_cost"`
	TopModels         []ModelUsageStat    `json:"top_models"`
	TopProviders      []ProviderUsageStat `json:"top_providers"`
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

type CreateRequestLogInput struct {
	TraceID          string
	RequestType      string
	ModelPublicName  string
	UpstreamModel    string
	ProviderID       int64
	ProviderKeyID    int64
	ClientIP         string
	RequestMethod    string
	RequestPath      string
	HTTPStatus       int
	Success          bool
	LatencyMS        int
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	EstimatedCost    float64
	ErrorType        string
	ErrorMessage     string
	RequestPayload   json.RawMessage
	ResponsePayload  json.RawMessage
	Metadata         json.RawMessage
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
	PublicName     string
	ProviderID     int64
	UpstreamModel  string
	RouteStrategy  string
	IsEnabled      bool
	MaxTokens      int
	Temperature    float64
	TimeoutSeconds int
	Metadata       json.RawMessage
}

type UpdateModelInput struct {
	ID             int64
	PublicName     string
	ProviderID     int64
	UpstreamModel  string
	RouteStrategy  string
	IsEnabled      bool
	MaxTokens      int
	Temperature    float64
	TimeoutSeconds int
	Metadata       json.RawMessage
}
