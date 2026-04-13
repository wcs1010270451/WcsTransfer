-- 开发环境初始化数据
-- 使用前请先把 provider_keys 里的 api_key 改成你自己的真实 Key

INSERT INTO providers (
    name,
    slug,
    provider_type,
    base_url,
    status,
    description,
    extra_config
) VALUES (
    'OpenAI Official',
    'openai-official',
    'openai',
    'https://api.openai.com/v1',
    'active',
    '开发环境默认的 OpenAI 官方提供方配置',
    '{}'::jsonb
)
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    provider_type = EXCLUDED.provider_type,
    base_url = EXCLUDED.base_url,
    status = EXCLUDED.status,
    description = EXCLUDED.description,
    extra_config = EXCLUDED.extra_config,
    updated_at = NOW();

WITH provider_row AS (
    SELECT id
    FROM providers
    WHERE slug = 'openai-official'
)
INSERT INTO provider_keys (
    provider_id,
    name,
    api_key,
    status,
    weight,
    priority,
    rpm_limit,
    tpm_limit
)
SELECT
    provider_row.id,
    'primary',
    'sk-replace-with-your-real-key',
    'active',
    100,
    100,
    0,
    0
FROM provider_row
ON CONFLICT (provider_id, name) DO UPDATE SET
    api_key = EXCLUDED.api_key,
    status = EXCLUDED.status,
    weight = EXCLUDED.weight,
    priority = EXCLUDED.priority,
    rpm_limit = EXCLUDED.rpm_limit,
    tpm_limit = EXCLUDED.tpm_limit,
    updated_at = NOW();

WITH provider_row AS (
    SELECT id
    FROM providers
    WHERE slug = 'openai-official'
)
INSERT INTO models (
    public_name,
    provider_id,
    upstream_model,
    route_strategy,
    is_enabled,
    max_tokens,
    temperature,
    timeout_seconds,
    metadata
)
SELECT
    'gpt-4o-mini',
    provider_row.id,
    'gpt-4o-mini',
    'fixed',
    TRUE,
    0,
    0.7,
    120,
    '{}'::jsonb
FROM provider_row
ON CONFLICT (public_name) DO UPDATE SET
    provider_id = EXCLUDED.provider_id,
    upstream_model = EXCLUDED.upstream_model,
    route_strategy = EXCLUDED.route_strategy,
    is_enabled = EXCLUDED.is_enabled,
    max_tokens = EXCLUDED.max_tokens,
    temperature = EXCLUDED.temperature,
    timeout_seconds = EXCLUDED.timeout_seconds,
    metadata = EXCLUDED.metadata,
    updated_at = NOW();
