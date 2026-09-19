-- 010_create_stock_movements_up.sql
-- StockMovement = riwayat mutasi stok milik satu store; stock snapshot before/after.
-- Stok live tetap di products.stock / product_variants.stock; movement hanya riwayat.
-- Constraint diberi nama eksplisit supaya mapping error di repository stabil.

CREATE TABLE IF NOT EXISTS stock_movements (
    id              UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID         NOT NULL,
    store_id        UUID         NOT NULL,
    product_id      UUID         NOT NULL,
    variant_id      UUID,
    type            VARCHAR(20)  NOT NULL,
    quantity        INTEGER      NOT NULL CHECK (quantity > 0),
    stock_before    INTEGER      NOT NULL,
    stock_after     INTEGER      NOT NULL,
    reference_type  VARCHAR(30),
    reference_id    UUID,
    notes           TEXT         NOT NULL DEFAULT '',
    created_by      UUID,
    created_at      TIMESTAMP    DEFAULT NOW(),
    CONSTRAINT valid_movement_type CHECK (type IN ('in', 'out', 'adjustment', 'return')),
    CONSTRAINT fk_stock_movements_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_stock_movements_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_stock_movements_product FOREIGN KEY (product_id)
        REFERENCES products(id) ON DELETE RESTRICT,
    CONSTRAINT fk_stock_movements_variant FOREIGN KEY (variant_id)
        REFERENCES product_variants(id) ON DELETE RESTRICT,
    CONSTRAINT fk_stock_movements_creator FOREIGN KEY (created_by)
        REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_store_id    ON stock_movements(store_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_product_id  ON stock_movements(product_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_variant_id  ON stock_movements(variant_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_reference   ON stock_movements(reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_created_at  ON stock_movements(created_at);
