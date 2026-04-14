ALTER TABLE client_api_keys
ADD COLUMN rpm_limit INTEGER NOT NULL DEFAULT 0,
ADD COLUMN daily_request_limit INTEGER NOT NULL DEFAULT 0,
ADD COLUMN daily_token_limit INTEGER NOT NULL DEFAULT 0;

ALTER TABLE client_api_keys
ADD CONSTRAINT chk_client_api_keys_rpm_limit CHECK (rpm_limit >= 0),
ADD CONSTRAINT chk_client_api_keys_daily_request_limit CHECK (daily_request_limit >= 0),
ADD CONSTRAINT chk_client_api_keys_daily_token_limit CHECK (daily_token_limit >= 0);

COMMENT ON COLUMN client_api_keys.rpm_limit IS '客户端每分钟请求数限制，0 表示不限制';
COMMENT ON COLUMN client_api_keys.daily_request_limit IS '客户端每日请求数限制，0 表示不限制';
COMMENT ON COLUMN client_api_keys.daily_token_limit IS '客户端每日 Token 总量限制，0 表示不限制';
