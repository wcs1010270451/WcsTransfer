WITH ledger AS (
    SELECT
        tenant_id,
        COALESCE(SUM(CASE WHEN direction = 'credit' THEN amount ELSE 0 END), 0)::numeric(18, 4) AS ledger_credit_amount,
        COALESCE(SUM(CASE WHEN direction = 'debit' THEN amount ELSE 0 END), 0)::numeric(18, 4) AS ledger_debit_amount
    FROM tenant_wallet_ledger
    GROUP BY tenant_id
),
logs AS (
    SELECT
        cak.tenant_id,
        COALESCE(SUM(rl.billable_amount), 0)::numeric(18, 4) AS log_billable_amount,
        COALESCE(SUM(rl.cost_amount), 0)::numeric(18, 4) AS log_cost_amount
    FROM request_logs rl
    JOIN client_api_keys cak ON cak.id = rl.client_api_key_id
    GROUP BY cak.tenant_id
)
SELECT
    t.id AS tenant_id,
    t.name AS tenant_name,
    t.wallet_balance::numeric(18, 4) AS wallet_balance,
    COALESCE(l.ledger_credit_amount, 0) AS ledger_credit_amount,
    COALESCE(l.ledger_debit_amount, 0) AS ledger_debit_amount,
    (COALESCE(l.ledger_credit_amount, 0) - COALESCE(l.ledger_debit_amount, 0))::numeric(18, 4) AS ledger_net_amount,
    COALESCE(g.log_billable_amount, 0) AS log_billable_amount,
    COALESCE(g.log_cost_amount, 0) AS log_cost_amount,
    (t.wallet_balance - (COALESCE(l.ledger_credit_amount, 0) - COALESCE(l.ledger_debit_amount, 0)))::numeric(18, 4) AS wallet_vs_ledger_diff,
    (COALESCE(l.ledger_debit_amount, 0) - COALESCE(g.log_billable_amount, 0))::numeric(18, 4) AS ledger_vs_logs_diff
FROM tenants t
LEFT JOIN ledger l ON l.tenant_id = t.id
LEFT JOIN logs g ON g.tenant_id = t.id
ORDER BY t.id DESC;
