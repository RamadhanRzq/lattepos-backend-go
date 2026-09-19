-- 011_create_kitchen_up.sql
-- Kitchen = antrian dapur per sale; satu baris kitchen_sales per sale.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS kitchen_sales (
    id              UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    sale_id         UUID         NOT NULL,
    organization_id UUID         NOT NULL,
    store_id        UUID         NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending'
        CONSTRAINT valid_kitchen_status CHECK (status IN ('pending', 'preparing', 'ready', 'served', 'cancelled')),
    priority        INTEGER      NOT NULL DEFAULT 0,
    notes           TEXT         NOT NULL DEFAULT '',
    started_at      TIMESTAMP,
    completed_at    TIMESTAMP,
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_kitchen_sale UNIQUE (sale_id),
    CONSTRAINT fk_kitchen_sales_sale FOREIGN KEY (sale_id)
        REFERENCES sales(id) ON DELETE CASCADE,
    CONSTRAINT fk_kitchen_sales_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_kitchen_sales_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS kitchen_sale_items (
    id              UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    kitchen_sale_id UUID         NOT NULL,
    sale_item_id    UUID         NOT NULL,
    product_id      UUID         NOT NULL,
    variant_id      UUID,
    quantity        INTEGER      NOT NULL CHECK (quantity > 0),
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending'
        CONSTRAINT valid_kitchen_item_status CHECK (status IN ('pending', 'preparing', 'ready', 'cancelled')),
    notes           TEXT         NOT NULL DEFAULT '',
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_kitchen_items_sale FOREIGN KEY (kitchen_sale_id)
        REFERENCES kitchen_sales(id) ON DELETE CASCADE,
    CONSTRAINT fk_kitchen_items_sale_item FOREIGN KEY (sale_item_id)
        REFERENCES sale_items(id) ON DELETE CASCADE,
    CONSTRAINT fk_kitchen_items_product FOREIGN KEY (product_id)
        REFERENCES products(id) ON DELETE RESTRICT,
    CONSTRAINT fk_kitchen_items_variant FOREIGN KEY (variant_id)
        REFERENCES product_variants(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_kitchen_sales_store_id ON kitchen_sales(store_id);
CREATE INDEX IF NOT EXISTS idx_kitchen_sales_org_id   ON kitchen_sales(organization_id);
CREATE INDEX IF NOT EXISTS idx_kitchen_sales_status   ON kitchen_sales(status);
CREATE INDEX IF NOT EXISTS idx_kitchen_sales_sale_id  ON kitchen_sales(sale_id);
CREATE INDEX IF NOT EXISTS idx_kitchen_items_kitchen_sale_id ON kitchen_sale_items(kitchen_sale_id);
