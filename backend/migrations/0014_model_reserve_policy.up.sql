ALTER TABLE models
    ADD COLUMN IF NOT EXISTS reserve_multiplier NUMERIC(10, 4) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS reserve_min_amount NUMERIC(12, 4) NOT NULL DEFAULT 0;

ALTER TABLE models
    ADD CONSTRAINT chk_models_reserve_multiplier_positive
        CHECK (reserve_multiplier > 0),
    ADD CONSTRAINT chk_models_reserve_min_amount_non_negative
        CHECK (reserve_min_amount >= 0);

COMMENT ON COLUMN models.reserve_multiplier IS '请求前预留金额倍率，基于估算计费金额放大';
COMMENT ON COLUMN models.reserve_min_amount IS '请求前最低预留金额（美元）';
