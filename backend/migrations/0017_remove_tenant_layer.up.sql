-- Step 1: 给 tenant_users 加钱包字段
ALTER TABLE tenant_users
    ADD COLUMN IF NOT EXISTS wallet_balance NUMERIC(18, 8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS min_available_balance NUMERIC(18, 8) NOT NULL DEFAULT 0;

ALTER TABLE tenant_users
    DROP CONSTRAINT IF EXISTS chk_tenant_users_wallet_non_negative;
ALTER TABLE tenant_users
    ADD CONSTRAINT chk_tenant_users_wallet_non_negative
        CHECK (wallet_balance >= 0);

ALTER TABLE tenant_users
    DROP CONSTRAINT IF EXISTS chk_tenant_users_min_balance_non_negative;
ALTER TABLE tenant_users
    ADD CONSTRAINT chk_tenant_users_min_balance_non_negative
        CHECK (min_available_balance >= 0);

-- Step 2: tenant_wallet_ledger 把 tenant_id 换成 user_id（指向 tenant_users）
ALTER TABLE tenant_wallet_ledger DROP CONSTRAINT IF EXISTS tenant_wallet_ledger_tenant_id_fkey;
DROP INDEX IF EXISTS idx_tenant_wallet_ledger_tenant_created_at;

-- 条件重命名（幂等：列可能已在上次失败的执行中改过名了）
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tenant_wallet_ledger' AND column_name = 'tenant_id'
    ) THEN
        ALTER TABLE tenant_wallet_ledger RENAME COLUMN tenant_id TO user_id;
    END IF;
END $$;

-- 将旧 tenant_id 值映射到对应租户的第一个用户 ID
-- （老架构中 ledger.tenant_id 指向 tenants.id，新架构中指向 tenant_users.id）
UPDATE tenant_wallet_ledger twl
SET user_id = (
    SELECT tu.id
    FROM tenant_users tu
    WHERE tu.tenant_id = twl.user_id
    ORDER BY tu.id
    LIMIT 1
)
WHERE EXISTS (
    SELECT 1 FROM tenant_users tu WHERE tu.tenant_id = twl.user_id
);

-- 删除无法映射到任何用户的孤立账本记录
DELETE FROM tenant_wallet_ledger
WHERE user_id NOT IN (SELECT id FROM tenant_users);

ALTER TABLE tenant_wallet_ledger DROP CONSTRAINT IF EXISTS tenant_wallet_ledger_user_id_fkey;
ALTER TABLE tenant_wallet_ledger
    ADD CONSTRAINT tenant_wallet_ledger_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES tenant_users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_tenant_wallet_ledger_user_id_created_at
    ON tenant_wallet_ledger (user_id, created_at DESC);

-- Step 3: client_api_keys 删掉 tenant_id，把 created_by_user_id 改名为 user_id
-- 先把 created_by_user_id 为 NULL 的行用 tenant_id 对应的第一个用户填充
UPDATE client_api_keys cak
SET created_by_user_id = (
    SELECT tu.id
    FROM tenant_users tu
    WHERE tu.tenant_id = cak.tenant_id
    ORDER BY tu.id
    LIMIT 1
)
WHERE cak.created_by_user_id IS NULL
  AND cak.tenant_id IS NOT NULL
  AND EXISTS (
      SELECT 1 FROM tenant_users tu WHERE tu.tenant_id = cak.tenant_id
  );

DROP INDEX IF EXISTS idx_client_api_keys_tenant_id;
ALTER TABLE client_api_keys DROP COLUMN IF EXISTS tenant_id;
DROP INDEX IF EXISTS idx_client_api_keys_created_by_user_id;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'client_api_keys' AND column_name = 'created_by_user_id'
    ) THEN
        ALTER TABLE client_api_keys RENAME COLUMN created_by_user_id TO user_id;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_client_api_keys_user_id ON client_api_keys(user_id);

-- Step 4: 删掉 tenant_users.tenant_id，然后删 tenants 表
DROP INDEX IF EXISTS idx_tenant_users_tenant_id;
ALTER TABLE tenant_users DROP COLUMN IF EXISTS tenant_id;
DROP TABLE IF EXISTS tenants CASCADE;
