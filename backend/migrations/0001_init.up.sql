CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE admin_users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_admin_users_status CHECK (status IN ('active', 'disabled'))
);

CREATE TRIGGER trg_admin_users_updated_at
BEFORE UPDATE ON admin_users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE providers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    slug VARCHAR(64) NOT NULL UNIQUE,
    provider_type VARCHAR(32) NOT NULL DEFAULT 'openai_compatible',
    base_url TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    description TEXT NOT NULL DEFAULT '',
    extra_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_providers_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_providers_type CHECK (provider_type IN ('openai_compatible', 'openai', 'azure_openai', 'custom'))
);

CREATE TRIGGER trg_providers_updated_at
BEFORE UPDATE ON providers
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE provider_keys (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    api_key TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    weight INTEGER NOT NULL DEFAULT 100,
    priority INTEGER NOT NULL DEFAULT 100,
    rpm_limit INTEGER NOT NULL DEFAULT 0,
    tpm_limit INTEGER NOT NULL DEFAULT 0,
    current_rpm INTEGER NOT NULL DEFAULT 0,
    current_tpm BIGINT NOT NULL DEFAULT 0,
    last_used_at TIMESTAMPTZ NULL,
    last_error_at TIMESTAMPTZ NULL,
    last_error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_provider_keys_status CHECK (status IN ('active', 'disabled', 'rate_limited', 'invalid')),
    CONSTRAINT chk_provider_keys_weight CHECK (weight >= 0),
    CONSTRAINT chk_provider_keys_priority CHECK (priority >= 0),
    CONSTRAINT chk_provider_keys_rpm_limit CHECK (rpm_limit >= 0),
    CONSTRAINT chk_provider_keys_tpm_limit CHECK (tpm_limit >= 0),
    CONSTRAINT uq_provider_keys_provider_name UNIQUE (provider_id, name)
);

CREATE INDEX idx_provider_keys_provider_id ON provider_keys(provider_id);
CREATE INDEX idx_provider_keys_status_priority ON provider_keys(status, priority, weight);

CREATE TRIGGER trg_provider_keys_updated_at
BEFORE UPDATE ON provider_keys
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE models (
    id BIGSERIAL PRIMARY KEY,
    public_name VARCHAR(128) NOT NULL UNIQUE,
    provider_id BIGINT NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    upstream_model VARCHAR(128) NOT NULL,
    route_strategy VARCHAR(32) NOT NULL DEFAULT 'fixed',
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    max_tokens INTEGER NOT NULL DEFAULT 0,
    temperature NUMERIC(4,2) NOT NULL DEFAULT 0,
    timeout_seconds INTEGER NOT NULL DEFAULT 120,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_models_route_strategy CHECK (route_strategy IN ('fixed', 'round_robin', 'failover')),
    CONSTRAINT chk_models_max_tokens CHECK (max_tokens >= 0),
    CONSTRAINT chk_models_timeout_seconds CHECK (timeout_seconds > 0)
);

CREATE INDEX idx_models_provider_id ON models(provider_id);
CREATE INDEX idx_models_is_enabled ON models(is_enabled);

CREATE TRIGGER trg_models_updated_at
BEFORE UPDATE ON models
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE request_logs (
    id BIGSERIAL PRIMARY KEY,
    trace_id VARCHAR(64) NOT NULL,
    request_type VARCHAR(32) NOT NULL DEFAULT 'chat_completions',
    model_public_name VARCHAR(128) NOT NULL DEFAULT '',
    upstream_model VARCHAR(128) NOT NULL DEFAULT '',
    provider_id BIGINT NULL REFERENCES providers(id) ON DELETE SET NULL,
    provider_key_id BIGINT NULL REFERENCES provider_keys(id) ON DELETE SET NULL,
    admin_user_id BIGINT NULL REFERENCES admin_users(id) ON DELETE SET NULL,
    client_ip INET NULL,
    request_method VARCHAR(16) NOT NULL DEFAULT 'POST',
    request_path VARCHAR(255) NOT NULL DEFAULT '',
    http_status INTEGER NOT NULL DEFAULT 0,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    estimated_cost NUMERIC(18,8) NOT NULL DEFAULT 0,
    error_type VARCHAR(64) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    request_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_request_logs_latency_ms CHECK (latency_ms >= 0),
    CONSTRAINT chk_request_logs_prompt_tokens CHECK (prompt_tokens >= 0),
    CONSTRAINT chk_request_logs_completion_tokens CHECK (completion_tokens >= 0),
    CONSTRAINT chk_request_logs_total_tokens CHECK (total_tokens >= 0)
);

