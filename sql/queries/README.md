# SQL queries (sqlc)

Hand-written SQL for [sqlc](https://sqlc.dev). One file per domain, e.g. `users.sql`.
Annotate each query: `-- name: GetUser :one` (`:one` / `:many` / `:exec`).
Schema is read from `../../migrations` (goose format). Run `make sqlc` to regenerate types.
Use the `add-sqlc-query` skill to add queries.
