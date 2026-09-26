-- 015_create_recipes_down.sql
-- Rollback non-destructive: hanya objek yang dibuat di 015 up.

DROP INDEX IF EXISTS idx_recipe_items_ingredient_id;
DROP INDEX IF EXISTS idx_recipe_items_recipe_id;
DROP TABLE IF EXISTS recipe_items;

DROP INDEX IF EXISTS idx_recipes_product_id;
DROP INDEX IF EXISTS idx_recipes_org_id;
DROP INDEX IF EXISTS idx_recipes_store_id;
DROP INDEX IF EXISTS unique_active_recipe_per_product;

DROP TABLE IF EXISTS recipes;
