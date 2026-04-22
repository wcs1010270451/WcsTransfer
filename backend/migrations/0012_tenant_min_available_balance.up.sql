ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS min_available_balance NUMERIC(14, 4) NOT NULL DEFAULT 0.0100;

ALTER TABLE tenants
    ADD CONSTRAINT chk_tenants_min_available_balance_non_negative
        CHECK (min_available_balance >= 0);

COMMENT ON COLUMN tenants.min_available_balance IS '租户最低可调用余额门槛（美元）';
