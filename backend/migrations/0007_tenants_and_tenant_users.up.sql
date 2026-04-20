CREATE TABLE tenants (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    slug VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_tenants_status CHECK (status IN ('active', 'disabled'))
);

CREATE TRIGGER trg_tenants_updated_at
BEFORE UPDATE ON tenants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE tenant_users (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(128) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_tenant_users_status CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX idx_tenant_users_tenant_id ON tenant_users(tenant_id);

CREATE TRIGGER trg_tenant_users_updated_at
BEFORE UPDATE ON tenant_users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE client_api_keys
    ADD COLUMN IF NOT EXISTS tenant_id BIGINT NULL REFERENCES tenants(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS created_by_user_id BIGINT NULL REFERENCES tenant_users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_client_api_keys_tenant_id ON client_api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_client_api_keys_created_by_user_id ON client_api_keys(created_by_user_id);

COMMENT ON TABLE tenants IS '租户表';
COMMENT ON COLUMN tenants.id IS '主键 ID';
COMMENT ON COLUMN tenants.name IS '租户名称';
COMMENT ON COLUMN tenants.slug IS '租户唯一标识';
COMMENT ON COLUMN tenants.status IS '租户状态';
COMMENT ON COLUMN tenants.created_at IS '创建时间';
COMMENT ON COLUMN tenants.updated_at IS '更新时间';

COMMENT ON TABLE tenant_users IS '租户用户表';
COMMENT ON COLUMN tenant_users.id IS '主键 ID';
COMMENT ON COLUMN tenant_users.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tenant_users.email IS '登录邮箱';
COMMENT ON COLUMN tenant_users.password_hash IS '密码哈希';
COMMENT ON COLUMN tenant_users.full_name IS '用户名称';
COMMENT ON COLUMN tenant_users.status IS '用户状态';
COMMENT ON COLUMN tenant_users.last_login_at IS '最后登录时间';
COMMENT ON COLUMN tenant_users.created_at IS '创建时间';
COMMENT ON COLUMN tenant_users.updated_at IS '更新时间';

COMMENT ON COLUMN client_api_keys.tenant_id IS '所属租户 ID，为空表示平台管理员托管的 key';
COMMENT ON COLUMN client_api_keys.created_by_user_id IS '创建该 client key 的租户用户 ID';
