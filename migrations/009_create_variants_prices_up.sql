-- 009_create_variants_prices_up.sql
-- ProductVariant = varian barang dalam satu product+store; ProductPrice = harga per product/variant.
-- Constraint diberi nama eksplisit supaya mapping error di repository stabil.

CREATE TABLE IF NOT EXISTS product_variants (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID          NOT NULL,
    store_id        UUID          NOT NULL,
    product_id      UUID          NOT NULL,
    name            VARCHAR(200)  NOT NULL,
    sku             VARCHAR(100),
    stock           INTEGER       NOT NULL DEFAULT 0 CHECK (stock >= 0),
    is_active       BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP     DEFAULT NOW(),
    updated_at      TIMESTAMP     DEFAULT NOW(),
    CONSTRAINT fk_variants_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_variants_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_variants_product FOREIGN KEY (product_id)
        REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT unique_variant_name UNIQUE (store_id, product_id, name)
);

CREATE UNIQUE INDEX IF NOT EXISTS unique_variant_sku_per_store
    ON product_variants(store_id, sku)
    WHERE sku IS NOT NULL AND sku <> '';

CREATE INDEX IF NOT EXISTS idx_variants_store_id ON product_variants(store_id);
CREATE INDEX IF NOT EXISTS idx_variants_product_id ON product_variants(product_id);
CREATE INDEX IF NOT EXISTS idx_variants_org_id ON product_variants(organization_id);

CREATE TABLE IF NOT EXISTS product_prices (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID          NOT NULL,
    store_id        UUID          NOT NULL,
    product_id      UUID          NOT NULL,
    variant_id      UUID,
    price_type      VARCHAR(30)   NOT NULL DEFAULT 'retail',
    price           NUMERIC(14,2) NOT NULL CHECK (price >= 0),
    min_quantity    INTEGER       NOT NULL DEFAULT 1 CHECK (min_quantity >= 1),
    is_active       BOOLEAN       NOT NULL DEFAULT TRUE,
    valid_from      TIMESTAMP,
    valid_until     TIMESTAMP,
    created_at      TIMESTAMP     DEFAULT NOW(),
    updated_at      TIMESTAMP     DEFAULT NOW(),
    CONSTRAINT fk_prices_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_prices_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_prices_product FOREIGN KEY (product_id)
        REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT fk_prices_variant FOREIGN KEY (variant_id)
        REFERENCES product_variants(id) ON DELETE CASCADE,
    CONSTRAINT valid_price_range CHECK (valid_until IS NULL OR valid_from IS NULL OR valid_until >= valid_from)
);

CREATE INDEX IF NOT EXISTS idx_prices_product_id ON product_prices(product_id);
CREATE INDEX IF NOT EXISTS idx_prices_variant_id ON product_prices(variant_id);
CREATE INDEX IF NOT EXISTS idx_prices_type ON product_prices(price_type);
CREATE INDEX IF NOT EXISTS idx_prices_store_id ON product_prices(store_id);
CREATE INDEX IF NOT EXISTS idx_prices_org_id ON product_prices(organization_id);
