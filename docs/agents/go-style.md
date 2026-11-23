# Go house style

[AGENTS.md](../../AGENTS.md) holds the stack, layout and conventions. This file adds the details.

Define small interfaces where the consumer needs a boundary; do not wrap a concrete implementation
without a second use. Domain stores use generated queries directly. Handlers map request, principal,
store and response, with no business rules or SQL. Domain packages import no transport packages.

Return early, check errors immediately, and take `context.Context` first on I/O functions.
Wrap with `%w` when the context helps diagnosis. Return domain errors through `apperr`; callers
observe a stable reason and safe message. Never send raw database or provider errors to callers.

Use structured `log/slog`, snake_case keys and `error` for errors. Never log tokens, authorization
headers, full request bodies, query strings, database URLs or passwords. Request logs use the
request-scoped logger. Durations labeled milliseconds contain milliseconds.

Comments explain hidden reasons and invariants. In-code comments are lowercase; exported identifier
docs start with the identifier. No history comments or line-by-line narration.

Tests are table-driven with testify require/assert. One `_test.go` per source file. Use `t.Helper()`
and `t.TempDir()`. Integration tests use `internal/test`, real Postgres and generated clients in an
external package. Test a foreign owner's row, every error reason, rejected input and no token.
Docker is required. Do not fake persistence merely to mirror implementation branches.

Write parameterized SQL with named arguments such as `@owner`; ownership belongs in every query.
Use `pgx.ErrNoRows` and `:execrows` counts to map missing rows; use `pgx.BeginFunc` for transactions.
Migrations under `sql/migrations/` contain reversible goose Up/Down sections. Generated output is
regenerated with `make gen`. A change is done when `make check` exits 0.
