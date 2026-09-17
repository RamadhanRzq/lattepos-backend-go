-- 004_scope_roles_to_org_down.sql
DROP INDEX IF EXISTS idx_user_roles_user_org;
DROP INDEX IF EXISTS idx_roles_global_name;
DROP INDEX IF EXISTS idx_roles_org_name;

-- Gagal bila sudah ada nama role yang sama di beberapa organisasi.
ALTER TABLE roles ADD CONSTRAINT roles_name_key UNIQUE (name);
