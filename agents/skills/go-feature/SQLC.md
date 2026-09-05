# sqlc queries (step 2 of the slice)

Hand-written SQL in `sql/queries/<domain>.sql`, one file per domain, is the source; `make sqlc`
generates `internal/db/gen`. Worked example: `sql/queries/notes.sql` →
`internal/db/gen/notes.sql.go`.

## The query file

```sql
-- name: CreateNote :one
-- CreateNote inserts a note owned by the caller. The id and timestamps come
-- from the column defaults.
INSERT INTO notes (owner_id, title, body)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetNote :one
-- GetNote matches on owner_id as well as id, so a caller can never address
-- another caller's row: a foreign id matches nothing.
SELECT * FROM notes
WHERE id = $1 AND owner_id = $2;

-- name: ListNotesByOwner :many
SELECT * FROM notes
WHERE owner_id = $1
ORDER BY created_at DESC;

-- name: DeleteNote :exec
DELETE FROM notes
WHERE id = $1 AND owner_id = $2;
```

The `-- name:` line names the generated method, in `PascalCase`. Comment lines between it and the
SQL become the godoc on that method and on its entry in `gen.Querier`, so write them for the caller.

| Annotation | Generates | Returns nothing found as |
|---|---|---|
| `:one` | `(gen.Note, error)` | `sql.ErrNoRows` |
| `:many` | `([]gen.Note, error)` | empty slice (`emit_empty_slices`) |
| `:exec` | `error` | no error — deleting nothing succeeds |
| `:execrows` | `(int64, error)` | `0` rows affected |

Pick `:execrows` over `:exec` when the caller must tell "updated" from "matched nothing"; pick
`:exec` when the operation is idempotent, as `DeleteNote` is.

## Parameters

Positional `$1, $2, …` always. Values reach the database as bind parameters, never as text spliced
into the statement.

- One parameter generates a bare argument: `ListNotesByOwner(ctx, ownerID string)`.
- Two or more generate a params struct: `GetNote(ctx, gen.GetNoteParams{ID: id, OwnerID: owner})`.
- `sqlc.arg(owner_id)` names a parameter when `$1` would be generated with a poor name, e.g.
  `WHERE created_at > sqlc.arg(since)`.
- `sqlc.narg(title)` makes it nullable, generating `sql.NullString` — the shape for an optional
  filter: `WHERE (sqlc.narg(title)::text IS NULL OR title = sqlc.narg(title))`.

Every query that returns or removes caller-visible rows takes the owner as a parameter and matches
on it in the `WHERE` clause. Ownership belongs in the SQL, not in an `if` in the handler: a query
that can only return the caller's rows cannot be misused by the handler above it.

## Generating

```bash
make sqlc          # sqlc generate
```

`sqlc.yaml` fixes the output shape, and three settings decide how handlers consume it:

- `emit_interface: true` → `gen.Querier`, an interface carrying every generated method, with
  `var _ Querier = (*Queries)(nil)`. That interface is what makes the store a **seam**: a handler
  declares the two or four methods it needs (see [`HANDLERS.md`](HANDLERS.md)) and `*gen.Queries`
  satisfies it.
- `emit_empty_slices: true` → a `:many` with no matches renders as `[]`, not `null`.
- `emit_json_tags` + `emit_db_tags` → `gen.Note` carries both tag sets. It is a row, not a wire
  type: map it to the generated `api.*` type before rendering.

Types follow the column: `uuid` → `uuid.UUID`, `timestamptz` → `time.Time`, `text NOT NULL` →
`string`, a nullable column → `sql.NullString` and friends. Commit `internal/db/gen` with the query
change; CI regenerates and fails on **drift**.

## When generation fails

- `relation "x" does not exist` / `column "y" does not exist` — the migration is missing or not
  saved. sqlc reads `migrations/`, not the running database. Go back to
  [`MIGRATIONS.md`](MIGRATIONS.md).
- `query has no name` — the `-- name:` annotation is missing or misspelled.
- A query whose result columns sqlc cannot infer (unusual `CASE`, some CTEs) generates an awkward
  type. Reshape the SQL rather than post-processing the result in Go.
