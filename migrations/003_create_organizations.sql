-- 003_create_organizations.up.sql
-- Subsystem multi-tenant organizations & scoping
-- 
DROP INDEX IF EXISTS idx_user_roles_org_id;
DROP INDEX IF EXISTS idx_roles_org_id;
DROP INDEX IF EXISTS idx_org_members_user_id;
DROP INDEX IF EXISTS idx_org_members_org_id;
DROP INDEX IF EXISTS idx_organizations_slug;

ALTER TABLE user_roles DROP COLUMN IF EXISTS org_id;
ALTER TABLE roles DROP COLUMN IF EXISTS org_id;

DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;

CREATE TABLE IF NOT EXISTS organizations (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP    DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS organization_members (
    id        UUID      PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id    UUID      NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id   UUID      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_org_user UNIQUE (org_id, user_id)
);

ALTER TABLE roles ADD COLUMN IF NOT EXISTS org_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS org_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);
CREATE INDEX IF NOT EXISTS idx_org_members_org_id ON organization_members(org_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON organization_members(user_id);
CREATE INDEX IF NOT EXISTS idx_roles_org_id ON roles(org_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_org_id ON user_roles(org_id);
