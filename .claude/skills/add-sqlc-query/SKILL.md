---
name: add-sqlc-query
description: Use when adding a database query — write annotated SQL in sql/queries, run make sqlc, then use the generated type-safe code.
---

# Add sqlc query

Write SQL by hand and let sqlc generate type-safe Go. Schema comes from goose migrations in
`migrations/`; queries live in `sql/queries/`; output goes to `internal/db/gen` (see `sqlc.yaml`).

## 1. Write the query

Add to (or create) `sql/queries/<domain>.sql`. Annotate each query with a name and cardinality:

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: CreateUser :exec
INSERT INTO users (id, email) VALUES ($1, $2);
```

- `:one` returns a single row, `:many` a slice, `:exec` no rows.
- Always use positional parameters (`$1`, `$2`) — never concatenate values into SQL.

## 2. Generate

```bash
make sqlc
```

This writes type-safe code into `internal/db/gen`. Never hand-edit generated files.

## 3. Use it

Call the generated `Queries` methods from application code (e.g. `internal/api/handlers`).

## Prerequisite

The referenced tables must exist in `migrations/`. If the schema (or a needed table) is missing,
use `add-migration` first. If sqlc has no schema at all yet, say so rather than inventing types —
`sqlc generate` errors until at least one query and its schema exist.
