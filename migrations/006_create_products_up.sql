-- 006_create_products_up.sql
-- Product = barang dagangan milik satu store dalam satu organization.
-- FK CASCADE konsisten migration existing: hapus organization/store hanya
-- menghapus product, tidak merambat ke entity lain.
-- Constraint diberi nama eksplisit supaya mapping error di repository stabil.

CREATE TABLE IF NOT EXISTS products (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id        UUID          NOT NULL,
    organization_id UUID          NOT NULL,
    name            VARCHAR(200)  NOT NULL,
    sku             VARCHAR(100)  NOT NULL,
    description     TEXT          NOT NULL DEFAULT '',
    price           BIGINT        NOT NULL DEFAULT 0 CHECK (price >= 0),
    stock           INTEGER       NOT NULL DEFAULT 0 CHECK (stock >= 0),
    unit            VARCHAR(20)   NOT NULL DEFAULT 'pcs',
    category_id     UUID,
    image_url       TEXT          NOT NULL DEFAULT '',
    is_active       BOOLEAN       NOT NULL DEFAULT TRUE,
    created_by      UUID,
    created_at      TIMESTAMP     DEFAULT NOW(),
    updated_at      TIMESTAMP     DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT fk_products_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_products_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT unique_product_sku_per_store UNIQUE (store_id, sku)
);

CREATE INDEX IF NOT EXISTS idx_products_store_id ON products(store_id);
CREATE INDEX IF NOT EXISTS idx_products_org_id   ON products(organization_id);
CREATE INDEX IF NOT EXISTS idx_products_name     ON products USING gin (to_tsvector('simple', name));
