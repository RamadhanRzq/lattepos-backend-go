-- 010_create_stock_movements_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 010 up.

DROP INDEX IF EXISTS idx_stock_movements_created_at;
DROP INDEX IF EXISTS idx_stock_movements_reference;
DROP INDEX IF EXISTS idx_stock_movements_variant_id;
DROP INDEX IF EXISTS idx_stock_movements_product_id;
DROP INDEX IF EXISTS idx_stock_movements_store_id;

DROP TABLE IF EXISTS stock_movements;
