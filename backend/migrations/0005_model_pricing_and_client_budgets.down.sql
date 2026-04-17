ALTER TABLE models
    DROP CONSTRAINT IF EXISTS chk_models_output_cost_per_1m_non_negative,
    DROP CONSTRAINT IF EXISTS chk_models_input_cost_per_1m_non_negative,
    DROP COLUMN IF EXISTS output_cost_per_1m,
    DROP COLUMN IF EXISTS input_cost_per_1m;

ALTER TABLE client_api_keys
    DROP CONSTRAINT IF EXISTS chk_client_api_keys_warning_threshold_range,
    DROP CONSTRAINT IF EXISTS chk_client_api_keys_monthly_cost_limit_non_negative,
    DROP CONSTRAINT IF EXISTS chk_client_api_keys_daily_cost_limit_non_negative,
    DROP COLUMN IF EXISTS warning_threshold,
    DROP COLUMN IF EXISTS monthly_cost_limit,
    DROP COLUMN IF EXISTS daily_cost_limit;
