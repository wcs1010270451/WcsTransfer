package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

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

func (s *Store) ListUsers(ctx context.Context) ([]entity.User, error) {
	const query = `
SELECT id, email, full_name, status, wallet_balance::float8, min_available_balance::float8, last_login_at, created_at, updated_at
FROM tenant_users
ORDER BY id DESC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	items := make([]entity.User, 0)
	for rows.Next() {
		var item entity.User
		if err := rows.Scan(&item.ID, &item.Email, &item.FullName, &item.Status, &item.WalletBalance, &item.MinAvailableBalance, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return items, nil
}

func (s *Store) CreateUser(ctx context.Context, input entity.CreateUserInput) (entity.User, error) {
	if strings.TrimSpace(input.Email) == "" {
		return entity.User{}, fmt.Errorf("email is required")
	}
	if strings.TrimSpace(input.FullName) == "" {
		return entity.User{}, fmt.Errorf("full name is required")
	}
	if len(strings.TrimSpace(input.Password)) < 8 {
		return entity.User{}, fmt.Errorf("password must be at least 8 characters")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return entity.User{}, fmt.Errorf("hash user password: %w", err)
	}

	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "active"
	}

	var item entity.User
	err = s.pool.QueryRow(ctx, `
INSERT INTO tenant_users (email, password_hash, full_name, status)
VALUES ($1, $2, $3, $4)
RETURNING id, email, full_name, status, wallet_balance::float8, min_available_balance::float8, last_login_at, created_at, updated_at`,
		strings.ToLower(strings.TrimSpace(input.Email)),
		string(passwordHash),
		strings.TrimSpace(input.FullName),
		status,
	).Scan(&item.ID, &item.Email, &item.FullName, &item.Status, &item.WalletBalance, &item.MinAvailableBalance, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return entity.User{}, fmt.Errorf("insert user: %w", err)
	}
	return item, nil
}

func (s *Store) UpdateUserStatus(ctx context.Context, input entity.UpdateUserStatusInput) (entity.User, error) {
	var item entity.User
	err := s.pool.QueryRow(ctx, `
UPDATE tenant_users SET status = $2 WHERE id = $1
RETURNING id, email, full_name, status, wallet_balance::float8, min_available_balance::float8, last_login_at, created_at, updated_at`,
		input.UserID, strings.TrimSpace(input.Status),
	).Scan(&item.ID, &item.Email, &item.FullName, &item.Status, &item.WalletBalance, &item.MinAvailableBalance, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return entity.User{}, fmt.Errorf("update user status: %w", err)
	}
	return item, nil
}

func (s *Store) ResetUserPassword(ctx context.Context, input entity.ResetUserPasswordInput) error {
	if len(strings.TrimSpace(input.Password)) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash user password: %w", err)
	}
	tag, err := s.pool.Exec(ctx, `UPDATE tenant_users SET password_hash = $2 WHERE id = $1`, input.UserID, string(passwordHash))
	if err != nil {
		return fmt.Errorf("reset user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) AdjustUserWallet(ctx context.Context, input entity.UserWalletAdjustmentInput) (entity.User, error) {
	if input.Amount <= 0 {
		return entity.User{}, fmt.Errorf("amount must be greater than 0")
	}
	return s.applyUserWalletAdminChange(ctx, input.UserID, "credit", input.Amount, input.Note, input.OperatorID)
}

func (s *Store) CorrectUserWallet(ctx context.Context, input entity.UserWalletCorrectionInput) (entity.User, error) {
	if input.Amount == 0 {
		return entity.User{}, fmt.Errorf("amount must not be zero")
	}
	if strings.TrimSpace(input.Note) == "" {
		return entity.User{}, fmt.Errorf("note is required")
	}
	direction := "credit"
	amount := input.Amount
	if amount < 0 {
		direction = "debit"
		amount = math.Abs(amount)
	}
	return s.applyUserWalletAdminChange(ctx, input.UserID, direction, amount, input.Note, input.OperatorID)
}

func (s *Store) applyUserWalletAdminChange(ctx context.Context, userID int64, direction string, amount float64, note string, operatorID int64) (entity.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return entity.User{}, fmt.Errorf("begin user wallet tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var item entity.User
	var before float64
	err = tx.QueryRow(ctx, `
SELECT id, email, full_name, status, wallet_balance::float8, min_available_balance::float8, last_login_at, created_at, updated_at
FROM tenant_users WHERE id = $1 FOR UPDATE`, userID).Scan(
		&item.ID, &item.Email, &item.FullName, &item.Status, &before,
		&item.MinAvailableBalance, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return entity.User{}, fmt.Errorf("load user wallet: %w", err)
	}

	after := before
	if direction == "debit" {
		after -= amount
		if after < 0 {
			return entity.User{}, fmt.Errorf("wallet balance cannot be negative after correction")
		}
	} else {
		after += amount
	}

	err = tx.QueryRow(ctx, `
UPDATE tenant_users SET wallet_balance = $2 WHERE id = $1
RETURNING id, email, full_name, status, wallet_balance::float8, min_available_balance::float8, last_login_at, created_at, updated_at`,
		userID, after,
	).Scan(&item.ID, &item.Email, &item.FullName, &item.Status, &item.WalletBalance, &item.MinAvailableBalance, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return entity.User{}, fmt.Errorf("update user wallet: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO tenant_wallet_ledger (user_id, direction, amount, balance_before, balance_after, note, operator_type, operator_user_id)
VALUES ($1, $2, $3, $4, $5, $6, 'admin', NULLIF($7, 0))`,
		userID, direction, amount, before, after, note, nullableInt64(operatorID),
	); err != nil {
		return entity.User{}, fmt.Errorf("insert user wallet ledger: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return entity.User{}, fmt.Errorf("commit user wallet tx: %w", err)
	}
	return item, nil
}

