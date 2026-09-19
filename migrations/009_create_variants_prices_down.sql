-- 009_create_variants_prices_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 009 up.

DROP INDEX IF EXISTS idx_prices_org_id;
DROP INDEX IF EXISTS idx_prices_store_id;
DROP INDEX IF EXISTS idx_prices_type;
DROP INDEX IF EXISTS idx_prices_variant_id;
DROP INDEX IF EXISTS idx_prices_product_id;

DROP TABLE IF EXISTS product_prices;

DROP INDEX IF EXISTS idx_variants_org_id;
DROP INDEX IF EXISTS idx_variants_product_id;
DROP INDEX IF EXISTS idx_variants_store_id;
DROP INDEX IF EXISTS unique_variant_sku_per_store;

DROP TABLE IF EXISTS product_variants;
