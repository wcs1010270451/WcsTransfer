CREATE TABLE IF NOT EXISTS client_api_key_models (
    client_api_key_id BIGINT NOT NULL REFERENCES client_api_keys(id) ON DELETE CASCADE,
    model_id BIGINT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_api_key_id, model_id)
);

CREATE INDEX IF NOT EXISTS idx_client_api_key_models_model_id
    ON client_api_key_models(model_id);

COMMENT ON TABLE client_api_key_models IS '客户端 API Key 与可访问模型的关联表';
COMMENT ON COLUMN client_api_key_models.client_api_key_id IS '客户端 API Key ID';
COMMENT ON COLUMN client_api_key_models.model_id IS '允许访问的模型 ID';
COMMENT ON COLUMN client_api_key_models.created_at IS '创建时间';