func (s *Store) ListUserWalletLedger(ctx context.Context, userID int64, page int, pageSize int) (entity.WalletLedgerPage, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tenant_wallet_ledger WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return entity.WalletLedgerPage{}, fmt.Errorf("count user wallet ledger: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
SELECT id, user_id, direction, amount::float8, balance_before::float8, balance_after::float8,
       note, operator_type, COALESCE(operator_user_id, 0), COALESCE(request_log_id, 0), trace_id,
       model_public_name, total_tokens, reserved_amount::float8, cost_amount::float8, billable_amount::float8, created_at
FROM tenant_wallet_ledger
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return entity.WalletLedgerPage{}, fmt.Errorf("query user wallet ledger: %w", err)
	}
	defer rows.Close()

	items := make([]entity.WalletLedgerEntry, 0)
	for rows.Next() {
		var item entity.WalletLedgerEntry
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Direction, &item.Amount,
			&item.BalanceBefore, &item.BalanceAfter, &item.Note, &item.OperatorType,
			&item.OperatorUserID, &item.RequestLogID, &item.TraceID, &item.ModelPublicName,
			&item.TotalTokens, &item.ReservedAmount, &item.CostAmount, &item.BillableAmount, &item.CreatedAt,
		); err != nil {
			return entity.WalletLedgerPage{}, fmt.Errorf("scan user wallet ledger: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return entity.WalletLedgerPage{}, fmt.Errorf("iterate user wallet ledger: %w", err)
	}
	return entity.WalletLedgerPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Store) ListClientAPIKeys(ctx context.Context) ([]entity.ClientAPIKey, error) {
	const query = `
SELECT cak.id, cak.name, cak.masked_key, cak.status, cak.description,
       COALESCE(cak.user_id, 0), COALESCE(tu.email, ''), COALESCE(tu.wallet_balance, 0)::float8, COALESCE(tu.min_available_balance, 0)::float8,
       cak.rpm_limit, cak.daily_request_limit, cak.daily_token_limit,
       cak.daily_cost_limit::float8, cak.monthly_cost_limit::float8, cak.warning_threshold::float8,
       COALESCE(model_access.allowed_model_ids, ARRAY[]::bigint[]),
       COALESCE(model_access.allowed_models, ARRAY[]::text[]),
       COALESCE(daily_usage.daily_cost_used, 0)::float8,
       COALESCE(monthly_usage.monthly_cost_used, 0)::float8,
       cak.expires_at, cak.last_used_at, cak.last_error_at, cak.last_error_message, cak.created_at, cak.updated_at
FROM client_api_keys cak
LEFT JOIN tenant_users tu ON tu.id = cak.user_id
LEFT JOIN LATERAL (
    SELECT
        array_agg(cam.model_id ORDER BY cam.model_id) AS allowed_model_ids,
        array_agg(m.public_name ORDER BY m.public_name) AS allowed_models
    FROM client_api_key_models cam
    JOIN models m ON m.id = cam.model_id
    WHERE cam.client_api_key_id = cak.id
) model_access ON TRUE
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(billable_amount), 0) AS daily_cost_used
    FROM request_logs
    WHERE client_api_key_id = cak.id
      AND created_at >= date_trunc('day', NOW())
) daily_usage ON TRUE
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(billable_amount), 0) AS monthly_cost_used
    FROM request_logs
    WHERE client_api_key_id = cak.id
      AND created_at >= date_trunc('month', NOW())
) monthly_usage ON TRUE
ORDER BY cak.id DESC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query client api keys: %w", err)
	}
	defer rows.Close()

	items := make([]entity.ClientAPIKey, 0)
	for rows.Next() {
		item, err := scanClientAPIKey(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate client api keys: %w", err)
	}

	return items, nil
}

func (s *Store) CreateClientAPIKey(ctx context.Context, input entity.CreateClientAPIKeyInput) (entity.ClientAPIKey, error) {
	plainKey, keyHash, maskedKey, err := generateClientAPIKeyMaterial()
	if err != nil {
		return entity.ClientAPIKey{}, fmt.Errorf("generate client api key: %w", err)
	}

	const query = `
INSERT INTO client_api_keys (
    user_id, name, key_hash, masked_key, status, description, rpm_limit, daily_request_limit, daily_token_limit,
    daily_cost_limit, monthly_cost_limit, warning_threshold, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING id`

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return entity.ClientAPIKey{}, fmt.Errorf("begin create client api key tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var clientKeyID int64
	err = tx.QueryRow(
		ctx,
		query,
		nullableInt64(input.UserID),
		input.Name,
		keyHash,
		maskedKey,
		input.Status,
		input.Description,
		input.RPMLimit,
		input.DailyRequestLimit,
		input.DailyTokenLimit,
		input.DailyCostLimit,
		input.MonthlyCostLimit,
		input.WarningThreshold,
		input.ExpiresAt,
	).Scan(&clientKeyID)
	if err != nil {
		return entity.ClientAPIKey{}, fmt.Errorf("insert client api key: %w", err)
	}

	if err := syncClientAPIKeyModels(ctx, tx, clientKeyID, input.AllowedModelIDs); err != nil {
		return entity.ClientAPIKey{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return entity.ClientAPIKey{}, fmt.Errorf("commit create client api key tx: %w", err)
	}

	item, err := s.getClientAPIKeyByID(ctx, clientKeyID)
	if err != nil {
		return entity.ClientAPIKey{}, err
	}
	item.PlainAPIKey = plainKey
	return item, nil
}

func (s *Store) UpdateClientAPIKey(ctx context.Context, input entity.UpdateClientAPIKeyInput) (entity.ClientAPIKey, error) {
	const query = `
UPDATE client_api_keys
SET name = $2,
    status = $3,
    description = $4,
    rpm_limit = $5,
    daily_request_limit = $6,
    daily_token_limit = $7,
    daily_cost_limit = $8,
    monthly_cost_limit = $9,
    warning_threshold = $10,
    expires_at = $11
WHERE id = $1
RETURNING id`

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return entity.ClientAPIKey{}, fmt.Errorf("begin update client api key tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var clientKeyID int64
	err = tx.QueryRow(
		ctx,
		query,
		input.ID,
		input.Name,
		input.Status,
		input.Description,
		input.RPMLimit,
		input.DailyRequestLimit,
		input.DailyTokenLimit,
		input.DailyCostLimit,
		input.MonthlyCostLimit,
		input.WarningThreshold,
		input.ExpiresAt,
	).Scan(&clientKeyID)
	if err != nil {
		return entity.ClientAPIKey{}, fmt.Errorf("update client api key: %w", err)
	}

	if err := syncClientAPIKeyModels(ctx, tx, clientKeyID, input.AllowedModelIDs); err != nil {
		return entity.ClientAPIKey{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return entity.ClientAPIKey{}, fmt.Errorf("commit update client api key tx: %w", err)
	}

	return s.getClientAPIKeyByID(ctx, clientKeyID)
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
       m.max_tokens, m.temperature::float8, m.timeout_seconds,
       m.cost_input_per_1m::float8, m.cost_output_per_1m::float8, m.sale_input_per_1m::float8, m.sale_output_per_1m::float8,
       m.reserve_multiplier::float8, m.reserve_min_amount::float8,
       m.metadata, m.created_at, m.updated_at
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
			&item.CostInputPer1M,
			&item.CostOutputPer1M,
			&item.SaleInputPer1M,
			&item.SaleOutputPer1M,
			&item.ReserveMultiplier,
			&item.ReserveMinAmount,
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
INSERT INTO models (
    public_name, provider_id, upstream_model, route_strategy, is_enabled, max_tokens,
    temperature, timeout_seconds, cost_input_per_1m, cost_output_per_1m, sale_input_per_1m, sale_output_per_1m,
    reserve_multiplier, reserve_min_amount, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING id, public_name, provider_id, upstream_model, route_strategy, is_enabled, max_tokens,
          temperature::float8, timeout_seconds, cost_input_per_1m::float8, cost_output_per_1m::float8, sale_input_per_1m::float8, sale_output_per_1m::float8,
          reserve_multiplier::float8, reserve_min_amount::float8,
          metadata, created_at, updated_at`

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
		input.CostInputPer1M,
		input.CostOutputPer1M,
		input.SaleInputPer1M,
		input.SaleOutputPer1M,
		input.ReserveMultiplier,
		input.ReserveMinAmount,
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
		&item.CostInputPer1M,
		&item.CostOutputPer1M,
		&item.SaleInputPer1M,
		&item.SaleOutputPer1M,
		&item.ReserveMultiplier,
		&item.ReserveMinAmount,
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
    cost_input_per_1m = $10,
    cost_output_per_1m = $11,
    sale_input_per_1m = $12,
    sale_output_per_1m = $13,
    reserve_multiplier = $14,
    reserve_min_amount = $15,
    metadata = $16
WHERE id = $1
RETURNING id, public_name, provider_id, upstream_model, route_strategy, is_enabled, max_tokens,
          temperature::float8, timeout_seconds, cost_input_per_1m::float8, cost_output_per_1m::float8, sale_input_per_1m::float8, sale_output_per_1m::float8,
          reserve_multiplier::float8, reserve_min_amount::float8,
          metadata, created_at, updated_at`

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
		input.CostInputPer1M,
		input.CostOutputPer1M,
		input.SaleInputPer1M,
		input.SaleOutputPer1M,
		input.ReserveMultiplier,
		input.ReserveMinAmount,
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
		&item.CostInputPer1M,
		&item.CostOutputPer1M,
		&item.SaleInputPer1M,
		&item.SaleOutputPer1M,
		&item.ReserveMultiplier,
		&item.ReserveMinAmount,
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
       m.max_tokens, m.temperature::float8, m.timeout_seconds,
       m.cost_input_per_1m::float8, m.cost_output_per_1m::float8, m.sale_input_per_1m::float8, m.sale_output_per_1m::float8,
       m.reserve_multiplier::float8, m.reserve_min_amount::float8,
       m.metadata, m.created_at, m.updated_at
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
			&item.CostInputPer1M,
			&item.CostOutputPer1M,
			&item.SaleInputPer1M,
			&item.SaleOutputPer1M,
			&item.ReserveMultiplier,
			&item.ReserveMinAmount,
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
	m.max_tokens, m.temperature::float8, m.timeout_seconds, m.cost_input_per_1m::float8, m.cost_output_per_1m::float8, m.sale_input_per_1m::float8, m.sale_output_per_1m::float8,
    m.reserve_multiplier::float8, m.reserve_min_amount::float8,
    m.metadata, m.created_at, m.updated_at,
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
		&route.Model.CostInputPer1M,
		&route.Model.CostOutputPer1M,
		&route.Model.SaleInputPer1M,
		&route.Model.SaleOutputPer1M,
		&route.Model.ReserveMultiplier,
		&route.Model.ReserveMinAmount,
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

func (s *Store) AuthenticateClientAPIKey(ctx context.Context, rawKey string) (entity.ClientAPIKey, error) {
	const query = `
WITH updated_key AS (
    UPDATE client_api_keys
    SET last_used_at = NOW()
    WHERE key_hash = $1
      AND status = 'active'
      AND (expires_at IS NULL OR expires_at > NOW())
    RETURNING id, user_id, name, masked_key, status, description, rpm_limit, daily_request_limit, daily_token_limit,
              daily_cost_limit, monthly_cost_limit, warning_threshold,
              expires_at, last_used_at, last_error_at, last_error_message, created_at, updated_at
)
SELECT uk.id, uk.name, uk.masked_key, uk.status, uk.description,
       COALESCE(uk.user_id, 0), COALESCE(tu.email, ''), COALESCE(tu.wallet_balance, 0)::float8, COALESCE(tu.min_available_balance, 0)::float8,
       uk.rpm_limit, uk.daily_request_limit, uk.daily_token_limit,
       uk.daily_cost_limit::float8, uk.monthly_cost_limit::float8, uk.warning_threshold::float8,
       COALESCE(model_access.allowed_model_ids, ARRAY[]::bigint[]),
       COALESCE(model_access.allowed_models, ARRAY[]::text[]),
       COALESCE(daily_usage.daily_cost_used, 0)::float8,
       COALESCE(monthly_usage.monthly_cost_used, 0)::float8,
       uk.expires_at, uk.last_used_at, uk.last_error_at, uk.last_error_message, uk.created_at, uk.updated_at
FROM updated_key uk
LEFT JOIN tenant_users tu ON tu.id = uk.user_id
LEFT JOIN LATERAL (
    SELECT
        array_agg(cam.model_id ORDER BY cam.model_id) AS allowed_model_ids,
        array_agg(m.public_name ORDER BY m.public_name) AS allowed_models
    FROM client_api_key_models cam
    JOIN models m ON m.id = cam.model_id
    WHERE cam.client_api_key_id = uk.id
) model_access ON TRUE
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(billable_amount), 0) AS daily_cost_used
    FROM request_logs
    WHERE client_api_key_id = uk.id
      AND created_at >= date_trunc('day', NOW())
) daily_usage ON TRUE
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(billable_amount), 0) AS monthly_cost_used
    FROM request_logs
    WHERE client_api_key_id = uk.id
      AND created_at >= date_trunc('month', NOW())
) monthly_usage ON TRUE
WHERE uk.user_id IS NULL OR tu.status = 'active'`

	item, err := scanClientAPIKey(s.pool.QueryRow(ctx, query, hashClientAPIKey(rawKey)))
	if err != nil {
		return entity.ClientAPIKey{}, fmt.Errorf("authenticate client api key: %w", err)
	}

	return item, nil
}

func (s *Store) AuthenticateAdminUser(ctx context.Context, username string, password string) (entity.AdminUser, error) {
	const query = `
