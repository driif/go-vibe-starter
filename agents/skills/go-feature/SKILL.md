---
name: go-feature
description: Building business logic in this repo — the vertical slice from schema to HTTP. Use when adding or changing an endpoint, a migration, a sqlc query, a handler or a route, when a ticket says "add /v1/<thing>", or when a schema change has to reach the API.
---

A feature here walks **the slice**: migration → sqlc query → OpenAPI → handler → route → test. The
steps below are that slice in order, each with its completion criterion and the reference file
holding its depth.

Two sources of truth drive it. **SQL-first**: `migrations/` is the schema and `sql/queries/*.sql`
are the queries, and sqlc derives the Go from them. **OpenAPI-first**: `oapi/openapi.yaml` is the
HTTP contract, and oapi-codegen derives the Go from it. `make gen` runs both.

**Never hand-edit `internal/db/gen/*.go` or `internal/api/*.gen.go`** — change the SQL or the spec
and regenerate. A generated file edited by hand is reverted by the next `make gen` and caught by CI.

The worked example is the `DEMO` notes slice, one file per step:
`migrations/0001_notes.sql` · `sql/queries/notes.sql` · `oapi/openapi.yaml` ·
`internal/api/handlers/notes.go` · `internal/api/router/routes_notes.go` ·
`internal/api/handlers/notes_test.go`. Read the one you are about to write.

Take one **tracer bullet** at a time: the thinnest end-to-end path (one endpoint, one query) all the
way to a passing test, then widen. Half-finished slices at three layers cost more than one finished
slice.

## 1. Schema change → [`MIGRATIONS.md`](MIGRATIONS.md)

Skip when the feature adds no table or column; start at step 2.

Write `migrations/NNNN_snake_name.sql` in goose format, with a `Down` that reverses the `Up`.

**Done when** `make migrate-up` applies it against a running database (`make db-up`) and
`./bin/app db migrate down` then `up` round-trips without error.

## 2. Queries → [`SQLC.md`](SQLC.md)

Write annotated, parameterized SQL in `sql/queries/<domain>.sql`. Scope every row-returning query by
its owner column so the handler cannot be tricked into reading a foreign row.

**Done when** `make sqlc` regenerates `internal/db/gen` with each new method on `gen.Querier`, and
`git diff` touches only `internal/db/gen`.

## 3. API surface → [`OPENAPI.md`](OPENAPI.md)

Skip when the feature has no HTTP surface (a background job stops at step 2).

Add the path, operation and schemas to `oapi/openapi.yaml`, including every error status the handler
can return.

**Done when** `make gen-oapi` puts the operation on `ServerInterface` in
`internal/api/openapi_server.gen.go` and its request/response structs in `openapi_types.gen.go`.

## 4. Handler + route → [`HANDLERS.md`](HANDLERS.md)

Write the handler in `internal/api/handlers/<domain>.go`, register the route in
`internal/api/router/routes_<domain>.go`, and call `register<Domain>(s)` from the **hub**
(`internal/api/router/routes.go`).

**Done when** `go build ./...` is clean, every response path exits through `rest.RenderJSON` or
`errs.Write`, and every query the handler runs carries the **principal**'s subject as its owner
argument.

## 5. Tests → [`TESTS.md`](TESTS.md)

Invoke `tdd` for the shape of the test and the **red-green** order; `TESTS.md` says which layer of
the slice carries which test and what the handler test has to cover.

**Done when** `make test` exits 0 and the handler test has a case for each status listed in the
spec — including the foreign-owner case, which must be 404 rather than someone else's row.

## Finish

```bash
make gen && git diff --exit-code   # no drift: generated output matches its source
make test                          # go test -race ./...
make lint                          # exits 0
gofmt -l .                         # prints nothing
```

A dirty `git diff` after `make gen` means the committed generated code no longer matches the SQL or
the spec. Commit the regenerated files; CI runs the same check.
