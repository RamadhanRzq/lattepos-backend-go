DROP INDEX IF EXISTS idx_user_roles_org_id;
DROP INDEX IF EXISTS idx_roles_org_id;
DROP INDEX IF EXISTS idx_org_members_user_id;
DROP INDEX IF EXISTS idx_org_members_org_id;
DROP INDEX IF EXISTS idx_organizations_slug;

ALTER TABLE user_roles DROP COLUMN IF EXISTS org_id;
ALTER TABLE roles DROP COLUMN IF EXISTS org_id;

DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;