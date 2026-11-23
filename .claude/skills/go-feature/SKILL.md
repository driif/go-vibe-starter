---
name: go-feature
description: Builds business logic in this repo as a vertical slice from contract to storage. Use when adding or changing an RPC, a proto message, a migration, a sqlc query, a store or a handler, or when a schema change has to reach the API.
---

Walk **the slice** in order: contract → migration → query → domain → handler → mount → test.
Take one tracer bullet all the way through before widening it. Read `AGENTS.md` for the layout.
The permanent example is `proto/api/v1/whoami.proto`, `internal/rpc/whoami/` and
`internal/server/routes_whoami.go`. It has no persistence; the references provide static snippets
for a hypothetical `tasks` domain. Generated files are output, never the edit target.

1. **Contract** — [PROTO.md](PROTO.md). Define messages and RPCs, input rules and safe reads.
   Done when `make gen-proto` adds the method to the generated handler interface.
2. **Schema** — [MIGRATIONS.md](MIGRATIONS.md). Skip when no schema changes.
   Done when up/down/up works through the migration commands.
3. **Queries** — [SQLC.md](SQLC.md). Named parameters and ownership in every statement.
   Done when `make sqlc` generates the store's needed queries.
4. **Domain** — [DOMAIN.md](DOMAIN.md). Keep rules, queries and transactions out of transport.
   Done when a store method performs the whole operation with stable domain errors.
5. **Handler and mount** — [RPC.md](RPC.md). Map input, principal, store and response.
   Done when the generated interface is fully implemented and the hub mounts the service.
6. **Tests** — [TESTS.md](TESTS.md). Use the real HTTP/OIDC/Postgres harness.
   Done when behavior, ownership, input rejection and missing auth have integration cases.

Finish with `make check`. Commit source and regenerated code together.
