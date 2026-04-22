ALTER TABLE tenant_wallet_ledger
    DROP COLUMN IF EXISTS reserved_amount;

ALTER TABLE request_logs
    DROP COLUMN IF EXISTS reserved_amount;
