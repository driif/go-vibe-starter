# AGENTS.md

The single source of truth for agents working in this repo. Read before making changes.
`CLAUDE.md` imports this file.

## 1. Hard constraints

- Never add AI co-authorship, attribution trailers, or generated-with lines to commits, PRs or issues.
- Never hand-edit generated `internal/gen/**`, `internal/db/gen/**` or `agents/skills/**`.
  Change proto, SQL or canonical skills and run the corresponding generator.
- Apply migrations through `make migrate-up`; no ad-hoc schema changes.
- Write minimal code and comment only when the reason is not clear from the code.
- Ask when requirements are unclear. Work within decisions already approved.
- Validate before claiming done: `make check` must pass.

## 2. Tech stack

Go | stdlib ServeMux + go-pkgz/routegroup | Connect v2 + buf + protovalidate | pgx/v5 + sqlc |
goose | Cobra | Keycloak OIDC | slog JSON | PostgreSQL | testify + testcontainers-go.
Read [stack notes](docs/agents/stack-notes.md) before writing transport code: Connect v2 and
routegroup APIs differ from older material. Go and generator versions live in `go.mod` and Makefile.

## 3. Layout

This is the only maintained package map.

| Path | Owns |
|---|---|
| `main.go`, `cmd/` | process entry point, run, db migration and probe commands |
| `proto/api/v1/` | wire contract |
| `internal/gen/` | protobuf and Connect generated output |
| `internal/domain/<domain>/` | stores and business rules; no transport imports |
| `internal/rpc/<domain>/` | thin handlers, request/store/response mapping |
| `internal/rpc/interceptor/` | errors, validation and RPC logging concerns |
| `internal/server/server.go` | HTTP stack construction and lifecycle |
| `internal/server/routes.go` | hub calling one mount function per domain |
| `internal/server/routes_<domain>.go` | mount file with that domain's dependencies |
| `internal/server/middleware/` | auth, access logs, CORS, request ids, metrics and recovery |
| `internal/server/config/` | typed environment parsing and validation |
| `internal/identity/` | principal; Keycloak verifier and test issuer in subpackages |
| `internal/apperr/` | transport-independent domain errors |
| `internal/reqlog/` | per-request logger and result carrier |
| `internal/db/` | pool, migrations and startup schema check |
| `internal/db/gen/` | sqlc output |
| `sql/migrations/`, `sql/queries/`, `sql/embed.go` | schema, queries, embedded migration FS |
| `internal/test/` | Postgres and real HTTP/OIDC integration test harness |
| `frontend/` | reserved for a separately chosen frontend stack |

## 4. Code generation

| Source changed | Command |
|---|---|
| `proto/**` | `make gen-proto` |
| `sql/queries/**`, `sql/migrations/**` | `make sqlc` |
| `.claude/skills/**` | `make sync-skills` |
| all generated Go | `make gen` |

Commit sources and generated output together. Generation deletes the old output first.

## 5. Commands

Run `make help` for the complete list. `make tools` installs the pinned tools. `make init`
initializes a derived project. `make dev-up` starts Postgres; `make keycloak-up` starts dev
Keycloak. `make migrate-up`, `make migrate-down`, `make migrate-status` operate embedded
migrations. `make run` builds and starts the server. `make check` runs drift, lint, race tests
and the skills mirror check. `make vuln`, `make image-smoke`, `make smoke` check advisories,
the non-root image and real Keycloak respectively. `make labels` creates issue labels.

## 6. Configuration

Environment variables only; process values precede `.env.local`, then built-in defaults.
Make creates `.env.local` from `.env.example` for local run/migration targets. Tests ignore it.
Keep [.env.example](.env.example) and [the reference](docs/agents/env-reference.md) synchronized.

`AUTH_PROVIDER` has no default; `APP_ENVIRONMENT` defaults to `production`. Provider `none`
is permitted only in `development`, `dev`, `local`, `test` and returns the `local-dev` principal
with no roles unless explicitly configured. `keycloak` requires issuer and audience. Both require
`DATABASE_URL`. Handlers read the caller through `identity.FromContext(ctx)`.

## 7. Go conventions

See [Go style](docs/agents/go-style.md). Use early returns, immediate error checks, `%w` where
it adds debugging context, structured slog with snake_case keys, and lowercase in-code comments.
Use consumer-defined small interfaces only where a real test double or boundary needs them.
Handlers translate between the RPC and store; rules and queries live in domain packages.
Return `apperr` errors; the error interceptor maps them to wire codes and ErrorInfo reasons.

