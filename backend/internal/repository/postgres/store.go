package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wcstransfer/backend/internal/entity"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListProviders(ctx context.Context) ([]entity.Provider, error) {
	const query = `
SELECT id, name, slug, provider_type, base_url, status, description, extra_config, created_at, updated_at
FROM providers
ORDER BY id DESC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query providers: %w", err)
	}
	defer rows.Close()

	items := make([]entity.Provider, 0)
	for rows.Next() {
		var item entity.Provider
		var rawConfig []byte
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.ProviderType,
			&item.BaseURL,
			&item.Status,
			&item.Description,
			&rawConfig,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		item.ExtraConfig = normalizeJSON(rawConfig)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}

	return items, nil
}

func (s *Store) CreateProvider(ctx context.Context, input entity.CreateProviderInput) (entity.Provider, error) {
	const query = `
INSERT INTO providers (name, slug, provider_type, base_url, status, description, extra_config)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, slug, provider_type, base_url, status, description, extra_config, created_at, updated_at`

	var item entity.Provider
	var rawConfig []byte
	err := s.pool.QueryRow(
		ctx,
		query,
		input.Name,
		input.Slug,
		input.ProviderType,
		input.BaseURL,
		input.Status,
		input.Description,
		normalizeJSON(input.ExtraConfig),
	).Scan(
		&item.ID,
		&item.Name,
		&item.Slug,
		&item.ProviderType,
		&item.BaseURL,
		&item.Status,
		&item.Description,
		&rawConfig,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return entity.Provider{}, fmt.Errorf("insert provider: %w", err)
	}

	item.ExtraConfig = normalizeJSON(rawConfig)
	return item, nil
}

func (s *Store) UpdateProvider(ctx context.Context, input entity.UpdateProviderInput) (entity.Provider, error) {
	const query = `
UPDATE providers
SET name = $2,
    slug = $3,
    provider_type = $4,
    base_url = $5,
    status = $6,
    description = $7,
    extra_config = $8
WHERE id = $1
RETURNING id, name, slug, provider_type, base_url, status, description, extra_config, created_at, updated_at`

	var item entity.Provider
	var rawConfig []byte
	err := s.pool.QueryRow(
		ctx,
		query,
		input.ID,
		input.Name,
		input.Slug,
		input.ProviderType,
		input.BaseURL,
		input.Status,
		input.Description,
		normalizeJSON(input.ExtraConfig),
	).Scan(
		&item.ID,
		&item.Name,
		&item.Slug,
		&item.ProviderType,
		&item.BaseURL,
		&item.Status,
		&item.Description,
		&rawConfig,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return entity.Provider{}, fmt.Errorf("update provider: %w", err)
	}

	item.ExtraConfig = normalizeJSON(rawConfig)
	return item, nil
}

func (s *Store) ListProviderKeys(ctx context.Context) ([]entity.ProviderKey, error) {
	const query = `
SELECT pk.id, pk.provider_id, p.name, pk.name, pk.api_key, pk.status, pk.weight, pk.priority, pk.rpm_limit, pk.tpm_limit,
       pk.current_rpm, pk.current_tpm, pk.last_error_message, pk.created_at, pk.updated_at
FROM provider_keys pk
JOIN providers p ON p.id = pk.provider_id
ORDER BY pk.id DESC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query provider keys: %w", err)
	}
	defer rows.Close()

	items := make([]entity.ProviderKey, 0)
	for rows.Next() {
		var item entity.ProviderKey
		if err := rows.Scan(
			&item.ID,
			&item.ProviderID,
			&item.ProviderName,
			&item.Name,
			&item.APIKey,
			&item.Status,
			&item.Weight,
			&item.Priority,
			&item.RPMLimit,
			&item.TPMLimit,
			&item.CurrentRPM,
			&item.CurrentTPM,
			&item.LastErrorMessage,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan provider key: %w", err)
		}
		item.MaskedAPIKey = maskAPIKey(item.APIKey)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider keys: %w", err)
	}

	return items, nil
}