SELECT id, username, display_name, status, password_hash, last_login_at, created_at, updated_at
FROM admin_users
WHERE username = $1
  AND status = 'active'`

	var user entity.AdminUser
	var passwordHash string
	err := s.pool.QueryRow(ctx, query, strings.TrimSpace(username)).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.Status,
		&passwordHash,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return entity.AdminUser{}, fmt.Errorf("authenticate admin user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return entity.AdminUser{}, pgx.ErrNoRows
	}

	return user, nil
}

func (s *Store) UpdateAdminUserLastLogin(ctx context.Context, userID int64) error {
	if _, err := s.pool.Exec(ctx, `UPDATE admin_users SET last_login_at = NOW() WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("update admin user last login: %w", err)
	}
	return nil
}

func (s *Store) GetAdminUserByID(ctx context.Context, userID int64) (entity.AdminUser, error) {
	const query = `
SELECT id, username, display_name, status, last_login_at, created_at, updated_at
FROM admin_users
WHERE id = $1
  AND status = 'active'`

	var user entity.AdminUser
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.Status,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return entity.AdminUser{}, fmt.Errorf("get admin user by id: %w", err)
	}

	return user, nil
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
       COALESCE(rl.client_api_key_id, 0), COALESCE(cak.name, ''),
       COALESCE(HOST(rl.client_ip), ''), rl.request_method, rl.request_path, rl.http_status, rl.success, rl.latency_ms,
       rl.prompt_tokens, rl.completion_tokens, rl.total_tokens, rl.reserved_amount::float8, rl.cost_amount::float8, rl.billable_amount::float8, rl.error_type, rl.error_message,
       rl.created_at
FROM request_logs rl
LEFT JOIN providers p ON p.id = rl.provider_id
LEFT JOIN provider_keys pk ON pk.id = rl.provider_key_id
LEFT JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
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
			&item.ClientAPIKeyID,
			&item.ClientAPIKeyName,
			&item.ClientIP,
			&item.RequestMethod,
			&item.RequestPath,
			&item.HTTPStatus,
			&item.Success,
			&item.LatencyMS,
			&item.PromptTokens,
			&item.CompletionTokens,
			&item.TotalTokens,
			&item.ReservedAmount,
			&item.CostAmount,
			&item.BillableAmount,
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
       COALESCE(rl.client_api_key_id, 0), COALESCE(cak.name, ''),
       COALESCE(HOST(rl.client_ip), ''), rl.request_method, rl.request_path, rl.http_status, rl.success, rl.latency_ms,
       rl.prompt_tokens, rl.completion_tokens, rl.total_tokens, rl.reserved_amount::float8, rl.cost_amount::float8, rl.billable_amount::float8, rl.error_type, rl.error_message,
       rl.created_at
