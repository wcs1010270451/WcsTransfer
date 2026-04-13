package platform

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"wcstransfer/backend/internal/config"
)

type CheckResult struct {
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

type Dependencies struct {
	config   config.Config
	Postgres *pgxpool.Pool
	Redis    *redis.Client
}

func New(ctx context.Context, cfg config.Config) (*Dependencies, error) {
	deps := &Dependencies{config: cfg}

	if cfg.DatabaseURL != "" {
		poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("parse database config: %w", err)
		}

		poolConfig.MaxConns = cfg.DatabaseMaxConns
		poolConfig.MinConns = cfg.DatabaseMinConns

		pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			return nil, fmt.Errorf("connect postgres: %w", err)
		}

		if err := deps.pingPostgres(ctx, pool); err != nil {
			pool.Close()
			return nil, err
		}

		deps.Postgres = pool
	}

	if cfg.RedisAddr != "" {
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		if err := deps.pingRedis(ctx, client); err != nil {
			_ = client.Close()
			if deps.Postgres != nil {
				deps.Postgres.Close()
			}
			return nil, err
		}

		deps.Redis = client
	}

	return deps, nil
}

func (d *Dependencies) Close() {
	if d.Redis != nil {
		_ = d.Redis.Close()
	}

	if d.Postgres != nil {
		d.Postgres.Close()
	}
}

func (d *Dependencies) Health(ctx context.Context) map[string]CheckResult {
	results := map[string]CheckResult{
		"postgres": {Status: "disabled"},
		"redis":    {Status: "disabled"},
	}

	if d.Postgres != nil {
		if err := d.pingPostgres(ctx, d.Postgres); err != nil {
			results["postgres"] = CheckResult{Status: "down", Details: err.Error()}
		} else {
			results["postgres"] = CheckResult{Status: "up"}
		}
	}

	if d.Redis != nil {
		if err := d.pingRedis(ctx, d.Redis); err != nil {
			results["redis"] = CheckResult{Status: "down", Details: err.Error()}
		} else {
			results["redis"] = CheckResult{Status: "up"}
		}
	}

	return results
}

func (d *Dependencies) pingPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, d.config.DependencyTimeout)
	defer cancel()

	if err := pool.Ping(timeoutCtx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	return nil
}

func (d *Dependencies) pingRedis(ctx context.Context, client *redis.Client) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, d.config.DependencyTimeout)
	defer cancel()

	if err := client.Ping(timeoutCtx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}

	return nil
}
