run:
	go run ./cmd/api

build:
	go build -o bin/lattepos ./cmd/api

test:
	go test ./...

include .env
export

migrate:
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/001_create_users.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/002_create_rbac.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/003_create_organizations.up.sql

migrate-fresh:
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/001_create_users.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/002_create_rbac.sql
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/003_create_organizations.sql

seed:
	psql -U $(DB_USER) -h $(DB_HOST) -d $(DB_NAME) -f migrations/seed.sql
