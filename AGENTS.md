# AGENTS.md — go-vibe-starter

The single source of truth for any AI agent working in this repo (Claude Code, Codex, Pi, opencode).
Read it before making changes. `CLAUDE.md` imports it.

---

## 1. Hard constraints

- **Never add AI co-authorship to commits, PRs, or issues** — no `Co-Authored-By: Claude …`, no
  `🤖 Generated with …`, no mention of Claude, Anthropic, or AI assistance anywhere in a commit
  message, PR body, or issue body.
- **Never hand-edit generated files** — `internal/api/*.gen.go` (from `oapi/openapi.yaml`) and
  `internal/db/gen/*.go` (from `sql/queries/*.sql`). Change the source, run `make gen`.
- **Run migrations through the CLI** — `make migrate-up`, never ad-hoc SQL against the database.
- **Write minimal code** — the simplest thing that works. No premature abstractions, no speculative
  features, no config knobs nobody asked for.
- **Comment sparingly** — see [`docs/agents/go-style.md`](docs/agents/go-style.md). A comment per
  change is a defect, not thoroughness.
- **Ask before implementing** when requirements are unclear or several approaches are reasonable.
- **Validate before claiming done** — `make test` and `make lint` both green.

## 2. Tech stack

Production-ready Go web server for rapid prototyping. Explicit, understandable infrastructure with
minimal boilerplate; LLM-first (SQL over ORMs, explicit over magic).

Go 1.25 | chi v5 | sqlc | oapi-codegen (chi-server) | goose | Cobra CLI | Keycloak OIDC (optional) |
`log/slog` JSON | PostgreSQL | testify + testcontainers-go | go-pkgz/rest

## 3. Entry points

| Purpose | Path |
|---|---|
| CLI → server startup | `main.go` → `cmd/root.go` → `cmd/run.go` |
| Server bootstrap + middleware stack | `internal/server/server.go` |
| Env-driven config | `internal/server/config/server_config.go` |
| Auth config + startup validation | `internal/server/config/auth_config.go` |
| Route hub (one file per domain) | `internal/api/router/routes.go` |
| Auth middleware + role guards | `internal/server/auth/auth.go` |
| Error response shape | `internal/server/errs/error.go` |
| OpenAPI spec | `oapi/openapi.yaml` |
| Test helpers | `internal/server/test/` |
| **Worked example of a feature** | the `notes` slice — see §8 |

## 4. Code generation

| Edited | Run |
|---|---|
| `oapi/openapi.yaml` | `make gen-oapi` |
| `sql/queries/*.sql` | `make sqlc` |
| `migrations/*.sql` | nothing — goose applies it via `make migrate-up` |
| `.claude/skills/**` | `make sync-skills` |

## 5. Commands

```
make help         # list every target
make init         # bootstrap a new project from this template
make build / run  # compile to bin/app / build and start
make test         # go test -race ./...
make cover        # coverage profile + summary
make lint         # golangci-lint run
make fmt          # gofmt + goimports
make gen          # gen-oapi + sqlc
make migrate-up   # apply pending migrations
make db-up/db-down# docker compose up -d / down
make labels       # create the GitHub issue labels the skills use
make sync-skills  # mirror .claude/skills -> agents/skills
make tools        # install the pinned codegen and lint tools
```

## 6. Configuration

Environment variables only, no config files. `.env.local` is loaded outside tests.
Every variable with its default: [`.env.example`](.env.example) and
[`docs/agents/env-reference.md`](docs/agents/env-reference.md) — keep both in sync when you add one.

The non-obvious one:

- **`AUTH_PROVIDER`** (default `none`) — `none` skips token verification and runs every request as a
  static dev principal (`AUTH_DEV_SUBJECT`, `AUTH_DEV_ROLES`), so a fresh clone boots with no
  external dependencies. `keycloak` verifies bearer tokens against the realm's JWKS. Startup
  **fails** if `AUTH_PROVIDER=none` is combined with `APP_ENVIRONMENT=production`.

Handlers reach the caller the same way under both providers:
`auth.PrincipalFromContext(r.Context())`. `auth.TokenFromContext` only returns a token under
`keycloak`. The middleware itself comes from `s.Authenticator()`.

## 7. Go conventions

The short version — the depth is in [`docs/agents/go-style.md`](docs/agents/go-style.md).

- **Logging:** `log/slog`, structured, `snake_case` keys. Never `fmt.Println`.
- **HTTP:** chi v5; middleware are `func(http.Handler) http.Handler`.
- **Handlers:** in `internal/api/handlers`; errors via `errs.Write(w, status, err)`; path params via
  `chi.URLParam`; routes registered in `internal/api/router/routes_<domain>.go`.
- **Interfaces:** defined in the **consumer** package, 1–3 methods, never for a single
  implementation with no test double.
- **Control flow:** return early; validate at the top.
- **Errors:** check immediately, wrap with `%w` only where the context helps debugging.
- **Comments:** lowercase in-code, godoc on exported identifiers, no history comments.
- **Tests:** table-driven, testify `require`/`assert`, one `_test.go` per source file, `t.Helper()`,
  `go test -race`.
- **Database:** parameterized SQL, scoped by owner in the query itself.
- **Commits/PRs:** no trailers, no "Generated with" lines, no "Test plan" sections.

## 8. The worked example

`notes` is a complete reference vertical slice, marked `// DEMO:` in every file and removed by
`make init`. When building a feature, copy its shape:

```
migrations/0001_notes.sql          schema
sql/queries/notes.sql              queries      -> internal/db/gen (generated)
oapi/openapi.yaml                  API surface  -> internal/api/*.gen.go (generated)
internal/api/handlers/notes.go     handler
internal/api/handlers/notes_test.go tests
internal/api/router/routes_notes.go route registration
```

The `go-feature` skill walks this path step by step.

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
| `go-feature` | The vertical slice: migration → sqlc → OpenAPI → handler → route → test |
| `go-middleware` | Adding or changing a chi middleware |
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
Never hand-edit the mirror. The `writing-for-agents` skill covers how to write one.
