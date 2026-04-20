ALTER TABLE models
    ADD COLUMN IF NOT EXISTS cost_input_per_1m NUMERIC(12, 6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cost_output_per_1m NUMERIC(12, 6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sale_input_per_1m NUMERIC(12, 6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sale_output_per_1m NUMERIC(12, 6) NOT NULL DEFAULT 0;

UPDATE models
SET cost_input_per_1m = input_cost_per_1m,
    cost_output_per_1m = output_cost_per_1m,
    sale_input_per_1m = input_cost_per_1m,
    sale_output_per_1m = output_cost_per_1m
WHERE cost_input_per_1m = 0
  AND cost_output_per_1m = 0
  AND sale_input_per_1m = 0
  AND sale_output_per_1m = 0;

ALTER TABLE models
    ADD CONSTRAINT chk_models_cost_input_per_1m_non_negative
        CHECK (cost_input_per_1m >= 0),
    ADD CONSTRAINT chk_models_cost_output_per_1m_non_negative
        CHECK (cost_output_per_1m >= 0),
    ADD CONSTRAINT chk_models_sale_input_per_1m_non_negative
        CHECK (sale_input_per_1m >= 0),
    ADD CONSTRAINT chk_models_sale_output_per_1m_non_negative
        CHECK (sale_output_per_1m >= 0);

ALTER TABLE request_logs
    ADD COLUMN IF NOT EXISTS cost_amount NUMERIC(18, 8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS billable_amount NUMERIC(18, 8) NOT NULL DEFAULT 0;

UPDATE request_logs
SET cost_amount = estimated_cost,
    billable_amount = estimated_cost
WHERE cost_amount = 0
  AND billable_amount = 0
  AND estimated_cost > 0;

COMMENT ON COLUMN models.cost_input_per_1m IS '每百万输入 token 上游成本价（美元）';
COMMENT ON COLUMN models.cost_output_per_1m IS '每百万输出 token 上游成本价（美元）';
COMMENT ON COLUMN models.sale_input_per_1m IS '每百万输入 token 对客售价（美元）';
COMMENT ON COLUMN models.sale_output_per_1m IS '每百万输出 token 对客售价（美元）';
COMMENT ON COLUMN request_logs.cost_amount IS '本次请求上游成本金额（美元）';
COMMENT ON COLUMN request_logs.billable_amount IS '本次请求对客计费金额（美元）';
