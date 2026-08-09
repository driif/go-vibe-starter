# CONTEXT

The project's **ubiquitous language** and module map. Agents read this before designing or naming
anything, so the words in the code, the issues, and the conversation stay the same words.

Maintained by the `domain-modeling` skill. Decisions that shaped it live as ADRs in
[`docs/adr/`](docs/adr/). `make init` resets the domain section for a new project — the
infrastructure terms below are true of the template itself and worth keeping.

---

## Domain terms

> Replace this section with your own domain. One entry per term: the term, then one or two sentences
> that make it precise. A term earns a place here when it was ambiguous until someone pinned it down.

**Note** — the removable reference slice's entity. A piece of text owned by exactly one principal.
Deleted by `make init` along with the rest of the demo.

## Infrastructure terms

**Principal** — the authenticated caller, carried on the request context and reached with
`auth.PrincipalFromContext`. Its `Subject` is the stable owner id that rows are scoped by. Present
under both auth providers, so handlers never branch on which one is configured.

**Auth provider** — what verifies a request. `none` injects a fixed dev principal so the server runs
with no external dependencies; `keycloak` verifies bearer tokens against the realm's JWKS. Selected
by `AUTH_PROVIDER`; refused in production when set to `none`.

**Vertical slice** — one feature taken all the way through the stack: migration, query, API surface,
handler, route, test. The unit of work a ticket describes and the `go-feature` skill builds. The
opposite of a horizontal layer ("all the migrations for the epic").

**Tracer bullet** — a ticket thin enough to finish in one pass but complete enough to leave the
system working. Tickets are tracer bullets; that is what makes them independently shippable.

**Spec / ticket / decision** — the three issue kinds. A spec says what and why; a ticket is one
tracer bullet of a spec; a decision is an open question whose resolution is a choice, not code.

**Plan** — a `docs/plans/*.md` file in ralphex's format. Generated from tickets, executed unattended,
archived to `completed/`. Disposable: the issue is the record, the plan is the run.

**Seam** — a boundary a test can substitute at. The sqlc-generated `Querier` is this codebase's main
one: handlers depend on the interface, tests supply a fake, and no database is needed to test
handler logic.

**Generated file** — anything under `internal/db/gen/` or matching `internal/api/*.gen.go`. Produced
by `make gen` from SQL and the OpenAPI spec. Edited only by changing its source.

---

## Module map

| Path | What lives here |
|---|---|
| `main.go`, `cmd/` | Cobra CLI — `run`, `db migrate`, `db seed` |
| `internal/server/` | Server struct, middleware stack, lifecycle |
| `internal/server/config/` | Every env var, parsed once, validated at startup |
| `internal/server/auth/` | Provider selection, principal injection, role guards |
| `internal/server/errs/` | The uniform error response shape |
| `internal/server/middleware/` | Repo-specific chi middleware |
| `internal/api/handlers/` | HTTP handlers, one file per domain |
| `internal/api/router/` | Route registration, one `routes_<domain>.go` per domain |
| `internal/db/gen/` | sqlc output — generated, never edited |
| `pkg/` | Reusable, outside-facing helpers: `db`, `dotenv`, `keycloak`, `tests` |
| `migrations/`, `sql/queries/` | Schema and hand-written SQL |
| `oapi/` | The OpenAPI spec — source of truth for the API surface |

**Depth note.** `internal/` is the hiding mechanism: anything under it can change freely. `pkg/` is a
promise to the outside world. Moving something from `internal/` to `pkg/` is a decision worth an ADR.
