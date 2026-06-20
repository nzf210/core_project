-- F034: Rollback — remove extended columns from addon_prices
ALTER TABLE addon_prices
    DROP COLUMN IF EXISTS unit,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS updated_at;
