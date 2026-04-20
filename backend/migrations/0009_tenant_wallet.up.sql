ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS wallet_balance NUMERIC(14, 4) NOT NULL DEFAULT 0;

ALTER TABLE tenants
    ADD CONSTRAINT chk_tenants_wallet_balance_non_negative
        CHECK (wallet_balance >= 0);

CREATE TABLE IF NOT EXISTS tenant_wallet_ledger (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    direction VARCHAR(16) NOT NULL CHECK (direction IN ('credit', 'debit')),
    amount NUMERIC(14, 4) NOT NULL CHECK (amount > 0),
    balance_before NUMERIC(14, 4) NOT NULL,
    balance_after NUMERIC(14, 4) NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    operator_type VARCHAR(32) NOT NULL DEFAULT 'system',
    operator_user_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_wallet_ledger_tenant_created_at
    ON tenant_wallet_ledger (tenant_id, created_at DESC);

COMMENT ON COLUMN tenants.wallet_balance IS '租户钱包余额（美元）';
COMMENT ON TABLE tenant_wallet_ledger IS '租户钱包流水';
COMMENT ON COLUMN tenant_wallet_ledger.direction IS '流水方向：credit 充值，debit 扣费';
COMMENT ON COLUMN tenant_wallet_ledger.amount IS '本次变动金额（美元）';
COMMENT ON COLUMN tenant_wallet_ledger.balance_before IS '变动前余额（美元）';
COMMENT ON COLUMN tenant_wallet_ledger.balance_after IS '变动后余额（美元）';
COMMENT ON COLUMN tenant_wallet_ledger.note IS '充值或扣费说明';
COMMENT ON COLUMN tenant_wallet_ledger.operator_type IS '操作来源：admin 或 system';
COMMENT ON COLUMN tenant_wallet_ledger.operator_user_id IS '操作人 ID，当前为空保留';