FROM request_logs rl
LEFT JOIN providers p ON p.id = rl.provider_id
LEFT JOIN provider_keys pk ON pk.id = rl.provider_key_id
LEFT JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
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
       COALESCE(rl.client_api_key_id, 0), COALESCE(cak.name, ''),
       COALESCE(HOST(rl.client_ip), ''), rl.request_method, rl.request_path, rl.http_status, rl.success, rl.latency_ms,
       rl.prompt_tokens, rl.completion_tokens, rl.total_tokens, rl.reserved_amount::float8, rl.cost_amount::float8, rl.billable_amount::float8, rl.error_type, rl.error_message,
       rl.created_at, rl.request_payload, rl.response_payload, rl.metadata
FROM request_logs rl
LEFT JOIN providers p ON p.id = rl.provider_id
LEFT JOIN provider_keys pk ON pk.id = rl.provider_key_id
LEFT JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
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
		&item.ClientAPIKeyID,
		&item.ClientAPIKeyName,
		&item.ClientIP,
		&item.RequestMethod,
		&item.RequestPath,
		&item.HTTPStatus,
		&item.Success,
		&item.LatencyMS,
		&item.PromptTokens,
		&item.CompletionTokens,
		&item.TotalTokens,
		&item.ReservedAmount,
		&item.CostAmount,
		&item.BillableAmount,
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
        (SELECT COUNT(*) FROM client_api_keys) AS client_key_count,
        (SELECT COUNT(*) FROM client_api_keys WHERE status = 'active') AS active_client_key_count,
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
        COALESCE(SUM(cost_amount), 0)::float8 AS cost_amount,
        COALESCE(SUM(billable_amount), 0)::float8 AS billable_amount
    FROM request_logs
    WHERE created_at >= NOW() - make_interval(hours => $1)
),
today_billing AS (
    SELECT
        COALESCE(SUM(cost_amount), 0)::float8 AS cost_amount,
        COALESCE(SUM(billable_amount), 0)::float8 AS billable_amount
    FROM request_logs
    WHERE created_at >= date_trunc('day', NOW())
),
month_billing AS (
    SELECT
        COALESCE(SUM(cost_amount), 0)::float8 AS cost_amount,
        COALESCE(SUM(billable_amount), 0)::float8 AS billable_amount
    FROM request_logs
    WHERE created_at >= date_trunc('month', NOW())
)
SELECT
    rc.provider_count,
    rc.key_count,
    rc.active_key_count,
    rc.client_key_count,
    rc.active_client_key_count,
    rc.model_count,
    rc.enabled_model_count,
    rs.request_count,
    rs.success_count,
    rs.failed_count,
    rs.average_latency_ms,
    rs.prompt_tokens,
    rs.completion_tokens,
    rs.total_tokens,
    rs.cost_amount,
    rs.billable_amount,
    tb.cost_amount,
    tb.billable_amount,
    mb.cost_amount,
    mb.billable_amount
FROM resource_counts rc
CROSS JOIN request_stats rs
CROSS JOIN today_billing tb
CROSS JOIN month_billing mb`

	stats := entity.DashboardStats{
		WindowHours:    windowHours,
		TopModels:      make([]entity.ModelUsageStat, 0),
		TopProviders:   make([]entity.ProviderUsageStat, 0),
		TopClients:     make([]entity.ClientUsageStat, 0),
		QuotaPressure:  make([]entity.ClientQuotaPressure, 0),
		BudgetPressure: make([]entity.ClientBudgetPressure, 0),
	}
	err := s.pool.QueryRow(ctx, summaryQuery, windowHours).Scan(
		&stats.ProviderCount,
		&stats.KeyCount,
		&stats.ActiveKeyCount,
		&stats.ClientKeyCount,
		&stats.ActiveClientKeyCount,
		&stats.ModelCount,
		&stats.EnabledModelCount,
		&stats.RequestCount,
		&stats.SuccessCount,
		&stats.FailedCount,
		&stats.AverageLatencyMS,
		&stats.PromptTokens,
		&stats.CompletionTokens,
		&stats.TotalTokens,
		&stats.CostAmount,
		&stats.BillableAmount,
		&stats.TodayCostAmount,
		&stats.TodayBillableAmount,
		&stats.MonthCostAmount,
		&stats.MonthBillableAmount,
	)
	if err != nil {
		return entity.DashboardStats{}, fmt.Errorf("query dashboard stats summary: %w", err)
	}
	stats.SuccessRate = calculateSuccessRate(stats.SuccessCount, stats.RequestCount)
	stats.GrossProfit = stats.BillableAmount - stats.CostAmount
	stats.TodayGrossProfit = stats.TodayBillableAmount - stats.TodayCostAmount
	stats.MonthGrossProfit = stats.MonthBillableAmount - stats.MonthCostAmount

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

	const topClientsQuery = `
SELECT
    COALESCE(cak.id, 0) AS client_api_key_id,
    COALESCE(cak.name, 'Anonymous Client') AS client_api_key_name,
    COUNT(*)::bigint AS request_count,
    COUNT(*) FILTER (WHERE rl.success)::bigint AS success_count,
    COUNT(*) FILTER (WHERE NOT rl.success)::bigint AS failed_count,
    COALESCE(AVG(rl.latency_ms), 0)::float8 AS average_latency_ms,
    COALESCE(SUM(rl.total_tokens), 0)::bigint AS total_tokens,
    COALESCE(SUM(rl.cost_amount), 0)::float8 AS cost_amount,
    COALESCE(SUM(rl.billable_amount), 0)::float8 AS billable_amount
FROM request_logs rl
LEFT JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
WHERE rl.created_at >= NOW() - make_interval(hours => $1)
GROUP BY COALESCE(cak.id, 0), COALESCE(cak.name, 'Anonymous Client')
ORDER BY billable_amount DESC, request_count DESC, client_api_key_name ASC
LIMIT 5`

	clientRows, err := s.pool.Query(ctx, topClientsQuery, windowHours)
	if err != nil {
		return entity.DashboardStats{}, fmt.Errorf("query dashboard top clients: %w", err)
	}
	defer clientRows.Close()

	for clientRows.Next() {
		var item entity.ClientUsageStat
		if err := clientRows.Scan(
			&item.ClientAPIKeyID,
			&item.ClientAPIKeyName,
			&item.RequestCount,
			&item.SuccessCount,
			&item.FailedCount,
			&item.AverageLatencyMS,
			&item.TotalTokens,
			&item.CostAmount,
			&item.BillableAmount,
		); err != nil {
			return entity.DashboardStats{}, fmt.Errorf("scan dashboard top client: %w", err)
		}
		item.SuccessRate = calculateSuccessRate(item.SuccessCount, item.RequestCount)
		stats.TopClients = append(stats.TopClients, item)
	}
	if err := clientRows.Err(); err != nil {
		return entity.DashboardStats{}, fmt.Errorf("iterate dashboard top clients: %w", err)
	}

	return stats, nil
}

func (s *Store) GetProviderRequestAnomalies(ctx context.Context, since time.Time, minRequests int, rateLimitedThreshold float64, serverErrorThreshold float64) ([]entity.ProviderRequestAnomaly, error) {
	const query = `
SELECT
    rl.provider_id,
    COALESCE(p.name, 'Unknown Provider') AS provider_name,
    COUNT(*)::bigint AS total_requests,
    COUNT(*) FILTER (WHERE rl.http_status = 429)::bigint AS rate_limited_count,
    COUNT(*) FILTER (WHERE rl.http_status >= 500)::bigint AS server_error_count
FROM request_logs rl
LEFT JOIN providers p ON p.id = rl.provider_id
WHERE rl.created_at >= $1
  AND rl.provider_id IS NOT NULL