func (s *Store) UpdateProviderKey(ctx context.Context, input entity.UpdateProviderKeyInput) (entity.ProviderKey, error) {
	const query = `
UPDATE provider_keys
SET provider_id = $2,
    name = $3,
    api_key = COALESCE(NULLIF($4, ''), api_key),
    status = $5,
    weight = $6,
    priority = $7,
    rpm_limit = $8,
    tpm_limit = $9
WHERE id = $1
RETURNING id, provider_id, name, api_key, status, weight, priority, rpm_limit, tpm_limit, current_rpm, current_tpm,
          last_error_message, created_at, updated_at`

	apiKey := ""
	if input.APIKey != nil {
		apiKey = strings.TrimSpace(*input.APIKey)
	}

	var item entity.ProviderKey
	err := s.pool.QueryRow(
		ctx,
		query,
		input.ID,
		input.ProviderID,
		input.Name,
		apiKey,
		input.Status,
		input.Weight,
		input.Priority,
		input.RPMLimit,
		input.TPMLimit,
	).Scan(
		&item.ID,
		&item.ProviderID,
		&item.Name,
		&item.APIKey,
		&item.Status,
		&item.Weight,
		&item.Priority,
		&item.RPMLimit,
		&item.TPMLimit,
		&item.CurrentRPM,
		&item.CurrentTPM,
		&item.LastErrorMessage,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return entity.ProviderKey{}, fmt.Errorf("update provider key: %w", err)
	}

	item.MaskedAPIKey = maskAPIKey(item.APIKey)

	const providerNameQuery = `SELECT name FROM providers WHERE id = $1`
	if err := s.pool.QueryRow(ctx, providerNameQuery, item.ProviderID).Scan(&item.ProviderName); err != nil {
		return entity.ProviderKey{}, fmt.Errorf("load provider name: %w", err)
	}

	return item, nil
}

func (s *Store) CreateProviderKey(ctx context.Context, input entity.CreateProviderKeyInput) (entity.ProviderKey, error) {
	const query = `
INSERT INTO provider_keys (provider_id, name, api_key, status, weight, priority, rpm_limit, tpm_limit)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, provider_id, name, api_key, status, weight, priority, rpm_limit, tpm_limit, current_rpm, current_tpm,
          last_error_message, created_at, updated_at`

	var item entity.ProviderKey
	err := s.pool.QueryRow(
		ctx,
		query,
		input.ProviderID,
		input.Name,
		input.APIKey,
		input.Status,
		input.Weight,
		input.Priority,
		input.RPMLimit,
		input.TPMLimit,
	).Scan(
		&item.ID,
		&item.ProviderID,
		&item.Name,
		&item.APIKey,
		&item.Status,
		&item.Weight,
		&item.Priority,
		&item.RPMLimit,
		&item.TPMLimit,
		&item.CurrentRPM,
		&item.CurrentTPM,
		&item.LastErrorMessage,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return entity.ProviderKey{}, fmt.Errorf("insert provider key: %w", err)
	}

	item.MaskedAPIKey = maskAPIKey(item.APIKey)

	const providerNameQuery = `SELECT name FROM providers WHERE id = $1`
	if err := s.pool.QueryRow(ctx, providerNameQuery, item.ProviderID).Scan(&item.ProviderName); err != nil {
		return entity.ProviderKey{}, fmt.Errorf("load provider name: %w", err)
	}

	return item, nil
}

