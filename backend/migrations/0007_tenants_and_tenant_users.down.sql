ALTER TABLE client_api_keys
    DROP COLUMN IF EXISTS created_by_user_id,
    DROP COLUMN IF EXISTS tenant_id;

DROP TABLE IF EXISTS tenant_users;
DROP TABLE IF EXISTS tenants;
