ALTER TABLE providers
    DROP CONSTRAINT IF EXISTS chk_providers_type;

ALTER TABLE providers
    ADD CONSTRAINT chk_providers_type
        CHECK (provider_type IN ('openai_compatible', 'openai', 'azure_openai', 'custom'));

COMMENT ON COLUMN providers.provider_type IS '提供方协议类型，例如 openai_compatible 或 azure_openai';
