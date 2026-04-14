ALTER TABLE client_api_keys
DROP CONSTRAINT IF EXISTS chk_client_api_keys_daily_token_limit,
DROP CONSTRAINT IF EXISTS chk_client_api_keys_daily_request_limit,
DROP CONSTRAINT IF EXISTS chk_client_api_keys_rpm_limit,
DROP COLUMN IF EXISTS daily_token_limit,
DROP COLUMN IF EXISTS daily_request_limit,
DROP COLUMN IF EXISTS rpm_limit;