func (s *Store) ListModels(ctx context.Context) ([]entity.Model, error) {
	const query = `
SELECT m.id, m.public_name, m.provider_id, p.name, m.upstream_model, m.route_strategy, m.is_enabled,
       m.max_tokens, m.temperature::float8, m.timeout_seconds, m.metadata, m.created_at, m.updated_at
FROM models m
JOIN providers p ON p.id = m.provider_id
ORDER BY m.id DESC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query models: %w", err)
	}
	defer rows.Close()

	items := make([]entity.Model, 0)
	for rows.Next() {
		var item entity.Model
		var rawMetadata []byte
		if err := rows.Scan(
			&item.ID,
			&item.PublicName,
			&item.ProviderID,
			&item.ProviderName,
			&item.UpstreamModel,
			&item.RouteStrategy,
			&item.IsEnabled,
			&item.MaxTokens,
			&item.Temperature,
			&item.TimeoutSeconds,
			&rawMetadata,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		item.Metadata = normalizeJSON(rawMetadata)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate models: %w", err)
	}

	return items, nil
}

func (s *Store) CreateModel(ctx context.Context, input entity.CreateModelInput) (entity.Model, error) {
	const query = `
INSERT INTO models (public_name, provider_id, upstream_model, route_strategy, is_enabled, max_tokens, temperature, timeout_seconds, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, public_name, provider_id, upstream_model, route_strategy, is_enabled, max_tokens,
          temperature::float8, timeout_seconds, metadata, created_at, updated_at`

	var item entity.Model
	var rawMetadata []byte
	err := s.pool.QueryRow(
		ctx,
		query,
		input.PublicName,
		input.ProviderID,
		input.UpstreamModel,
		input.RouteStrategy,
		input.IsEnabled,
		input.MaxTokens,
		input.Temperature,
		input.TimeoutSeconds,
		normalizeJSON(input.Metadata),
	).Scan(
		&item.ID,
		&item.PublicName,
		&item.ProviderID,
		&item.UpstreamModel,
		&item.RouteStrategy,
		&item.IsEnabled,
		&item.MaxTokens,
		&item.Temperature,
		&item.TimeoutSeconds,
		&rawMetadata,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return entity.Model{}, fmt.Errorf("insert model: %w", err)
	}

	item.Metadata = normalizeJSON(rawMetadata)

	const providerNameQuery = `SELECT name FROM providers WHERE id = $1`
	if err := s.pool.QueryRow(ctx, providerNameQuery, item.ProviderID).Scan(&item.ProviderName); err != nil {
		return entity.Model{}, fmt.Errorf("load model provider name: %w", err)
	}

	return item, nil
}

func (s *Store) UpdateModel(ctx context.Context, input entity.UpdateModelInput) (entity.Model, error) {
	const query = `
UPDATE models
SET public_name = $2,
    provider_id = $3,
    upstream_model = $4,
    route_strategy = $5,
    is_enabled = $6,
    max_tokens = $7,
    temperature = $8,
    timeout_seconds = $9,
    metadata = $10
WHERE id = $1
RETURNING id, public_name, provider_id, upstream_model, route_strategy, is_enabled, max_tokens,
          temperature::float8, timeout_seconds, metadata, created_at, updated_at`

	var item entity.Model
	var rawMetadata []byte
	err := s.pool.QueryRow(
		ctx,
		query,
		input.ID,
		input.PublicName,
		input.ProviderID,
		input.UpstreamModel,
		input.RouteStrategy,
		input.IsEnabled,
		input.MaxTokens,
		input.Temperature,
		input.TimeoutSeconds,
		normalizeJSON(input.Metadata),
	).Scan(
		&item.ID,
		&item.PublicName,
		&item.ProviderID,
		&item.UpstreamModel,
		&item.RouteStrategy,
		&item.IsEnabled,
		&item.MaxTokens,
		&item.Temperature,
		&item.TimeoutSeconds,
		&rawMetadata,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return entity.Model{}, fmt.Errorf("update model: %w", err)
	}

	item.Metadata = normalizeJSON(rawMetadata)

	const providerNameQuery = `SELECT name FROM providers WHERE id = $1`
	if err := s.pool.QueryRow(ctx, providerNameQuery, item.ProviderID).Scan(&item.ProviderName); err != nil {
		return entity.Model{}, fmt.Errorf("load model provider name: %w", err)
	}

	return item, nil
}

func (s *Store) ListEnabledModels(ctx context.Context) ([]entity.Model, error) {
	const query = `
SELECT m.id, m.public_name, m.provider_id, p.name, m.upstream_model, m.route_strategy, m.is_enabled,
       m.max_tokens, m.temperature::float8, m.timeout_seconds, m.metadata, m.created_at, m.updated_at
FROM models m
JOIN providers p ON p.id = m.provider_id
WHERE m.is_enabled = TRUE
ORDER BY m.public_name ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query enabled models: %w", err)
	}
	defer rows.Close()

	items := make([]entity.Model, 0)
	for rows.Next() {
		var item entity.Model
		var rawMetadata []byte
		if err := rows.Scan(
			&item.ID,
			&item.PublicName,
			&item.ProviderID,
			&item.ProviderName,
			&item.UpstreamModel,
			&item.RouteStrategy,
			&item.IsEnabled,
			&item.MaxTokens,
			&item.Temperature,
			&item.TimeoutSeconds,
			&rawMetadata,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan enabled model: %w", err)
		}
		item.Metadata = normalizeJSON(rawMetadata)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enabled models: %w", err)
	}

	return items, nil
}

