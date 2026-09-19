-- 012_sales_variant_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 012 up.

DROP INDEX IF EXISTS idx_sale_items_variant_id;
ALTER TABLE sale_items DROP CONSTRAINT IF EXISTS fk_sale_items_variant;
ALTER TABLE sale_items DROP COLUMN IF EXISTS variant_id;
