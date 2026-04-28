-- 回滚：重建 tenants 表，恢复各表的 tenant_id 列
-- 注意：已有数据无法自动恢复关联关系，仅恢复结构

-- Step 1: 重建 tenants 表
CREATE TABLE IF NOT EXISTS tenants (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    slug VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    max_client_keys INT NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    wallet_balance NUMERIC(18, 8) NOT NULL DEFAULT 0,
    min_available_balance NUMERIC(18, 8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_tenants_status CHECK (status IN ('pending', 'active', 'disabled'))
);

-- Step 2: 还原 tenant_users
ALTER TABLE tenant_users
    DROP CONSTRAINT IF EXISTS chk_tenant_users_wallet_non_negative,
    DROP CONSTRAINT IF EXISTS chk_tenant_users_min_balance_non_negative,
    DROP COLUMN IF EXISTS wallet_balance,
    DROP COLUMN IF EXISTS min_available_balance;

ALTER TABLE tenant_users
    ADD COLUMN IF NOT EXISTS tenant_id BIGINT NULL REFERENCES tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_tenant_users_tenant_id ON tenant_users(tenant_id);

-- Step 3: 还原 tenant_wallet_ledger
ALTER TABLE tenant_wallet_ledger DROP CONSTRAINT IF EXISTS tenant_wallet_ledger_user_id_fkey;
DROP INDEX IF EXISTS idx_tenant_wallet_ledger_user_id_created_at;
ALTER TABLE tenant_wallet_ledger RENAME COLUMN user_id TO tenant_id;
ALTER TABLE tenant_wallet_ledger
    ADD CONSTRAINT tenant_wallet_ledger_tenant_id_fkey
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_tenant_wallet_ledger_tenant_created_at
    ON tenant_wallet_ledger (tenant_id, created_at DESC);

-- Step 4: 还原 client_api_keys
DROP INDEX IF EXISTS idx_client_api_keys_user_id;
ALTER TABLE client_api_keys RENAME COLUMN user_id TO created_by_user_id;
ALTER TABLE client_api_keys
    ADD COLUMN IF NOT EXISTS tenant_id BIGINT NULL REFERENCES tenants(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_client_api_keys_tenant_id ON client_api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_client_api_keys_created_by_user_id ON client_api_keys(created_by_user_id);
