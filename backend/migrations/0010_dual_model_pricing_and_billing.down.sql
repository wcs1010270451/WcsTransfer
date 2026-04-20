ALTER TABLE request_logs
    DROP COLUMN IF EXISTS billable_amount,
    DROP COLUMN IF EXISTS cost_amount;

ALTER TABLE models
    DROP CONSTRAINT IF EXISTS chk_models_sale_output_per_1m_non_negative,
    DROP CONSTRAINT IF EXISTS chk_models_sale_input_per_1m_non_negative,
    DROP CONSTRAINT IF EXISTS chk_models_cost_output_per_1m_non_negative,
    DROP CONSTRAINT IF EXISTS chk_models_cost_input_per_1m_non_negative,
    DROP COLUMN IF EXISTS sale_output_per_1m,
    DROP COLUMN IF EXISTS sale_input_per_1m,
    DROP COLUMN IF EXISTS cost_output_per_1m,
    DROP COLUMN IF EXISTS cost_input_per_1m;