func (s *Store) ResolveModelRoute(ctx context.Context, publicName string) (entity.ModelRoute, error) {
	const modelQuery = `
SELECT
	m.id, m.public_name, m.provider_id, m.upstream_model, m.route_strategy, m.is_enabled,
	m.max_tokens, m.temperature::float8, m.timeout_seconds, m.metadata, m.created_at, m.updated_at,
	p.id, p.name, p.slug, p.provider_type, p.base_url, p.status, p.description, p.extra_config, p.created_at, p.updated_at
FROM models m
JOIN providers p ON p.id = m.provider_id
WHERE m.public_name = $1
  AND m.is_enabled = TRUE
  AND p.status = 'active'
LIMIT 1`

	var route entity.ModelRoute
	var modelMetadata []byte
	var providerExtraConfig []byte
	err := s.pool.QueryRow(ctx, modelQuery, publicName).Scan(
		&route.Model.ID,
		&route.Model.PublicName,
		&route.Model.ProviderID,
		&route.Model.UpstreamModel,
		&route.Model.RouteStrategy,
		&route.Model.IsEnabled,
		&route.Model.MaxTokens,
		&route.Model.Temperature,
		&route.Model.TimeoutSeconds,
		&modelMetadata,
		&route.Model.CreatedAt,
		&route.Model.UpdatedAt,
		&route.Provider.ID,
		&route.Provider.Name,
		&route.Provider.Slug,
		&route.Provider.ProviderType,
		&route.Provider.BaseURL,
		&route.Provider.Status,
		&route.Provider.Description,
		&providerExtraConfig,
		&route.Provider.CreatedAt,
		&route.Provider.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return entity.ModelRoute{}, err
		}

		return entity.ModelRoute{}, fmt.Errorf("resolve model route: %w", err)
	}

	route.Model.ProviderName = route.Provider.Name
	route.Model.Metadata = normalizeJSON(modelMetadata)
	route.Provider.ExtraConfig = normalizeJSON(providerExtraConfig)

	keys, err := s.listActiveProviderKeys(ctx, route.Provider.ID)
	if err != nil {
		return entity.ModelRoute{}, err
	}

	route.Keys = keys
	return route, nil
}

