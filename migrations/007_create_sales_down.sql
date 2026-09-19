-- 007_create_sales_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 007 up.

DROP INDEX IF EXISTS idx_sale_items_product_id;
DROP INDEX IF EXISTS idx_sale_items_sale_id;
DROP INDEX IF EXISTS idx_sales_created_at;
DROP INDEX IF EXISTS idx_sales_status;
DROP INDEX IF EXISTS idx_sales_org_id;
DROP INDEX IF EXISTS idx_sales_store_id;

DROP TABLE IF EXISTS sale_items;
DROP TABLE IF EXISTS sales;
