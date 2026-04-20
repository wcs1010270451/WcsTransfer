ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS max_client_keys INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';

ALTER TABLE tenants
    DROP CONSTRAINT IF EXISTS chk_tenants_status;

ALTER TABLE tenants
    ADD CONSTRAINT chk_tenants_status
        CHECK (status IN ('pending', 'active', 'disabled'));

ALTER TABLE tenants
    ADD CONSTRAINT chk_tenants_max_client_keys_non_negative
        CHECK (max_client_keys >= 0);

ALTER TABLE tenants
    ALTER COLUMN status SET DEFAULT 'pending';

COMMENT ON COLUMN tenants.max_client_keys IS '租户可创建的 client key 最大数量，0 表示不允许创建';
COMMENT ON COLUMN tenants.notes IS '管理员备注';
COMMENT ON COLUMN tenants.status IS '租户状态，pending 表示待审核，active 表示可用，disabled 表示禁用';