GROUP BY rl.provider_id, COALESCE(p.name, 'Unknown Provider')
HAVING COUNT(*) >= $2
ORDER BY total_requests DESC, provider_name ASC`

	rows, err := s.pool.Query(ctx, query, since, minRequests)
	if err != nil {
		return nil, fmt.Errorf("query provider request anomalies: %w", err)
	}
	defer rows.Close()

	items := make([]entity.ProviderRequestAnomaly, 0)
	for rows.Next() {
		var item entity.ProviderRequestAnomaly
		if err := rows.Scan(
			&item.ProviderID,
			&item.ProviderName,
			&item.TotalRequests,
			&item.RateLimitedCount,
			&item.ServerErrorCount,
		); err != nil {
			return nil, fmt.Errorf("scan provider request anomaly: %w", err)
		}

		if item.TotalRequests > 0 {
			item.RateLimitedRatio = float64(item.RateLimitedCount) / float64(item.TotalRequests)
			item.ServerErrorRatio = float64(item.ServerErrorCount) / float64(item.TotalRequests)
		}
		item.IsRateLimitedAnomalous = item.RateLimitedRatio >= rateLimitedThreshold
		item.IsServerErrorAnomalous = item.ServerErrorRatio >= serverErrorThreshold
		if item.IsRateLimitedAnomalous || item.IsServerErrorAnomalous {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider request anomalies: %w", err)
	}

	return items, nil
}

func (s *Store) CreateAdminActionLog(ctx context.Context, input entity.CreateAdminActionLogInput) error {
	const query = `
INSERT INTO admin_action_logs (
    admin_user_id, admin_username, admin_display_name, auth_mode, action, resource_type, resource_id,
    resource_name, request_method, request_path, client_ip, metadata
) VALUES (
    NULLIF($1, 0), $2, $3, $4, $5, $6, NULLIF($7, 0), $8, $9, $10, NULLIF($11, '')::inet, $12
)`

	if _, err := s.pool.Exec(
		ctx,
		query,
		input.AdminUserID,
		input.AdminUsername,
		input.AdminDisplayName,
		input.AuthMode,
		input.Action,
		input.ResourceType,
		input.ResourceID,
		input.ResourceName,
		input.RequestMethod,
		input.RequestPath,
		input.ClientIP,
		normalizeJSON(input.Metadata),
	); err != nil {
		return fmt.Errorf("insert admin action log: %w", err)
	}

	return nil
}

func (s *Store) CreateRequestLog(ctx context.Context, input entity.CreateRequestLogInput) (int64, error) {
	const query = `
INSERT INTO request_logs (
    trace_id, request_type, model_public_name, upstream_model, provider_id, provider_key_id, client_api_key_id,
    client_ip, request_method, request_path, http_status, success, latency_ms, prompt_tokens,
    completion_tokens, total_tokens, reserved_amount, estimated_cost, cost_amount, billable_amount, error_type, error_message,
    request_payload, response_payload, metadata
) VALUES (
    $1, $2, $3, $4, NULLIF($5, 0), NULLIF($6, 0), NULLIF($7, 0), NULLIF($8, '')::inet, $9, $10, $11, $12, $13,
    $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
) RETURNING id`

	var id int64
	err := s.pool.QueryRow(
		ctx,
		query,
		input.TraceID,
		input.RequestType,
		input.ModelPublicName,
		input.UpstreamModel,
		input.ProviderID,
		input.ProviderKeyID,
		input.ClientAPIKeyID,
		input.ClientIP,
		input.RequestMethod,
		input.RequestPath,
		input.HTTPStatus,
		input.Success,
		input.LatencyMS,
		input.PromptTokens,
		input.CompletionTokens,
		input.TotalTokens,
		input.ReservedAmount,
		input.BillableAmount,
		input.CostAmount,
		input.BillableAmount,
		input.ErrorType,
		input.ErrorMessage,
		normalizeJSON(input.RequestPayload),
		normalizeJSON(input.ResponsePayload),
		normalizeJSON(input.Metadata),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert request log: %w", err)
	}

	return id, nil
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

func generateClientAPIKeyMaterial() (string, string, string, error) {
	buffer := make([]byte, 18)
	if _, err := rand.Read(buffer); err != nil {
		return "", "", "", err
	}

	suffix := hex.EncodeToString(buffer)
	plainKey := "wcs_live_" + suffix
	return plainKey, hashClientAPIKey(plainKey), maskClientKey(plainKey), nil
}

func hashClientAPIKey(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func maskClientKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 12 {
		return trimmed
	}

	return trimmed[:12] + "..." + trimmed[len(trimmed)-4:]
}

func nullableInt64(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func scanClientAPIKey(scanner interface {
	Scan(dest ...any) error
}) (entity.ClientAPIKey, error) {
	var item entity.ClientAPIKey
	var dailyCostUsed float64
	var monthlyCostUsed float64
	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.MaskedKey,
		&item.Status,
		&item.Description,
		&item.UserID,
		&item.UserEmail,
		&item.UserWalletBalance,
		&item.UserMinAvailBalance,
		&item.RPMLimit,
		&item.DailyRequestLimit,
		&item.DailyTokenLimit,
		&item.DailyCostLimit,
		&item.MonthlyCostLimit,
		&item.WarningThreshold,
		&item.AllowedModelIDs,
		&item.AllowedModels,
		&dailyCostUsed,
		&monthlyCostUsed,
		&item.ExpiresAt,
		&item.LastUsedAt,
		&item.LastErrorAt,
		&item.LastErrorMessage,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return entity.ClientAPIKey{}, fmt.Errorf("scan client api key: %w", err)
	}

	item.CostUsage = buildClientCostUsage(item, dailyCostUsed, monthlyCostUsed)
	return item, nil
}

func (s *Store) getClientAPIKeyByID(ctx context.Context, id int64) (entity.ClientAPIKey, error) {
	const query = `
SELECT cak.id, cak.name, cak.masked_key, cak.status, cak.description,
       COALESCE(cak.user_id, 0), COALESCE(tu.email, ''), COALESCE(tu.wallet_balance, 0)::float8, COALESCE(tu.min_available_balance, 0)::float8,
       cak.rpm_limit, cak.daily_request_limit, cak.daily_token_limit,
       cak.daily_cost_limit::float8, cak.monthly_cost_limit::float8, cak.warning_threshold::float8,
       COALESCE(model_access.allowed_model_ids, ARRAY[]::bigint[]),
       COALESCE(model_access.allowed_models, ARRAY[]::text[]),
       COALESCE(daily_usage.daily_cost_used, 0)::float8,
       COALESCE(monthly_usage.monthly_cost_used, 0)::float8,
       cak.expires_at, cak.last_used_at, cak.last_error_at, cak.last_error_message, cak.created_at, cak.updated_at
FROM client_api_keys cak
LEFT JOIN tenant_users tu ON tu.id = cak.user_id
LEFT JOIN LATERAL (
    SELECT
        array_agg(cam.model_id ORDER BY cam.model_id) AS allowed_model_ids,
        array_agg(m.public_name ORDER BY m.public_name) AS allowed_models
    FROM client_api_key_models cam
    JOIN models m ON m.id = cam.model_id
    WHERE cam.client_api_key_id = cak.id
) model_access ON TRUE
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(billable_amount), 0) AS daily_cost_used
    FROM request_logs
    WHERE client_api_key_id = cak.id
      AND created_at >= date_trunc('day', NOW())
) daily_usage ON TRUE
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(billable_amount), 0) AS monthly_cost_used
    FROM request_logs
