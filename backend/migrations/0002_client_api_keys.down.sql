DROP INDEX IF EXISTS idx_request_logs_client_api_key_id_created_at;
ALTER TABLE request_logs DROP COLUMN IF EXISTS client_api_key_id;

DROP TRIGGER IF EXISTS trg_client_api_keys_updated_at ON client_api_keys;
DROP INDEX IF EXISTS idx_client_api_keys_expires_at;
DROP INDEX IF EXISTS idx_client_api_keys_status;
DROP TABLE IF EXISTS client_api_keys;
