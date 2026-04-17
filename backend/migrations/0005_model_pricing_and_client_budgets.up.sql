ALTER TABLE client_api_keys
    ADD COLUMN IF NOT EXISTS daily_cost_limit NUMERIC(12, 4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS monthly_cost_limit NUMERIC(12, 4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS warning_threshold NUMERIC(5, 2) NOT NULL DEFAULT 80;

ALTER TABLE client_api_keys
    ADD CONSTRAINT chk_client_api_keys_daily_cost_limit_non_negative
        CHECK (daily_cost_limit >= 0),
    ADD CONSTRAINT chk_client_api_keys_monthly_cost_limit_non_negative
        CHECK (monthly_cost_limit >= 0),
    ADD CONSTRAINT chk_client_api_keys_warning_threshold_range
        CHECK (warning_threshold >= 0 AND warning_threshold <= 100);

ALTER TABLE models
    ADD COLUMN IF NOT EXISTS input_cost_per_1m NUMERIC(12, 6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS output_cost_per_1m NUMERIC(12, 6) NOT NULL DEFAULT 0;

ALTER TABLE models
    ADD CONSTRAINT chk_models_input_cost_per_1m_non_negative
        CHECK (input_cost_per_1m >= 0),
    ADD CONSTRAINT chk_models_output_cost_per_1m_non_negative
        CHECK (output_cost_per_1m >= 0);

COMMENT ON COLUMN client_api_keys.daily_cost_limit IS '客户端每日成本预算上限（美元）';
COMMENT ON COLUMN client_api_keys.monthly_cost_limit IS '客户端每月成本预算上限（美元）';
COMMENT ON COLUMN client_api_keys.warning_threshold IS '预算预警阈值百分比';
COMMENT ON COLUMN models.input_cost_per_1m IS '每百万输入 token 成本（美元）';
COMMENT ON COLUMN models.output_cost_per_1m IS '每百万输出 token 成本（美元）';