WHERE client_api_key_id = cak.id
      AND created_at >= date_trunc('month', NOW())
) monthly_usage ON TRUE
WHERE cak.id = $1`

	return scanClientAPIKey(s.pool.QueryRow(ctx, query, id))
}

func syncClientAPIKeyModels(ctx context.Context, tx pgx.Tx, clientKeyID int64, modelIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM client_api_key_models WHERE client_api_key_id = $1`, clientKeyID); err != nil {
		return fmt.Errorf("clear client api key models: %w", err)
	}

	normalized := normalizeInt64IDs(modelIDs)
	for _, modelID := range normalized {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO client_api_key_models (client_api_key_id, model_id) VALUES ($1, $2)`,
			clientKeyID,
			modelID,
		); err != nil {
			return fmt.Errorf("insert client api key model: %w", err)
		}
	}

	return nil
}

func normalizeInt64IDs(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(values))
	items := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}

	return items
}

func buildClientCostUsage(item entity.ClientAPIKey, dailyCostUsed float64, monthlyCostUsed float64) *entity.ClientCostUsage {
	now := time.Now().UTC()
	dailyResetAt := nextDayUTC(now)
	monthlyResetAt := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)

	usage := &entity.ClientCostUsage{
		DailyCostUsed:   dailyCostUsed,
		MonthlyCostUsed: monthlyCostUsed,
		DailyResetAt:    &dailyResetAt,
		MonthlyResetAt:  &monthlyResetAt,
	}

	if item.DailyCostLimit > 0 {
		usage.DailyCostRemaining = costRemaining(item.DailyCostLimit, dailyCostUsed)
		usage.DailyCostUsagePercent = costUsagePercent(item.DailyCostLimit, dailyCostUsed)
		usage.IsDailyCostLimited = dailyCostUsed >= item.DailyCostLimit
	}
	if item.MonthlyCostLimit > 0 {
		usage.MonthlyCostRemaining = costRemaining(item.MonthlyCostLimit, monthlyCostUsed)
		usage.MonthlyCostUsagePercent = costUsagePercent(item.MonthlyCostLimit, monthlyCostUsed)
		usage.IsMonthlyCostLimited = monthlyCostUsed >= item.MonthlyCostLimit
	}

	highestUsagePercent := usage.DailyCostUsagePercent
	if usage.MonthlyCostUsagePercent > highestUsagePercent {
		highestUsagePercent = usage.MonthlyCostUsagePercent
	}
	if item.WarningThreshold > 0 && highestUsagePercent >= item.WarningThreshold {
		usage.IsWarningTriggered = true
	}

	return usage
}

func nextDayUTC(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
}

func costRemaining(limit float64, used float64) float64 {
	remaining := limit - used
	if remaining < 0 {
		return 0
	}
	return remaining
}

func costUsagePercent(limit float64, used float64) float64 {
	if limit <= 0 {
		return 0
	}
	return used * 100 / limit
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

	return trimmed[:4] + "***" + trimmed[len(trimmed)-4:]
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

	if input.UserID > 0 {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM client_api_keys cak_filter WHERE cak_filter.id = rl.client_api_key_id AND cak_filter.user_id = "+nextArg(input.UserID)+")")
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
			&item.ClientAPIKeyID,
			&item.ClientAPIKeyName,
			&item.ClientIP,
			&item.RequestMethod,
			&item.RequestPath,
			&item.HTTPStatus,
			&item.Success,
			&item.LatencyMS,
			&item.PromptTokens,
			&item.CompletionTokens,
			&item.TotalTokens,
			&item.ReservedAmount,
			&item.CostAmount,
			&item.BillableAmount,
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

func roundCurrency(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func isCurrencyZero(value float64) bool {
	return math.Abs(value) < 0.0001
}

// --- UserAuthStore ---

func (s *Store) AuthenticateUser(ctx context.Context, email string, password string) (entity.User, error) {
	const query = `
SELECT id, email, full_name, status, wallet_balance::float8, min_available_balance::float8, password_hash, last_login_at, created_at, updated_at
FROM tenant_users
WHERE email = $1
  AND status = 'active'`

	var item entity.User
	var passwordHash string
	err := s.pool.QueryRow(ctx, query, strings.ToLower(strings.TrimSpace(email))).Scan(
		&item.ID, &item.Email, &item.FullName, &item.Status,
		&item.WalletBalance, &item.MinAvailableBalance,
		&passwordHash, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return entity.User{}, fmt.Errorf("authenticate user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return entity.User{}, pgx.ErrNoRows
	}
	return item, nil
}

func (s *Store) UpdateUserLastLogin(ctx context.Context, userID int64) error {
	if _, err := s.pool.Exec(ctx, `UPDATE tenant_users SET last_login_at = NOW() WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("update user last login: %w", err)
	}
	return nil
}

func (s *Store) GetUserByID(ctx context.Context, userID int64) (entity.User, error) {
	const query = `
SELECT id, email, full_name, status, wallet_balance::float8, min_available_balance::float8, last_login_at, created_at, updated_at
FROM tenant_users
WHERE id = $1
  AND status = 'active'`

	var item entity.User
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&item.ID, &item.Email, &item.FullName, &item.Status,
		&item.WalletBalance, &item.MinAvailableBalance,
		&item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return entity.User{}, fmt.Errorf("get user by id: %w", err)
	}
	return item, nil
}

// --- UserClientKeyStore ---

func (s *Store) ListUserClientAPIKeys(ctx context.Context, userID int64) ([]entity.ClientAPIKey, error) {
	const query = `
SELECT cak.id, cak.name, cak.masked_key, cak.status, cak.description,
       COALESCE(cak.user_id, 0), COALESCE(tu.email, ''), COALESCE(tu.wallet_balance, 0)::float8, COALESCE(tu.min_available_balance, 0)::float8,
       cak.rpm_limit, cak.daily_request_limit, cak.daily_token_limit,
       cak.daily_cost_limit::float8, cak.monthly_cost_limit::float8, cak.warning_threshold::float8,
       COALESCE(model_access.allowed_model_ids, ARRAY[]::bigint[]),
       COALESCE(model_access.allowed_models, ARRAY[]::text[]),
       COALESCE(daily_usage.daily_cost_used, 0)::float8,
       COALESCE(monthly_usage.monthly_cost_used, 0)::float8,
       cak.expires_at, cak.last_used_at, cak.last_error_at, cak.last_error_message, cak.created_at, cak.updated_at
FROM client_api_keys cak
LEFT JOIN tenant_users tu ON tu.id = cak.user_id
LEFT JOIN LATERAL (
    SELECT array_agg(cam.model_id ORDER BY cam.model_id) AS allowed_model_ids,
           array_agg(m.public_name ORDER BY m.public_name) AS allowed_models
    FROM client_api_key_models cam
    JOIN models m ON m.id = cam.model_id
    WHERE cam.client_api_key_id = cak.id
) model_access ON TRUE
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(billable_amount), 0) AS daily_cost_used
    FROM request_logs
    WHERE client_api_key_id = cak.id AND created_at >= date_trunc('day', NOW())
) daily_usage ON TRUE
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(billable_amount), 0) AS monthly_cost_used
    FROM request_logs
    WHERE client_api_key_id = cak.id AND created_at >= date_trunc('month', NOW())
) monthly_usage ON TRUE
WHERE cak.user_id = $1
ORDER BY cak.id DESC`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user client api keys: %w", err)
	}
	defer rows.Close()

	items := make([]entity.ClientAPIKey, 0)
	for rows.Next() {
		item, err := scanClientAPIKey(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user client api keys: %w", err)
	}
	return items, nil
}

