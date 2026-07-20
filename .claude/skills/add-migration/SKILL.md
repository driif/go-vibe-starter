---
name: add-migration
description: Use when changing the database schema — scaffolds a goose migration in migrations/ with Up/Down and reminds how to apply it.
---

# Add migration

Ask for a short description of the schema change if not given, then create a goose migration.

## 1. Create the file

`migrations/<NNNN>_<snake_description>.sql`, where `<NNNN>` is the next zero-padded sequence
number (match whatever convention already exists in `migrations/`; if empty, start at `0001`).

```sql
-- +goose Up
-- +goose StatementBegin
<forward SQL>
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
<reverse SQL>
-- +goose StatementEnd
```

Always write a real `Down` that reverses the `Up`.

## 2. Apply it

Run the app's migrate command (see `cmd/db_migrate.go`):

```bash
go run . db migrate   # or: ./bin/app db migrate
```

## 3. Regenerate queries if needed

If any sqlc query depends on the new or changed schema, run `make sqlc` afterward so the generated
types match.
