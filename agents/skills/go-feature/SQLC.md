# Queries

Write named, parameterized statements in `sql/queries/<domain>.sql`.

```sql
-- name: GetTask :one
SELECT * FROM tasks WHERE id = @id AND owner = @owner;

-- name: CreateTask :one
INSERT INTO tasks (owner, title) VALUES (@owner, @title) RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM tasks WHERE id = @id AND owner = @owner;
```

Every statement carries the owner. Ownership in SQL makes another caller's id indistinguishable
from a missing row. A `:one` miss returns `pgx.ErrNoRows`; `:many` returns an empty slice;
`:execrows` returns the affected count so deletes can map zero to not-found.

`make sqlc` deletes the output directory first and skips when there are no query files. Do not put
handwritten code there. The domain store calls `gen.New(pool)` directly: there is no generated
interface. UUID and non-null timestamps are overridden to `uuid.UUID` and `time.Time`; nullable
values use `pgtype`. sqlc reads embedded schema sources, not a running database.