func (s *Store) CreateUserClientAPIKey(ctx context.Context, input entity.CreateClientAPIKeyInput) (entity.ClientAPIKey, error) {
	return s.CreateClientAPIKey(ctx, input)
}

func (s *Store) DisableUserClientAPIKey(ctx context.Context, userID int64, id int64) (entity.ClientAPIKey, error) {
	var clientKeyID int64
	if err := s.pool.QueryRow(ctx, `
UPDATE client_api_keys SET status = 'disabled'
WHERE id = $1 AND user_id = $2
RETURNING id`, id, userID).Scan(&clientKeyID); err != nil {
		return entity.ClientAPIKey{}, fmt.Errorf("disable user client api key: %w", err)
	}
	return s.getClientAPIKeyByID(ctx, clientKeyID)
}

func (s *Store) GetUserPortalStats(ctx context.Context, userID int64) (entity.UserPortalStats, error) {
	const query = `
WITH request_stats AS (
    SELECT COUNT(*)::bigint AS request_count,
           COUNT(*) FILTER (WHERE rl.success)::bigint AS success_count,
           COUNT(*) FILTER (WHERE NOT rl.success)::bigint AS failed_count,
           COALESCE(AVG(rl.latency_ms), 0)::float8 AS average_latency_ms,
           COALESCE(SUM(rl.prompt_tokens), 0)::bigint AS prompt_tokens,
           COALESCE(SUM(rl.completion_tokens), 0)::bigint AS completion_tokens,
           COALESCE(SUM(rl.total_tokens), 0)::bigint AS total_tokens,
           COALESCE(SUM(rl.cost_amount), 0)::float8 AS cost_amount,
           COALESCE(SUM(rl.billable_amount), 0)::float8 AS billable_amount
    FROM request_logs rl
    JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
    WHERE cak.user_id = $1
),
today_billing AS (
    SELECT COALESCE(SUM(rl.cost_amount), 0)::float8 AS cost_amount,
           COALESCE(SUM(rl.billable_amount), 0)::float8 AS billable_amount
    FROM request_logs rl
    JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
    WHERE cak.user_id = $1 AND rl.created_at >= date_trunc('day', NOW())
),
month_billing AS (
    SELECT COALESCE(SUM(rl.cost_amount), 0)::float8 AS cost_amount,
           COALESCE(SUM(rl.billable_amount), 0)::float8 AS billable_amount
    FROM request_logs rl
    JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
    WHERE cak.user_id = $1 AND rl.created_at >= date_trunc('month', NOW())
),
key_stats AS (
    SELECT COUNT(*)::bigint AS client_key_count,
           COUNT(*) FILTER (WHERE status = 'active')::bigint AS active_client_keys
    FROM client_api_keys WHERE user_id = $1
),
wallet AS (
    SELECT wallet_balance::float8 FROM tenant_users WHERE id = $1
)
SELECT rs.request_count, rs.success_count, rs.failed_count, rs.average_latency_ms,
       rs.prompt_tokens, rs.completion_tokens, rs.total_tokens, rs.cost_amount, rs.billable_amount,
       tb.cost_amount, tb.billable_amount, mb.cost_amount, mb.billable_amount,
       ks.client_key_count, ks.active_client_keys, w.wallet_balance
FROM request_stats rs
CROSS JOIN today_billing tb
CROSS JOIN month_billing mb
CROSS JOIN key_stats ks
CROSS JOIN wallet w`

	var stats entity.UserPortalStats
	if err := s.pool.QueryRow(ctx, query, userID).Scan(
		&stats.RequestCount, &stats.SuccessCount, &stats.FailedCount, &stats.AverageLatencyMS,
		&stats.PromptTokens, &stats.CompletionTokens, &stats.TotalTokens,
		&stats.CostAmount, &stats.BillableAmount,
		&stats.TodayCostAmount, &stats.TodayBillable,
		&stats.MonthCostAmount, &stats.MonthBillable,
		&stats.ClientKeyCount, &stats.ActiveClientKeys, &stats.WalletBalance,
	); err != nil {
		return entity.UserPortalStats{}, fmt.Errorf("query user portal stats: %w", err)
	}
	if stats.RequestCount > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) * 100 / float64(stats.RequestCount)
	}
	return stats, nil
}

func (s *Store) ListUserRequestLogs(ctx context.Context, userID int64, input entity.ListRequestLogsInput) (entity.RequestLogPage, error) {
	input.UserID = userID
	return s.ListRequestLogs(ctx, input)
}

func (s *Store) GetUserRequestLog(ctx context.Context, userID int64, id int64) (entity.RequestLogDetail, error) {
	const query = `
SELECT rl.id, rl.trace_id, rl.request_type, rl.model_public_name, rl.upstream_model,
       COALESCE(rl.provider_id, 0), COALESCE(p.name, ''), COALESCE(rl.provider_key_id, 0), COALESCE(pk.name, ''),
       COALESCE(rl.client_api_key_id, 0), COALESCE(cak.name, ''),
       COALESCE(HOST(rl.client_ip), ''), rl.request_method, rl.request_path, rl.http_status, rl.success, rl.latency_ms,
       rl.prompt_tokens, rl.completion_tokens, rl.total_tokens,
       rl.reserved_amount::float8, rl.cost_amount::float8, rl.billable_amount::float8,
       rl.error_type, rl.error_message, rl.created_at, rl.request_payload, rl.response_payload, rl.metadata
FROM request_logs rl
LEFT JOIN providers p ON p.id = rl.provider_id
LEFT JOIN provider_keys pk ON pk.id = rl.provider_key_id
LEFT JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
WHERE rl.id = $1
  AND EXISTS (
      SELECT 1 FROM client_api_keys cak_filter
      WHERE cak_filter.id = rl.client_api_key_id AND cak_filter.user_id = $2
  )`

	var item entity.RequestLogDetail
	err := s.pool.QueryRow(ctx, query, id, userID).Scan(
		&item.ID, &item.TraceID, &item.RequestType, &item.ModelPublicName, &item.UpstreamModel,
		&item.ProviderID, &item.ProviderName, &item.ProviderKeyID, &item.ProviderKeyName,
		&item.ClientAPIKeyID, &item.ClientAPIKeyName,
		&item.ClientIP, &item.RequestMethod, &item.RequestPath,
		&item.HTTPStatus, &item.Success, &item.LatencyMS,
		&item.PromptTokens, &item.CompletionTokens, &item.TotalTokens,
		&item.ReservedAmount, &item.CostAmount, &item.BillableAmount,
		&item.ErrorType, &item.ErrorMessage, &item.CreatedAt,
		&item.RequestPayload, &item.ResponsePayload, &item.Metadata,
	)
	if err != nil {
		return entity.RequestLogDetail{}, fmt.Errorf("query user request log: %w", err)
	}
	return item, nil
}

func (s *Store) ExportUserRequestLogs(ctx context.Context, userID int64, input entity.ListRequestLogsInput) ([]entity.RequestLog, error) {
	input.UserID = userID
	return s.ExportRequestLogs(ctx, input)
}

// --- RequestLogWriter ---

