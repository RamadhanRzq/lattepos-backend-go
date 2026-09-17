-- 005_create_stores_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 005 up.

DROP INDEX IF EXISTS idx_user_stores_store_id;
DROP INDEX IF EXISTS idx_user_stores_user_id;
DROP INDEX IF EXISTS idx_stores_org_id;

DROP TABLE IF EXISTS user_stores;
DROP TABLE IF EXISTS stores;