CREATE INDEX idx_request_logs_trace_id ON request_logs(trace_id);
CREATE INDEX idx_request_logs_created_at ON request_logs(created_at DESC);
CREATE INDEX idx_request_logs_provider_id_created_at ON request_logs(provider_id, created_at DESC);
CREATE INDEX idx_request_logs_provider_key_id_created_at ON request_logs(provider_key_id, created_at DESC);
CREATE INDEX idx_request_logs_model_public_name_created_at ON request_logs(model_public_name, created_at DESC);
CREATE INDEX idx_request_logs_success_created_at ON request_logs(success, created_at DESC);

COMMENT ON FUNCTION set_updated_at() IS '在更新记录前自动刷新 updated_at 时间';

COMMENT ON TABLE admin_users IS '管理后台使用的管理员账号表';
COMMENT ON COLUMN admin_users.id IS '主键 ID';
COMMENT ON COLUMN admin_users.username IS '管理员唯一登录名';
COMMENT ON COLUMN admin_users.password_hash IS '管理员登录密码的哈希值';
COMMENT ON COLUMN admin_users.display_name IS '管理员显示名称';
COMMENT ON COLUMN admin_users.status IS '账号状态，active 表示启用，disabled 表示禁用';
COMMENT ON COLUMN admin_users.last_login_at IS '最后一次成功登录时间';
COMMENT ON COLUMN admin_users.created_at IS '记录创建时间';
COMMENT ON COLUMN admin_users.updated_at IS '记录最后更新时间';

COMMENT ON TABLE providers IS '上游大模型服务提供方配置表';
COMMENT ON COLUMN providers.id IS '主键 ID';
COMMENT ON COLUMN providers.name IS '上游服务提供方显示名称';
COMMENT ON COLUMN providers.slug IS '系统内唯一标识';
COMMENT ON COLUMN providers.provider_type IS '提供方协议类型，例如 openai_compatible 或 azure_openai';
COMMENT ON COLUMN providers.base_url IS '上游服务接口基础地址';
COMMENT ON COLUMN providers.status IS '提供方状态，active 表示启用，disabled 表示禁用';
COMMENT ON COLUMN providers.description IS '提供方描述信息';
COMMENT ON COLUMN providers.extra_config IS '提供方扩展配置，使用 JSON 存储';
COMMENT ON COLUMN providers.created_at IS '记录创建时间';
COMMENT ON COLUMN providers.updated_at IS '记录最后更新时间';

COMMENT ON TABLE provider_keys IS '每个提供方对应的 API Key 配置表';
COMMENT ON COLUMN provider_keys.id IS '主键 ID';
COMMENT ON COLUMN provider_keys.provider_id IS '所属提供方 ID';
COMMENT ON COLUMN provider_keys.name IS 'Key 的别名，供后台展示';
COMMENT ON COLUMN provider_keys.api_key IS '上游服务真实 API Key';
COMMENT ON COLUMN provider_keys.status IS 'Key 状态，可为 active、disabled、rate_limited、invalid';
COMMENT ON COLUMN provider_keys.weight IS '负载均衡权重';
COMMENT ON COLUMN provider_keys.priority IS '故障切换优先级';
COMMENT ON COLUMN provider_keys.rpm_limit IS '每分钟请求数限制';
COMMENT ON COLUMN provider_keys.tpm_limit IS '每分钟 Token 数限制';
COMMENT ON COLUMN provider_keys.current_rpm IS '当前观测到的每分钟请求数';
COMMENT ON COLUMN provider_keys.current_tpm IS '当前观测到的每分钟 Token 数';
COMMENT ON COLUMN provider_keys.last_used_at IS '最近一次成功使用时间';
COMMENT ON COLUMN provider_keys.last_error_at IS '最近一次上游报错时间';
COMMENT ON COLUMN provider_keys.last_error_message IS '最近一次错误信息';
COMMENT ON COLUMN provider_keys.created_at IS '记录创建时间';
COMMENT ON COLUMN provider_keys.updated_at IS '记录最后更新时间';

