DROP TABLE IF EXISTS tenant_wallet_ledger;

ALTER TABLE tenants
    DROP CONSTRAINT IF EXISTS chk_tenants_wallet_balance_non_negative;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS wallet_balance;
