-- 016_create_tables_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 016 up.

DROP INDEX IF EXISTS idx_tables_org_id;
DROP INDEX IF EXISTS idx_tables_store_id;

DROP TABLE IF EXISTS tables;