func (s *Store) ListRequestLogs(ctx context.Context, input entity.ListRequestLogsInput) (entity.RequestLogPage, error) {
	page := input.Page
	if page <= 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	baseWhere, args := buildRequestLogFilters(input)

	countQuery := "SELECT COUNT(*) FROM request_logs rl " + baseWhere
	var total int64
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return entity.RequestLogPage{}, fmt.Errorf("count request logs: %w", err)
	}

	args = append(args, pageSize, (page-1)*pageSize)
	query := `
SELECT rl.id, rl.trace_id, rl.request_type, rl.model_public_name, rl.upstream_model,
       COALESCE(rl.provider_id, 0), COALESCE(p.name, ''), COALESCE(rl.provider_key_id, 0), COALESCE(pk.name, ''),
       COALESCE(HOST(rl.client_ip), ''), rl.request_method, rl.request_path, rl.http_status, rl.success, rl.latency_ms,
       rl.prompt_tokens, rl.completion_tokens, rl.total_tokens, rl.estimated_cost::float8, rl.error_type, rl.error_message,
       rl.created_at
FROM request_logs rl
LEFT JOIN providers p ON p.id = rl.provider_id
LEFT JOIN provider_keys pk ON pk.id = rl.provider_key_id
` + baseWhere + `
ORDER BY rl.created_at DESC, rl.id DESC
LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return entity.RequestLogPage{}, fmt.Errorf("query request logs: %w", err)
	}
	defer rows.Close()

	items := make([]entity.RequestLog, 0)
	for rows.Next() {
		var item entity.RequestLog
		if err := rows.Scan(
			&item.ID,
			&item.TraceID,
			&item.RequestType,
			&item.ModelPublicName,
			&item.UpstreamModel,
			&item.ProviderID,
			&item.ProviderName,
			&item.ProviderKeyID,
			&item.ProviderKeyName,
			&item.ClientIP,
			&item.RequestMethod,
			&item.RequestPath,
			&item.HTTPStatus,
			&item.Success,
			&item.LatencyMS,
			&item.PromptTokens,
			&item.CompletionTokens,
			&item.TotalTokens,
			&item.EstimatedCost,
			&item.ErrorType,
			&item.ErrorMessage,
			&item.CreatedAt,
		); err != nil {
			return entity.RequestLogPage{}, fmt.Errorf("scan request log: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return entity.RequestLogPage{}, fmt.Errorf("iterate request logs: %w", err)
	}

	return entity.RequestLogPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Store) ExportRequestLogs(ctx context.Context, input entity.ListRequestLogsInput) ([]entity.RequestLog, error) {
	baseWhere, args := buildRequestLogFilters(input)
	query := `
SELECT rl.id, rl.trace_id, rl.request_type, rl.model_public_name, rl.upstream_model,
       COALESCE(rl.provider_id, 0), COALESCE(p.name, ''), COALESCE(rl.provider_key_id, 0), COALESCE(pk.name, ''),
       COALESCE(HOST(rl.client_ip), ''), rl.request_method, rl.request_path, rl.http_status, rl.success, rl.latency_ms,
       rl.prompt_tokens, rl.completion_tokens, rl.total_tokens, rl.estimated_cost::float8, rl.error_type, rl.error_message,
       rl.created_at
FROM request_logs rl
LEFT JOIN providers p ON p.id = rl.provider_id
LEFT JOIN provider_keys pk ON pk.id = rl.provider_key_id
` + baseWhere + `
ORDER BY rl.created_at DESC, rl.id DESC
LIMIT 5000`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("export request logs: %w", err)
	}
	defer rows.Close()

	return scanRequestLogs(rows)
}

func (s *Store) GetRequestLog(ctx context.Context, id int64) (entity.RequestLogDetail, error) {
	const query = `
SELECT rl.id, rl.trace_id, rl.request_type, rl.model_public_name, rl.upstream_model,
       COALESCE(rl.provider_id, 0), COALESCE(p.name, ''), COALESCE(rl.provider_key_id, 0), COALESCE(pk.name, ''),
       COALESCE(HOST(rl.client_ip), ''), rl.request_method, rl.request_path, rl.http_status, rl.success, rl.latency_ms,
       rl.prompt_tokens, rl.completion_tokens, rl.total_tokens, rl.estimated_cost::float8, rl.error_type, rl.error_message,
       rl.created_at, rl.request_payload, rl.response_payload, rl.metadata
FROM request_logs rl
LEFT JOIN providers p ON p.id = rl.provider_id
LEFT JOIN provider_keys pk ON pk.id = rl.provider_key_id
WHERE rl.id = $1`

	var item entity.RequestLogDetail
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.TraceID,
		&item.RequestType,
		&item.ModelPublicName,
		&item.UpstreamModel,
		&item.ProviderID,
		&item.ProviderName,
		&item.ProviderKeyID,
		&item.ProviderKeyName,
		&item.ClientIP,
		&item.RequestMethod,
		&item.RequestPath,
		&item.HTTPStatus,
		&item.Success,
		&item.LatencyMS,
		&item.PromptTokens,
		&item.CompletionTokens,
		&item.TotalTokens,
		&item.EstimatedCost,
		&item.ErrorType,
		&item.ErrorMessage,
		&item.CreatedAt,
		&item.RequestPayload,
		&item.ResponsePayload,
		&item.Metadata,
	)
	if err != nil {
		return entity.RequestLogDetail{}, fmt.Errorf("query request log detail: %w", err)
	}

	item.RequestPayload = normalizeJSON(item.RequestPayload)
	item.ResponsePayload = normalizeJSON(item.ResponsePayload)
	item.Metadata = normalizeJSON(item.Metadata)
	return item, nil
}

func (s *Store) GetDashboardStats(ctx context.Context) (entity.DashboardStats, error) {
	const windowHours = 24
	const summaryQuery = `
