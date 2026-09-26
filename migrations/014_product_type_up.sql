-- 014_product_type_up.sql
-- products.product_type memisahkan barang jual (MENU) dari bahan baku,
-- packaging, dan other. Default MENU supaya baris lama tetap barang jual.
-- Constraint diberi nama eksplisit supaya mapping error di repository stabil.

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS product_type VARCHAR(20) NOT NULL DEFAULT 'MENU';

ALTER TABLE products DROP CONSTRAINT IF EXISTS valid_product_type;
ALTER TABLE products
    ADD CONSTRAINT valid_product_type
    CHECK (product_type IN ('MENU', 'RAW_MATERIAL', 'PACKAGING', 'OTHER'));

CREATE INDEX IF NOT EXISTS idx_products_store_type ON products(store_id, product_type);
