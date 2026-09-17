-- 004_scope_roles_to_org_up.sql
-- Role kini benar-benar ter-scope organisasi: nama role unik per organisasi,
-- bukan unik secara global. Role global (org_id NULL) tetap unik di antara
-- sesama role global.

ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_name_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_org_name
    ON roles (org_id, name) WHERE org_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_global_name
    ON roles (name) WHERE org_id IS NULL;

-- Resolusi permission selalu menyaring per (user, organisasi).
CREATE INDEX IF NOT EXISTS idx_user_roles_user_org ON user_roles (user_id, org_id);
