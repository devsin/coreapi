# Helper targets for migrations and sqlc codegen

MIGRATIONS_DIR ?= db/migrations
DB_URL ?= $(DATABASE_URL)
MIGRATE_BIN ?= migrate
SQLC_BIN ?= sqlc
MIGRATION_NAME ?= add_change

.PHONY: migrate-up migrate-down migrate-new migrate-force sqlc fmt

migrate-up:
	@if [ -z "$(DB_URL)" ]; then echo "DB_URL or DATABASE_URL must be set"; exit 1; fi
	$(MIGRATE_BIN) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down:
	@if [ -z "$(DB_URL)" ]; then echo "DB_URL or DATABASE_URL must be set"; exit 1; fi
	$(MIGRATE_BIN) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

migrate-new:
	@if [ -z "$(MIGRATION_NAME)" ]; then echo "MIGRATION_NAME is required"; exit 1; fi
	$(MIGRATE_BIN) create -ext sql -dir $(MIGRATIONS_DIR) -seq $(MIGRATION_NAME)

migrate-force:
	@if [ -z "$(DB_URL)" ]; then echo "DB_URL or DATABASE_URL must be set"; exit 1; fi
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required (e.g., VERSION=20240101120000)"; exit 1; fi
	$(MIGRATE_BIN) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(VERSION)

sqlc: sqlc-install
	$(SQLC_BIN) generate

sqlc-install:
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

golang-migrate-install:
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

golangci-lint-install:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	golangci-lint run --fix

run:
	go run ./cmd/coreapi/...