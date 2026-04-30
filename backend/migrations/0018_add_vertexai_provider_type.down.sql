ALTER TABLE providers
    DROP CONSTRAINT IF EXISTS chk_providers_type;

ALTER TABLE providers
    ADD CONSTRAINT chk_providers_type
        CHECK (provider_type IN ('openai_compatible', 'openai', 'azure_openai', 'custom', 'anthropic', 'gemini'));
