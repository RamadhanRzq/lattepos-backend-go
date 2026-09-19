-- 012_sales_variant_up.sql
-- sale_items mendapat variant_id opsional: transaksi bisa menyasar varian
-- tertentu. NULL = product polos. FK RESTRICT konsisten 007 (product juga
-- RESTRICT); variant dihapus → transaksi lama tetap utuh sebagai snapshot.
-- Constraint diberi nama eksplisit supaya mapping error di repository stabil.

ALTER TABLE sale_items
    ADD COLUMN variant_id UUID,
    ADD CONSTRAINT fk_sale_items_variant FOREIGN KEY (variant_id)
        REFERENCES product_variants(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_sale_items_variant_id ON sale_items(variant_id);
