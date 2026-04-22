CREATE TABLE IF NOT EXISTS admin_action_logs (
    id BIGSERIAL PRIMARY KEY,
    admin_user_id BIGINT NULL REFERENCES admin_users(id) ON DELETE SET NULL,
    admin_username VARCHAR(128) NOT NULL DEFAULT '',
    admin_display_name VARCHAR(128) NOT NULL DEFAULT '',
    auth_mode VARCHAR(32) NOT NULL DEFAULT '',
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id BIGINT NULL,
    resource_name VARCHAR(255) NOT NULL DEFAULT '',
    request_method VARCHAR(16) NOT NULL DEFAULT '',
    request_path VARCHAR(255) NOT NULL DEFAULT '',
    client_ip INET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_action_logs_created_at
    ON admin_action_logs (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_admin_action_logs_admin_user_id_created_at
    ON admin_action_logs (admin_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_admin_action_logs_action_created_at
    ON admin_action_logs (action, created_at DESC);

COMMENT ON TABLE admin_action_logs IS '管理端操作审计日志';
COMMENT ON COLUMN admin_action_logs.admin_user_id IS '操作者管理员 ID，legacy token 模式下可为空';
COMMENT ON COLUMN admin_action_logs.admin_username IS '操作者管理员登录名快照';
COMMENT ON COLUMN admin_action_logs.admin_display_name IS '操作者管理员显示名快照';
COMMENT ON COLUMN admin_action_logs.auth_mode IS '鉴权模式，例如 session 或 legacy_token';
COMMENT ON COLUMN admin_action_logs.action IS '操作动作，例如 tenant.update、wallet.credit';
COMMENT ON COLUMN admin_action_logs.resource_type IS '资源类型，例如 tenant、provider、model';
COMMENT ON COLUMN admin_action_logs.resource_id IS '资源主键 ID';
COMMENT ON COLUMN admin_action_logs.resource_name IS '资源名称快照';
COMMENT ON COLUMN admin_action_logs.request_method IS '发起操作的 HTTP 方法';
COMMENT ON COLUMN admin_action_logs.request_path IS '发起操作的请求路径';
COMMENT ON COLUMN admin_action_logs.client_ip IS '操作者来源 IP';
COMMENT ON COLUMN admin_action_logs.metadata IS '结构化审计附加信息';
COMMENT ON COLUMN admin_action_logs.created_at IS '操作记录创建时间';
