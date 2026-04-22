ALTER TABLE request_logs
    ADD COLUMN IF NOT EXISTS reserved_amount NUMERIC(18, 8) NOT NULL DEFAULT 0;

ALTER TABLE tenant_wallet_ledger
    ADD COLUMN IF NOT EXISTS reserved_amount NUMERIC(18, 8) NOT NULL DEFAULT 0;

COMMENT ON COLUMN request_logs.reserved_amount IS '请求执行前估算的最低预留金额（美元）';
COMMENT ON COLUMN tenant_wallet_ledger.reserved_amount IS '与本次扣费关联的预留金额（美元）';
