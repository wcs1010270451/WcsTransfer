DROP INDEX IF EXISTS idx_tenant_wallet_ledger_request_log_id;

ALTER TABLE tenant_wallet_ledger
    DROP COLUMN IF EXISTS billable_amount,
    DROP COLUMN IF EXISTS cost_amount,
    DROP COLUMN IF EXISTS total_tokens,
    DROP COLUMN IF EXISTS model_public_name,
    DROP COLUMN IF EXISTS trace_id,
    DROP COLUMN IF EXISTS request_log_id;
