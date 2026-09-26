run:
	go run ./cmd/server

build:
	go build -o bin/lattepos ./cmd/server

test:
	go test ./...

include .env
export

migrate:
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/001_create_users.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/002_create_rbac.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/003_create_organizations_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/004_scope_roles_to_org_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/005_create_stores_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/006_create_products_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/007_create_sales_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/008_create_categories_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/009_create_variants_prices_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/010_create_stock_movements_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/011_create_kitchen_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/012_sales_variant_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/013_create_refresh_tokens_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/014_product_type_up.sql
migrate-fresh:
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/001_create_users.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/002_create_rbac.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/003_create_organizations_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/004_scope_roles_to_org_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/005_create_stores_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/006_create_products_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/007_create_sales_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/008_create_categories_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/009_create_variants_prices_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/010_create_stock_movements_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/011_create_kitchen_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/012_sales_variant_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/013_create_refresh_tokens_up.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/014_product_type_up.sql

seed:
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/seed.sql
