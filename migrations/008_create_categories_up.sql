-- 008_create_categories_up.sql
-- Category = pengelompokan product dalam satu store.
-- Constraint diberi nama eksplisit supaya mapping error di repository stabil.

CREATE TABLE IF NOT EXISTS categories (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID          NOT NULL,
    store_id        UUID          NOT NULL,
    name            VARCHAR(200)  NOT NULL,
    slug            VARCHAR(200)  NOT NULL,
    description     TEXT          NOT NULL DEFAULT '',
    parent_id       UUID,
    created_at      TIMESTAMP     DEFAULT NOW(),
    updated_at      TIMESTAMP     DEFAULT NOW(),
    CONSTRAINT fk_categories_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_categories_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id)
        REFERENCES categories(id) ON DELETE SET NULL,
    CONSTRAINT unique_category_slug_per_store UNIQUE (store_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_categories_store_id ON categories(store_id);
CREATE INDEX IF NOT EXISTS idx_categories_org_id   ON categories(organization_id);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);

ALTER TABLE products
    ADD CONSTRAINT fk_products_category FOREIGN KEY (category_id)
        REFERENCES categories(id) ON DELETE SET NULL;
