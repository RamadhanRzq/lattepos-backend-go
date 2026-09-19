-- 008_create_categories_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 008 up.

ALTER TABLE products DROP CONSTRAINT IF EXISTS fk_products_category;

DROP INDEX IF EXISTS idx_categories_parent_id;
DROP INDEX IF EXISTS idx_categories_org_id;
DROP INDEX IF EXISTS idx_categories_store_id;

DROP TABLE IF EXISTS categories;
