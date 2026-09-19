-- 007_create_sales_up.sql
-- Sale = transaksi milik satu store; sale_items = snapshot harga saat transaksi.
-- FK CASCADE konsisten migration existing: hapus store/user hanya menghapus
-- baris terkait, tidak merambat ke entity lain. Product dipakai RESTRICT agar
-- riwayat transaksi tidak yatim; products memakai soft delete sehingga
-- penghapusan fisik tidak terjadi di jalur normal.
-- Constraint diberi nama eksplisit supaya mapping error di repository stabil.

CREATE TABLE IF NOT EXISTS sales (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id        UUID          NOT NULL,
    organization_id UUID          NOT NULL,
    user_id         UUID          NOT NULL,
    total_amount    NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    discount_amount NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    tax_amount      NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    grand_total     NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (grand_total >= 0),
    payment_method  VARCHAR(30)   NOT NULL,
    status          VARCHAR(20)   NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'completed', 'cancelled')),
    notes           TEXT,
    created_at      TIMESTAMP     DEFAULT NOW(),
    updated_at      TIMESTAMP     DEFAULT NOW(),
    CONSTRAINT fk_sales_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_sales_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_sales_user FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sale_items (
    id          UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    sale_id     UUID          NOT NULL,
    product_id  UUID          NOT NULL,
    quantity    INTEGER       NOT NULL CHECK (quantity > 0),
    unit_price  NUMERIC(14,2) NOT NULL CHECK (unit_price >= 0),
    subtotal    NUMERIC(14,2) NOT NULL CHECK (subtotal >= 0),
    created_at  TIMESTAMP     DEFAULT NOW(),
    CONSTRAINT fk_sale_items_sale FOREIGN KEY (sale_id)
        REFERENCES sales(id) ON DELETE CASCADE,
    CONSTRAINT fk_sale_items_product FOREIGN KEY (product_id)
        REFERENCES products(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_sales_store_id   ON sales(store_id);
CREATE INDEX IF NOT EXISTS idx_sales_org_id     ON sales(organization_id);
CREATE INDEX IF NOT EXISTS idx_sales_status     ON sales(status);
CREATE INDEX IF NOT EXISTS idx_sales_created_at ON sales(created_at);
CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id    ON sale_items(sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_items_product_id ON sale_items(product_id);
