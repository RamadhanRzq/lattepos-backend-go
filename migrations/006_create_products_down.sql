-- 006_create_products_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 006 up.

DROP INDEX IF EXISTS idx_products_name;
DROP INDEX IF EXISTS idx_products_org_id;
DROP INDEX IF EXISTS idx_products_store_id;

DROP TABLE IF EXISTS products;
