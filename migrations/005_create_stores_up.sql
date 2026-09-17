-- 005_create_stores_up.sql
-- Store = outlet fisik dalam satu organization; user_stores = akses user ke outlet.
-- FK CASCADE konsisten migration existing (organization_members, user_roles):
-- hapus organization/store/user hanya menghapus baris pivot, tidak merambat ke entity lain.
-- Constraint diberi nama eksplisit supaya mapping error di repository stabil.

CREATE TABLE IF NOT EXISTS stores (
    id              UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID         NOT NULL,
    name            VARCHAR(100) NOT NULL,
    code            VARCHAR(50)  NOT NULL,
    address         TEXT,
    phone           VARCHAR(30),
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP    DEFAULT NOW(),
    updated_at      TIMESTAMP    DEFAULT NOW(),
    CONSTRAINT fk_stores_org FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT unique_store_code_per_org UNIQUE (organization_id, code)
);

CREATE TABLE IF NOT EXISTS user_stores (
    user_id    UUID      NOT NULL,
    store_id   UUID      NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_user_stores_user FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_stores_store FOREIGN KEY (store_id)
        REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT unique_user_store UNIQUE (user_id, store_id)
);

CREATE INDEX IF NOT EXISTS idx_stores_org_id         ON stores(organization_id);
CREATE INDEX IF NOT EXISTS idx_user_stores_user_id   ON user_stores(user_id);
CREATE INDEX IF NOT EXISTS idx_user_stores_store_id  ON user_stores(store_id);
