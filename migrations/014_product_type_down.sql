-- 014_product_type_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 014 up.

DROP INDEX IF EXISTS idx_products_store_type;

ALTER TABLE products DROP CONSTRAINT IF EXISTS valid_product_type;

ALTER TABLE products DROP COLUMN IF EXISTS product_type;
