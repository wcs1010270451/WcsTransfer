ALTER TABLE tenants
    DROP CONSTRAINT IF EXISTS chk_tenants_min_available_balance_non_negative;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS min_available_balance;
