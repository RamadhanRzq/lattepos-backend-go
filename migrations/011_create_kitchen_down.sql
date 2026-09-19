-- 011_create_kitchen_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 011 up.

DROP INDEX IF EXISTS idx_kitchen_items_kitchen_sale_id;
DROP INDEX IF EXISTS idx_kitchen_sales_sale_id;
DROP INDEX IF EXISTS idx_kitchen_sales_status;
DROP INDEX IF EXISTS idx_kitchen_sales_org_id;
DROP INDEX IF EXISTS idx_kitchen_sales_store_id;

DROP TABLE IF EXISTS kitchen_sale_items;
DROP TABLE IF EXISTS kitchen_sales;
