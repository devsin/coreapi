# Core API — Agent Instructions

Go REST API. See root `CLAUDE.md` for full context.

## Quick Reference

```bash
go build ./...          # must pass
make lint               # must pass (golangci-lint, zero issues)
make sqlc               # after editing db/queries/*.sql
make migrate-new MIGRATION_NAME=xxx  # new migration pair
```

## Adding a New Domain

1. Create `internal/<domain>/` with `handler.go`, `service.go`, `repo.go`, `model.go`
2. Add SQL queries in `db/queries/<table>.sql`, run `make sqlc`
3. Wire in `cmd/coreapi/main.go`: `pool → repo → service → handler`
4. Register routes in `internal/app/routes.go`

## Adding a New Endpoint

1. Add SQL query (`db/queries/`) → `make sqlc`
2. Add repo method wrapping generated query
3. Add service method with business logic + sentinel errors
4. Add handler method + `handleServiceError` case
5. Register route in `routes.go`
6. `go build ./...` + `make lint`

## SQL Query Patterns

```sql
-- name: GetFoo :one
SELECT * FROM foos WHERE id = $1;

-- name: ListFoos :many
SELECT * FROM foos WHERE owner_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateFoo :one
UPDATE foos SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description)
WHERE id = $1 AND owner_id = $2
RETURNING *;
```

- Use `sqlc.narg('name')` for optional/nullable params → generates `pgtype.X`
- Convert `*uuid.UUID` to pgtype: `pgtype.UUID{Bytes: *id, Valid: true}`

## Linting

Key enabled linters: `gosec`, `revive`, `errorlint`, `tagliatelle` (JSON snake_case), `bodyclose`, `gocritic`.
Suppress with: `//nolint:lintername // reason`

## Testing

No test suite exists yet. When adding tests, use standard `*_test.go` files with `testing` package.
