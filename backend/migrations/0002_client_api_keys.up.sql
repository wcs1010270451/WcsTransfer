CREATE TABLE client_api_keys (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    masked_key VARCHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    description TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NULL,
    last_used_at TIMESTAMPTZ NULL,
    last_error_at TIMESTAMPTZ NULL,
    last_error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_client_api_keys_status CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX idx_client_api_keys_status ON client_api_keys(status);
CREATE INDEX idx_client_api_keys_expires_at ON client_api_keys(expires_at);

CREATE TRIGGER trg_client_api_keys_updated_at
BEFORE UPDATE ON client_api_keys
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE request_logs
ADD COLUMN client_api_key_id BIGINT NULL REFERENCES client_api_keys(id) ON DELETE SET NULL;

CREATE INDEX idx_request_logs_client_api_key_id_created_at ON request_logs(client_api_key_id, created_at DESC);

COMMENT ON TABLE client_api_keys IS '业务侧接入网关时使用的客户端 API Key 表';
COMMENT ON COLUMN client_api_keys.id IS '主键 ID';
COMMENT ON COLUMN client_api_keys.name IS '客户端 key 的显示名称';
COMMENT ON COLUMN client_api_keys.key_hash IS '客户端 key 的哈希值，仅用于鉴权比对';
COMMENT ON COLUMN client_api_keys.masked_key IS '脱敏后的 key 展示值';
COMMENT ON COLUMN client_api_keys.status IS '客户端 key 状态，active 表示启用，disabled 表示禁用';
COMMENT ON COLUMN client_api_keys.description IS '客户端 key 备注信息';
COMMENT ON COLUMN client_api_keys.expires_at IS '客户端 key 过期时间，为空表示不过期';
COMMENT ON COLUMN client_api_keys.last_used_at IS '最近一次成功鉴权时间';
COMMENT ON COLUMN client_api_keys.last_error_at IS '最近一次鉴权或调用错误时间';
COMMENT ON COLUMN client_api_keys.last_error_message IS '最近一次错误信息';
COMMENT ON COLUMN client_api_keys.created_at IS '记录创建时间';
COMMENT ON COLUMN client_api_keys.updated_at IS '记录最后更新时间';

COMMENT ON COLUMN request_logs.client_api_key_id IS '发起本次业务请求的客户端 API Key ID';
