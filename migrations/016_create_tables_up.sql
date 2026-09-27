-- 016_create_tables_up.sql
-- Table (meja) = unit layanan dine-in dalam satu store.
-- Constraint diberi nama eksplisit supaya mapping error di repository stabil.

CREATE TABLE IF NOT EXISTS tables (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID          NOT NULL,
    store_id        UUID          NOT NULL,
    name            VARCHAR(100)  NOT NULL,
    area            VARCHAR(100)  NOT NULL DEFAULT '',
    capacity        INTEGER       NOT NULL DEFAULT 0,
    status          VARCHAR(20)   NOT NULL DEFAULT 'available',
    is_active       BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP     DEFAULT NOW(),
    updated_at      TIMESTAMP     DEFAULT NOW(),
    CONSTRAINT fk_tables_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_tables_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT unique_table_name_per_store UNIQUE (store_id, name),
    CONSTRAINT chk_tables_status CHECK (status IN ('available', 'occupied', 'reserved')),
    CONSTRAINT chk_tables_capacity CHECK (capacity >= 0)
);

CREATE INDEX IF NOT EXISTS idx_tables_store_id ON tables(store_id);
CREATE INDEX IF NOT EXISTS idx_tables_org_id   ON tables(organization_id);
