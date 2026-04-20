ALTER TABLE tenants
    ALTER COLUMN status SET DEFAULT 'active';

ALTER TABLE tenants
    DROP CONSTRAINT IF EXISTS chk_tenants_max_client_keys_non_negative;

ALTER TABLE tenants
    DROP CONSTRAINT IF EXISTS chk_tenants_status;

ALTER TABLE tenants
    ADD CONSTRAINT chk_tenants_status
        CHECK (status IN ('active', 'disabled'));

ALTER TABLE tenants
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS max_client_keys;