WITH resource_counts AS (
    SELECT
        (SELECT COUNT(*) FROM providers) AS provider_count,
        (SELECT COUNT(*) FROM provider_keys) AS key_count,
        (SELECT COUNT(*) FROM provider_keys WHERE status = 'active') AS active_key_count,
        (SELECT COUNT(*) FROM models) AS model_count,
        (SELECT COUNT(*) FROM models WHERE is_enabled = TRUE) AS enabled_model_count
),
request_stats AS (
    SELECT
        COUNT(*)::bigint AS request_count,
        COUNT(*) FILTER (WHERE success)::bigint AS success_count,
        COUNT(*) FILTER (WHERE NOT success)::bigint AS failed_count,
        COALESCE(AVG(latency_ms), 0)::float8 AS average_latency_ms,
        COALESCE(SUM(prompt_tokens), 0)::bigint AS prompt_tokens,
        COALESCE(SUM(completion_tokens), 0)::bigint AS completion_tokens,
        COALESCE(SUM(total_tokens), 0)::bigint AS total_tokens,
        COALESCE(SUM(estimated_cost), 0)::float8 AS estimated_cost
    FROM request_logs
    WHERE created_at >= NOW() - make_interval(hours => $1)
)
SELECT
    rc.provider_count,
    rc.key_count,
    rc.active_key_count,
    rc.model_count,
    rc.enabled_model_count,
    rs.request_count,
    rs.success_count,
    rs.failed_count,
    rs.average_latency_ms,
    rs.prompt_tokens,
    rs.completion_tokens,
    rs.total_tokens,
    rs.estimated_cost
FROM resource_counts rc
CROSS JOIN request_stats rs`

	stats := entity.DashboardStats{
		WindowHours:  windowHours,
		TopModels:    make([]entity.ModelUsageStat, 0),
		TopProviders: make([]entity.ProviderUsageStat, 0),
	}
	err := s.pool.QueryRow(ctx, summaryQuery, windowHours).Scan(
		&stats.ProviderCount,
		&stats.KeyCount,
		&stats.ActiveKeyCount,
		&stats.ModelCount,
		&stats.EnabledModelCount,
		&stats.RequestCount,
		&stats.SuccessCount,
		&stats.FailedCount,
		&stats.AverageLatencyMS,
		&stats.PromptTokens,
		&stats.CompletionTokens,
		&stats.TotalTokens,
		&stats.EstimatedCost,
	)
	if err != nil {
		return entity.DashboardStats{}, fmt.Errorf("query dashboard stats summary: %w", err)
	}
	stats.SuccessRate = calculateSuccessRate(stats.SuccessCount, stats.RequestCount)

	const topModelsQuery = `
SELECT
    model_public_name,
    COUNT(*)::bigint AS request_count,
    COUNT(*) FILTER (WHERE success)::bigint AS success_count,
    COUNT(*) FILTER (WHERE NOT success)::bigint AS failed_count,
    COALESCE(AVG(latency_ms), 0)::float8 AS average_latency_ms,
    COALESCE(SUM(total_tokens), 0)::bigint AS total_tokens
FROM request_logs
WHERE created_at >= NOW() - make_interval(hours => $1)
  AND model_public_name <> ''
