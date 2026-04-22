ALTER TABLE tenant_wallet_ledger
    ADD COLUMN IF NOT EXISTS request_log_id BIGINT REFERENCES request_logs(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS trace_id VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS model_public_name VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS total_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cost_amount NUMERIC(18, 8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS billable_amount NUMERIC(18, 8) NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_tenant_wallet_ledger_request_log_id
    ON tenant_wallet_ledger (request_log_id);

COMMENT ON COLUMN tenant_wallet_ledger.request_log_id IS '关联的请求日志 ID';
COMMENT ON COLUMN tenant_wallet_ledger.trace_id IS '关联请求的 trace_id';
COMMENT ON COLUMN tenant_wallet_ledger.model_public_name IS '关联请求的公共模型名';
COMMENT ON COLUMN tenant_wallet_ledger.total_tokens IS '关联请求的总 token 数';
COMMENT ON COLUMN tenant_wallet_ledger.cost_amount IS '关联请求的上游成本金额（美元）';
COMMENT ON COLUMN tenant_wallet_ledger.billable_amount IS '关联请求的对客计费金额（美元）';
