-- 015_create_recipes_up.sql
-- Recipe = komposisi bahan baku yang dikonsumsi saat satu product terjual.
-- Satu product boleh punya beberapa versi; hanya satu yang aktif (partial unique index).
-- Constraint diberi nama eksplisit supaya mapping error di repository stabil.

CREATE TABLE IF NOT EXISTS recipes (
    id             UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID        NOT NULL,
    store_id       UUID         NOT NULL,
    product_id     UUID         NOT NULL,
    name           VARCHAR(200) NOT NULL,
    version        INTEGER      NOT NULL DEFAULT 1 CHECK (version > 0),
    yield_quantity NUMERIC(12,3) NOT NULL DEFAULT 1 CHECK (yield_quantity > 0),
    is_active      BOOLEAN      NOT NULL DEFAULT FALSE,
    notes          TEXT         NOT NULL DEFAULT '',
    created_by     UUID,
    created_at     TIMESTAMP    DEFAULT NOW(),
    updated_at     TIMESTAMP    DEFAULT NOW(),
    CONSTRAINT fk_recipes_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_recipes_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_recipes_product FOREIGN KEY (product_id)
        REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT fk_recipes_creator FOREIGN KEY (created_by)
        REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT unique_recipe_version_per_product UNIQUE (product_id, version)
);

-- Hanya satu recipe aktif per product; sisanya bebas.
CREATE UNIQUE INDEX IF NOT EXISTS unique_active_recipe_per_product
    ON recipes(product_id) WHERE is_active;

CREATE INDEX IF NOT EXISTS idx_recipes_store_id   ON recipes(store_id);
CREATE INDEX IF NOT EXISTS idx_recipes_org_id     ON recipes(organization_id);
CREATE INDEX IF NOT EXISTS idx_recipes_product_id ON recipes(product_id);

CREATE TABLE IF NOT EXISTS recipe_items (
    id                   UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id            UUID          NOT NULL,
    organization_id      UUID          NOT NULL,
    store_id             UUID          NOT NULL,
    ingredient_product_id UUID         NOT NULL,
    quantity             NUMERIC(12,3) NOT NULL CHECK (quantity > 0),
    unit                 VARCHAR(20)   NOT NULL DEFAULT 'pcs',
    wastage_percentage   NUMERIC(5,2)  NOT NULL DEFAULT 0
        CHECK (wastage_percentage >= 0 AND wastage_percentage <= 100),
    sequence             INTEGER       NOT NULL DEFAULT 0,
    notes                TEXT          NOT NULL DEFAULT '',
    created_at           TIMESTAMP     DEFAULT NOW(),
    CONSTRAINT fk_recipe_items_recipe FOREIGN KEY (recipe_id)
        REFERENCES recipes(id) ON DELETE CASCADE,
    CONSTRAINT fk_recipe_items_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_recipe_items_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_recipe_items_ingredient FOREIGN KEY (ingredient_product_id)
        REFERENCES products(id) ON DELETE RESTRICT,
    CONSTRAINT unique_recipe_item_per_ingredient UNIQUE (recipe_id, ingredient_product_id)
);

CREATE INDEX IF NOT EXISTS idx_recipe_items_recipe_id     ON recipe_items(recipe_id);
CREATE INDEX IF NOT EXISTS idx_recipe_items_ingredient_id ON recipe_items(ingredient_product_id);