GROUP BY model_public_name
ORDER BY request_count DESC, total_tokens DESC, model_public_name ASC
LIMIT 5`

	modelRows, err := s.pool.Query(ctx, topModelsQuery, windowHours)
	if err != nil {
		return entity.DashboardStats{}, fmt.Errorf("query dashboard top models: %w", err)
	}
	defer modelRows.Close()

	for modelRows.Next() {
		var item entity.ModelUsageStat
		if err := modelRows.Scan(
			&item.ModelPublicName,
			&item.RequestCount,
			&item.SuccessCount,
			&item.FailedCount,
			&item.AverageLatencyMS,
			&item.TotalTokens,
		); err != nil {
			return entity.DashboardStats{}, fmt.Errorf("scan dashboard top model: %w", err)
		}
		item.SuccessRate = calculateSuccessRate(item.SuccessCount, item.RequestCount)
		stats.TopModels = append(stats.TopModels, item)
	}
	if err := modelRows.Err(); err != nil {
		return entity.DashboardStats{}, fmt.Errorf("iterate dashboard top models: %w", err)
	}

	const topProvidersQuery = `
SELECT
    COALESCE(p.id, 0) AS provider_id,
    COALESCE(p.name, 'Unknown Provider') AS provider_name,
    COUNT(*)::bigint AS request_count,
    COUNT(*) FILTER (WHERE rl.success)::bigint AS success_count,
    COUNT(*) FILTER (WHERE NOT rl.success)::bigint AS failed_count,
    COALESCE(AVG(rl.latency_ms), 0)::float8 AS average_latency_ms,
    COALESCE(SUM(rl.total_tokens), 0)::bigint AS total_tokens
FROM request_logs rl
LEFT JOIN providers p ON p.id = rl.provider_id
WHERE rl.created_at >= NOW() - make_interval(hours => $1)
GROUP BY COALESCE(p.id, 0), COALESCE(p.name, 'Unknown Provider')
ORDER BY request_count DESC, total_tokens DESC, provider_name ASC
LIMIT 5`

	providerRows, err := s.pool.Query(ctx, topProvidersQuery, windowHours)
	if err != nil {
		return entity.DashboardStats{}, fmt.Errorf("query dashboard top providers: %w", err)
	}
	defer providerRows.Close()

	for providerRows.Next() {
		var item entity.ProviderUsageStat
		if err := providerRows.Scan(
			&item.ProviderID,
			&item.ProviderName,
			&item.RequestCount,
			&item.SuccessCount,
			&item.FailedCount,
			&item.AverageLatencyMS,
			&item.TotalTokens,
		); err != nil {
			return entity.DashboardStats{}, fmt.Errorf("scan dashboard top provider: %w", err)
		}
		item.SuccessRate = calculateSuccessRate(item.SuccessCount, item.RequestCount)
		stats.TopProviders = append(stats.TopProviders, item)
	}
	if err := providerRows.Err(); err != nil {
		return entity.DashboardStats{}, fmt.Errorf("iterate dashboard top providers: %w", err)
	}

	return stats, nil
}

func (s *Store) CreateRequestLog(ctx context.Context, input entity.CreateRequestLogInput) error {
	const query = `
INSERT INTO request_logs (
    trace_id, request_type, model_public_name, upstream_model, provider_id, provider_key_id,
    client_ip, request_method, request_path, http_status, success, latency_ms, prompt_tokens,
    completion_tokens, total_tokens, estimated_cost, error_type, error_message, request_payload,
    response_payload, metadata
) VALUES (
    $1, $2, $3, $4, NULLIF($5, 0), NULLIF($6, 0), NULLIF($7, '')::inet, $8, $9, $10, $11, $12,
    $13, $14, $15, $16, $17, $18, $19, $20, $21
)`

	_, err := s.pool.Exec(
		ctx,
		query,
		input.TraceID,
		input.RequestType,
		input.ModelPublicName,
		input.UpstreamModel,
		input.ProviderID,
		input.ProviderKeyID,
		input.ClientIP,
		input.RequestMethod,
		input.RequestPath,
		input.HTTPStatus,
		input.Success,
		input.LatencyMS,
		input.PromptTokens,
		input.CompletionTokens,
		input.TotalTokens,
		input.EstimatedCost,
		input.ErrorType,
		input.ErrorMessage,
		normalizeJSON(input.RequestPayload),
		normalizeJSON(input.ResponsePayload),
		normalizeJSON(input.Metadata),
	)
	if err != nil {
		return fmt.Errorf("insert request log: %w", err)
	}

	return nil
}

func normalizeJSON(input []byte) json.RawMessage {
	trimmed := strings.TrimSpace(string(input))
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}

	if !json.Valid([]byte(trimmed)) {
		return json.RawMessage(`{}`)
	}

	return json.RawMessage(trimmed)
}

func (s *Store) listActiveProviderKeys(ctx context.Context, providerID int64) ([]entity.ProviderKey, error) {
	const query = `