COMMENT ON TABLE models IS '网关对外暴露的模型映射表';
COMMENT ON COLUMN models.id IS '主键 ID';
COMMENT ON COLUMN models.public_name IS '对客户端暴露的模型名称';
COMMENT ON COLUMN models.provider_id IS '所属上游提供方 ID';
COMMENT ON COLUMN models.upstream_model IS '实际请求上游时使用的模型名称';
COMMENT ON COLUMN models.route_strategy IS '路由策略，例如 fixed、round_robin、failover';
COMMENT ON COLUMN models.is_enabled IS '该模型是否对外可用';
COMMENT ON COLUMN models.max_tokens IS '该模型默认或最大 Token 限制';
COMMENT ON COLUMN models.temperature IS '该模型默认温度参数';
COMMENT ON COLUMN models.timeout_seconds IS '请求该模型时的超时时间，单位秒';
COMMENT ON COLUMN models.metadata IS '模型扩展配置，使用 JSON 存储';
COMMENT ON COLUMN models.created_at IS '记录创建时间';
COMMENT ON COLUMN models.updated_at IS '记录最后更新时间';

COMMENT ON TABLE request_logs IS '网关请求日志与成本观测记录表';
COMMENT ON COLUMN request_logs.id IS '主键 ID';
COMMENT ON COLUMN request_logs.trace_id IS '用于串联日志链路的追踪 ID';
COMMENT ON COLUMN request_logs.request_type IS '请求类型，例如 chat_completions';
COMMENT ON COLUMN request_logs.model_public_name IS '客户端请求的对外模型名称';
COMMENT ON COLUMN request_logs.upstream_model IS '实际调用的上游模型名称';
COMMENT ON COLUMN request_logs.provider_id IS '本次请求命中的提供方 ID';
COMMENT ON COLUMN request_logs.provider_key_id IS '本次请求命中的 Key ID';
COMMENT ON COLUMN request_logs.admin_user_id IS '相关后台管理员 ID，如无则为空';
COMMENT ON COLUMN request_logs.client_ip IS '客户端请求 IP 地址';
COMMENT ON COLUMN request_logs.request_method IS '客户端请求使用的 HTTP 方法';
COMMENT ON COLUMN request_logs.request_path IS '网关请求路径';
COMMENT ON COLUMN request_logs.http_status IS '网关返回的 HTTP 状态码';
COMMENT ON COLUMN request_logs.success IS '请求是否成功完成';
COMMENT ON COLUMN request_logs.latency_ms IS '请求总耗时，单位毫秒';
COMMENT ON COLUMN request_logs.prompt_tokens IS '提示词 Token 数';
COMMENT ON COLUMN request_logs.completion_tokens IS '补全文本 Token 数';
COMMENT ON COLUMN request_logs.total_tokens IS '总 Token 数';
COMMENT ON COLUMN request_logs.estimated_cost IS '预估请求成本';
COMMENT ON COLUMN request_logs.error_type IS '归一化后的错误类型';
COMMENT ON COLUMN request_logs.error_message IS '详细错误信息';
COMMENT ON COLUMN request_logs.request_payload IS '请求载荷，使用 JSON 存储';
COMMENT ON COLUMN request_logs.response_payload IS '响应载荷，使用 JSON 存储';
COMMENT ON COLUMN request_logs.metadata IS '附加结构化信息，使用 JSON 存储';
COMMENT ON COLUMN request_logs.created_at IS '记录创建时间';