HTTP middleware runs before body decoding and can reject unauthenticated requests cheaply.
Interceptors operate on decoded RPC messages and return RPC errors. See [request path](docs/agents/request-path.md).
Use parameterized owner-scoped SQL and `pgx.BeginFunc` for transactions. Tests use the harness in
`internal/test` with real Postgres, external test packages, generated clients and `t.Parallel()`.
Docker is a test prerequisite. Use table cases, testify require/assert, `t.Helper()` and `t.TempDir()`.
No attribution trailers or "Test plan" sections in commits/PRs.

## 8. The worked example

The permanent `whoami` slice shows the [contract](proto/api/v1/whoami.proto),
[handler](internal/rpc/whoami/whoami.go), [mount](internal/server/routes_whoami.go) and
[test](internal/rpc/whoami/whoami_test.go). It reads the authenticated caller without persistence.
The `go-feature` skill provides static persistence snippets: migration → queries → domain store →
handler → mount → integration test. Follow one thin path through all required layers before widening.

## 9. Development workflow

Two substrates, one bridge:

- **GitHub Issues hold intent** — `spec` (what and why), `ticket` (one tracer-bullet slice),
  `decision` (an open question), `bug`. Blocking edges are `Blocked by #N` lines in the issue body.
  Triage labels: `needs-info`, `needs-grilling`, `ready`, `blocked`, `wontfix`. `make labels`
  creates all of them.
- **`docs/plans/*.md` holds execution** — one ralphex-format plan file, run unattended by the
  `ralphex` binary, archived to `docs/plans/completed/`. Disposable.

```
loose idea ─▶ /grill-me ─▶ /to-spec ─▶ /to-tickets ─┬─▶ /implement            (interactive)
                                                     └─▶ /to-plan + ralphex   (unattended)
```

On-ramps: a bug arrives → `diagnosing-bugs`. Work too big to hold in one session → `/wayfinder`.
A repo just created from the template → `/setup-project`. Not sure → `/vibe`.

## 10. Skill index

Canonical skills live in `.claude/skills/`; Claude Code discovers them automatically. Other tools
read the identical mirror at `agents/skills/<name>/SKILL.md`.

**Invoke by name** (these carry no description the agent can see, so nothing but you reaches them):

| Skill | When to reach for it |
|---|---|
| `/vibe` | The map — which skill fits this situation |
| `/setup-project` | Once, in a repo created from this template |
| `/grill-me` | Sharpen a plan or design before any code is written |
| `/grill-with-docs` | Same, and leave `CONTEXT.md` terms and ADRs behind |
| `/to-spec` | Turn what the conversation settled into a spec issue |
| `/to-tickets` | Break a spec into tracer-bullet ticket issues |
| `/to-plan` | Turn tickets into a ralphex plan file under `docs/plans/` |
| `/implement` | Build one ticket end to end with you in the loop, then close it |
| `/triage` | Move issues through the label state machine |
| `/wayfinder` | Chart work too big for one session as `decision` issues |
| `/improve-architecture` | Rank deepening opportunities, then grill the one you pick |
| `/handoff` | Compact this session into a handoff another can start from |

**Fire on their own** (the agent reaches for these when the task fits):

| Skill | Covers |
|---|---|
| `go-feature` | The vertical slice: proto → migration → sqlc → domain → handler → mount → test |
| `go-middleware` | HTTP middleware and Connect interceptors |
| `tdd` | Go tests, test-first, and what makes one worth keeping |
| `code-review` | Two-axis review (standards, spec) over the six Go reviewers |
| `diagnosing-bugs` | Reproduce → minimize → hypothesize → instrument → fix → verify |
| `grilling` | The interview primitive the grill skills run on |
| `domain-modeling` | `CONTEXT.md` terms and ADRs |
| `codebase-design` | Deep modules, seams, interface depth in Go terms |
| `research` | Questions answered against primary sources, cited |
| `resolving-merge-conflicts` | A merge or rebase stopped on conflicts |
| `writing-style` | Commit messages, PR bodies, issue bodies, review comments |
| `writing-for-agents` | Writing or editing a skill, `AGENTS.md`, or a doc a skill points at |

## 11. Review engine

`code-review` runs the six Go reviewers in `.claude/skills/code-review/reviewers/` — as read-only
parallel subagents on Claude Code (`.claude/agents/go-*.md`), sequentially elsewhere. Two modes:
**comprehensive** (quality, implementation, testing, simplification, documentation, smells) and
**critical-only** (quality + implementation). Every finding is **verified at `file:line`** and
marked CONFIRMED or FALSE POSITIVE before anything is fixed. Severity: critical / major / minor.

## 12. Editing skills

Edit `.claude/skills/` only, then run `make sync-skills` to regenerate the `agents/skills/` mirror.
Never hand-edit the mirror. `make check` fails when it is stale. The `writing-for-agents` skill covers how to write one.