SELECT id, provider_id, name, api_key, status, weight, priority, rpm_limit, tpm_limit,
       current_rpm, current_tpm, last_error_message, created_at, updated_at
FROM provider_keys
WHERE provider_id = $1
  AND status = 'active'
ORDER BY priority ASC, weight DESC, id ASC`

	rows, err := s.pool.Query(ctx, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("query active provider keys: %w", err)
	}
	defer rows.Close()

	items := make([]entity.ProviderKey, 0)
	for rows.Next() {
		var item entity.ProviderKey
		if err := rows.Scan(
			&item.ID,
			&item.ProviderID,
			&item.Name,
			&item.APIKey,
			&item.Status,
			&item.Weight,
			&item.Priority,
			&item.RPMLimit,
			&item.TPMLimit,
			&item.CurrentRPM,
			&item.CurrentTPM,
			&item.LastErrorMessage,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan active provider key: %w", err)
		}

		item.MaskedAPIKey = maskAPIKey(item.APIKey)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active provider keys: %w", err)
	}

	return items, nil
}

func maskAPIKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	if len(trimmed) <= 8 {
		return "****"
	}

	return trimmed[:4] + strings.Repeat("*", len(trimmed)-8) + trimmed[len(trimmed)-4:]
}

func calculateSuccessRate(successCount int64, requestCount int64) float64 {
	if requestCount == 0 {
		return 0
	}

	return float64(successCount) * 100 / float64(requestCount)
}

func buildRequestLogFilters(input entity.ListRequestLogsInput) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	nextArg := func(value any) string {
		args = append(args, value)
		return "$" + strconv.Itoa(len(args))
	}

	if input.ProviderID > 0 {
		clauses = append(clauses, "rl.provider_id = "+nextArg(input.ProviderID))
	}
	if strings.TrimSpace(input.ModelPublicName) != "" {
		clauses = append(clauses, "rl.model_public_name = "+nextArg(strings.TrimSpace(input.ModelPublicName)))
	}
	if input.Success != nil {
		clauses = append(clauses, "rl.success = "+nextArg(*input.Success))
	}
	if input.HTTPStatus > 0 {
		clauses = append(clauses, "rl.http_status = "+nextArg(input.HTTPStatus))
	}
	if strings.TrimSpace(input.TraceID) != "" {
		clauses = append(clauses, "rl.trace_id ILIKE "+nextArg("%"+strings.TrimSpace(input.TraceID)+"%"))
	}
	if input.CreatedFrom != nil {
		clauses = append(clauses, "rl.created_at >= "+nextArg(*input.CreatedFrom))
	}
	if input.CreatedTo != nil {
		clauses = append(clauses, "rl.created_at <= "+nextArg(*input.CreatedTo))
	}

	if len(clauses) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

func scanRequestLogs(rows pgx.Rows) ([]entity.RequestLog, error) {
	items := make([]entity.RequestLog, 0)
	for rows.Next() {
		var item entity.RequestLog
		if err := rows.Scan(
			&item.ID,
			&item.TraceID,
			&item.RequestType,
			&item.ModelPublicName,
			&item.UpstreamModel,
			&item.ProviderID,
			&item.ProviderName,
			&item.ProviderKeyID,
			&item.ProviderKeyName,
			&item.ClientIP,
			&item.RequestMethod,
			&item.RequestPath,
			&item.HTTPStatus,
			&item.Success,
			&item.LatencyMS,
			&item.PromptTokens,
			&item.CompletionTokens,
			&item.TotalTokens,
			&item.EstimatedCost,
			&item.ErrorType,
			&item.ErrorMessage,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan request log: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate request logs: %w", err)
	}

	return items, nil
}