func (s *Store) DeductUserWalletUsage(ctx context.Context, input entity.UserWalletUsageDebitInput) error {
	if input.ClientAPIKeyID <= 0 || input.BillableAmount <= 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deduct user wallet tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var userID int64
	var before float64
	err = tx.QueryRow(ctx, `
SELECT tu.id, tu.wallet_balance::float8
FROM client_api_keys cak
JOIN tenant_users tu ON tu.id = cak.user_id
WHERE cak.id = $1
FOR UPDATE`, input.ClientAPIKeyID).Scan(&userID, &before)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("load user wallet for debit: %w", err)
	}

	after := before - input.BillableAmount
	if after < 0 {
		after = 0
	}

	if _, err := tx.Exec(ctx, `UPDATE tenant_users SET wallet_balance = $2 WHERE id = $1`, userID, after); err != nil {
		return fmt.Errorf("update user wallet debit: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO tenant_wallet_ledger (
    user_id, direction, amount, balance_before, balance_after, note, operator_type,
    request_log_id, trace_id, model_public_name, total_tokens, reserved_amount, cost_amount, billable_amount
) VALUES ($1, 'debit', $2, $3, $4, $5, 'system', NULLIF($6, 0), $7, $8, $9, $10, $11, $12)`,
		userID, input.BillableAmount, before, after, input.Note,
		input.RequestLogID, input.TraceID, input.ModelPublicName,
		input.TotalTokens, input.ReservedAmount, input.CostAmount, input.BillableAmount,
	); err != nil {
		return fmt.Errorf("insert debit wallet ledger: %w", err)
	}

	return tx.Commit(ctx)
}

// --- Alert store methods ---

func (s *Store) GetUserBillingReconciliation(ctx context.Context) ([]entity.UserBillingReconciliation, error) {
	const query = `
WITH ledger AS (
    SELECT user_id,
           COALESCE(SUM(CASE WHEN direction = 'credit' THEN amount ELSE 0 END), 0)::float8 AS ledger_credit_amount,
           COALESCE(SUM(CASE WHEN direction = 'debit' THEN amount ELSE 0 END), 0)::float8 AS ledger_debit_amount
    FROM tenant_wallet_ledger
    GROUP BY user_id
),
logs AS (
    SELECT cak.user_id,
           COALESCE(SUM(rl.billable_amount), 0)::float8 AS log_billable_amount,
           COALESCE(SUM(rl.cost_amount), 0)::float8 AS log_cost_amount
    FROM request_logs rl
    JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
    GROUP BY cak.user_id
)
SELECT tu.id, tu.email, tu.wallet_balance::float8,
       COALESCE(l.ledger_credit_amount, 0)::float8,
       COALESCE(l.ledger_debit_amount, 0)::float8,
       (COALESCE(l.ledger_credit_amount, 0) - COALESCE(l.ledger_debit_amount, 0))::float8,
       COALESCE(g.log_billable_amount, 0)::float8,
       COALESCE(g.log_cost_amount, 0)::float8
FROM tenant_users tu
LEFT JOIN ledger l ON l.user_id = tu.id
LEFT JOIN logs g ON g.user_id = tu.id
ORDER BY tu.id DESC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query user billing reconciliation: %w", err)
	}
	defer rows.Close()

	items := make([]entity.UserBillingReconciliation, 0)
	for rows.Next() {
		var item entity.UserBillingReconciliation
		if err := rows.Scan(
			&item.UserID, &item.UserEmail, &item.WalletBalance,
			&item.LedgerCreditAmount, &item.LedgerDebitAmount, &item.LedgerNetAmount,
			&item.LogBillableAmount, &item.LogCostAmount,
		); err != nil {
			return nil, fmt.Errorf("scan user billing reconciliation: %w", err)
		}
		item.WalletVsLedgerDiff = roundCurrency(item.WalletBalance - item.LedgerNetAmount)
		item.LedgerVsLogsDiff = roundCurrency(item.LedgerDebitAmount - item.LogBillableAmount)
		item.IsWalletBalanced = isCurrencyZero(item.WalletVsLedgerDiff)
		item.IsBillingBalanced = isCurrencyZero(item.LedgerVsLogsDiff)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user billing reconciliation: %w", err)
	}
	return items, nil
}

func (s *Store) GetUserWalletBlockAnomalies(ctx context.Context, since time.Time, walletBlockThreshold int, reserveBlockThreshold int) ([]entity.UserWalletBlockAnomaly, error) {
	const query = `
SELECT tu.id, tu.email,
       COUNT(*) FILTER (WHERE rl.error_type IN ('wallet_empty', 'wallet_below_minimum'))::bigint AS wallet_blocked_count,
       COUNT(*) FILTER (WHERE rl.error_type = 'wallet_reserve_insufficient')::bigint AS reserve_blocked_count
FROM request_logs rl
JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
JOIN tenant_users tu ON tu.id = cak.user_id
WHERE rl.created_at >= $1
  AND rl.error_type IN ('wallet_empty', 'wallet_below_minimum', 'wallet_reserve_insufficient')
GROUP BY tu.id, tu.email
ORDER BY tu.id DESC`

	rows, err := s.pool.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("query user wallet block anomalies: %w", err)
	}
	defer rows.Close()

	items := make([]entity.UserWalletBlockAnomaly, 0)
	for rows.Next() {
		var item entity.UserWalletBlockAnomaly
		if err := rows.Scan(&item.UserID, &item.UserEmail, &item.WalletBlockedCount, &item.ReserveBlockedCount); err != nil {
			return nil, fmt.Errorf("scan user wallet block anomaly: %w", err)
		}
		item.IsWalletBlockedAnomalous = item.WalletBlockedCount >= int64(walletBlockThreshold)
		item.IsReserveBlockedAnomalous = item.ReserveBlockedCount >= int64(reserveBlockThreshold)
		if item.IsWalletBlockedAnomalous || item.IsReserveBlockedAnomalous {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user wallet block anomalies: %w", err)
	}
	return items, nil
}

func (s *Store) GetUserBillingDebitAnomalies(ctx context.Context, since time.Time, minCount int, minBillableAmount float64) ([]entity.UserBillingDebitAnomaly, error) {
	const query = `
SELECT tu.id, tu.email,
       COUNT(*)::bigint AS missing_debit_count,
       COALESCE(SUM(rl.billable_amount), 0)::float8 AS missing_billable_amount,
       COALESCE(SUM(rl.cost_amount), 0)::float8 AS missing_cost_amount
FROM request_logs rl
JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
JOIN tenant_users tu ON tu.id = cak.user_id
LEFT JOIN tenant_wallet_ledger twl ON twl.request_log_id = rl.id AND twl.direction = 'debit'
WHERE rl.created_at >= $1
  AND rl.success = TRUE
  AND rl.billable_amount > 0
  AND twl.id IS NULL
GROUP BY tu.id, tu.email
ORDER BY missing_billable_amount DESC, missing_debit_count DESC, tu.id ASC`

	rows, err := s.pool.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("query user billing debit anomalies: %w", err)
	}
	defer rows.Close()

	items := make([]entity.UserBillingDebitAnomaly, 0)
	for rows.Next() {
		var item entity.UserBillingDebitAnomaly
		if err := rows.Scan(
			&item.UserID, &item.UserEmail,
			&item.MissingDebitCount, &item.MissingBillableAmount, &item.MissingCostAmount,
		); err != nil {
			return nil, fmt.Errorf("scan user billing debit anomaly: %w", err)
		}
		item.IsCountAnomalous = item.MissingDebitCount >= int64(minCount)
		item.IsBillableAmountAnomalous = item.MissingBillableAmount >= minBillableAmount
		if item.IsCountAnomalous || item.IsBillableAmountAnomalous {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user billing debit anomalies: %w", err)
	}
	return items, nil
}
