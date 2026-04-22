ALTER TABLE models
    DROP CONSTRAINT IF EXISTS chk_models_reserve_min_amount_non_negative;

ALTER TABLE models
    DROP CONSTRAINT IF EXISTS chk_models_reserve_multiplier_positive;

ALTER TABLE models
    DROP COLUMN IF EXISTS reserve_min_amount,
    DROP COLUMN IF EXISTS reserve_multiplier;
