# AGENTS.md — go-vibe-starter

The single source of guidance for any AI agent (Claude Code, Codex, Pi, opencode) working in this
repo. Read this before making changes. `CLAUDE.md` points here.

For the detailed directory map, see [`docs/agents/README.md`](./docs/agents/README.md).

---

## 1. Project & tech stack

Production-ready Go web server foundation for rapid prototyping. Explicit, understandable
infrastructure with minimal boilerplate; LLM-first (SQL over ORMs, explicit over magic).

- **Language:** Go 1.25
- **CLI:** cobra (`cmd/`)
- **HTTP router:** chi v5 (`internal/api`, `internal/server`)
- **Auth:** Keycloak / OIDC / JWT (`pkg/keycloak`, `internal/server/auth`)
- **Database:** PostgreSQL (`pkg/db`)
- **Migrations:** goose (`migrations/`)
- **Queries:** sqlc (`sql/queries/` → `internal/db/gen`) — SQL over ORM
- **API:** OpenAPI-first via oapi-codegen (`oapi/openapi.yaml` → `make gen-oapi`)
- **Logging:** `log/slog`
- **Tests:** stdlib `testing` + `stretchr/testify`

## 2. Core principles

Write **minimal, pragmatic** code that solves the problem at hand — the simplest thing that works
correctly. Add complexity only when explicitly needed. Favor straightforward over clever. Ask when
requirements are unclear rather than over-engineering.

## 3. Go conventions

- **Logging:** `log/slog`, structured. Never `fmt.Println` or a third-party logger.
- **HTTP:** chi v5; middleware are `func(http.Handler) http.Handler`. No other HTTP framework.
- **Handlers:** in `internal/api/handlers`; write errors with `errs.Write(w, status, err)`; read
  path params via `chi.URLParam`. Register routes in `internal/api/router/routes.go` with the
  correct auth middleware.
- **Interfaces:** define them in the **consumer** package, not the producer. Keep them small (1–3
  methods). Don't create an interface for a single implementation.
- **Control flow:** return early to avoid deep nesting; validate inputs at the top.
- **Errors:** check immediately and return. Wrap with `fmt.Errorf("...: %w", err)` only when the
  context helps debugging — don't stack wrapping layers.
- **Comments:** lowercase in-code comments; document exported identifiers. **No history/changelog
  comments** ("now uses X", "previously Y") — describe current state only.
- **Tests:** table-driven with descriptive case names; `require` for fatal assertions, `assert`
  for non-fatal; one test file per source file; `httptest` for handlers; `go test -race ./...`.
- **Database:** SQL over ORM. Schema via goose migrations; queries via sqlc (parameterized, never
  string-concatenated). Never hand-edit `*.gen.go`.
- **Config:** all configuration via environment variables (`internal/server/config/env`).
- **Commits/PRs:** no commit trailers or "Generated with…" lines; no "Test plan" sections in PRs.

## 4. Configuration

All config is via environment variables. Core set (full list:
[`docs/agents/env-reference.md`](./docs/agents/env-reference.md)):

| Variable | Default | Description |
|---|---|---|
| `APP_ENVIRONMENT` | `development` | Environment name |
| `SERVICE_PORT` | `:9880` | HTTP server port |
| `KEYCLOAK_URL` | `http://localhost:8080` | Keycloak base URL |
| `KEYCLOAK_REALM` | `myrealm` | Keycloak realm |
| `KEYCLOAK_CLIENT_ID` | `myclient` | Keycloak client ID |

Endpoints: REST API `http://localhost:9880`, Keycloak `http://localhost:8080`.

## 5. Commands

```
make build        # build bin/app
make run          # build and run the server
make test         # go test -race ./...
make lint         # golangci-lint run
make gen-oapi     # regenerate OpenAPI types + chi server
make sqlc         # sqlc generate (needs queries in sql/queries)
make sync-skills  # mirror .claude/skills -> agents/skills
```

## 6. Development workflow

The local skills encode one lifecycle. Use the skill that matches the phase:

```
UNDERSTAND ──▶ PLAN ─────────▶ IMPLEMENT ─────▶ REVIEW ─────────▶ (DEBUG) ─▶ FINISH
 grilling      make-plan       execute-plan     code-review        diagnose-  handoff
 grill-with-   wayfinder       + tdd            (verify-then-fix)   bug
 docs          (big work)
```

- Building something new? Start with `grilling` / `grill-with-docs`, then `make-plan`.
- Work too big for one session? `wayfinder` first (a decision map), then `make-plan` per piece.
- Executing a plan file? `execute-plan` (one task per pass) with `tdd`.
- Reviewing changes? `code-review`. A bug or failing test? `diagnose-bug`.
- (Claude only) exploratory design brainstorming and creating/editing skills are handled by the
  environment's `superpowers:brainstorming` and `skill-creator` — not duplicated here.

## 7. Skill index

Canonical skills live in `.claude/skills/` (Claude Code discovers them automatically). Other tools
read the identical mirror at `agents/skills/<name>/SKILL.md`.

| Skill | When to use |
|---|---|
| `grilling` | Stress-test a plan/decision/idea; relentless one-question-at-a-time interview |
| `grill-with-docs` | Sharpen a design and record decisions as ADRs (`docs/adr/`) + glossary |
| `wayfinder` | Plan work too large for one session as a local decision map (`docs/plans/maps/`) |
| `make-plan` | Turn a spec/change into an approved ralphex-format plan (`docs/plans/`) |
| `execute-plan` | Drive a plan file one task per pass: implement → test → validate → commit |
| `tdd` | Red-green-refactor with this repo's Go test conventions |
| `code-review` | Multi-reviewer review, verify every finding at file:line, fix confirmed |
| `diagnose-bug` | Reproduce → minimize → hypothesize → instrument → fix → verify |
| `writing-style` | Strip AI-speak from commits/PRs/review comments |
| `handoff` | Compact handoff doc for continuation |
| `add-middleware` | Scaffold a chi middleware in `internal/server/middleware` |
| `add-endpoint` | OpenAPI-first endpoint: spec → `make gen-oapi` → handler → route |
| `add-migration` | Scaffold a goose migration in `migrations/` |
| `add-sqlc-query` | Write SQL in `sql/queries/` → `make sqlc` → typed code |

## 8. Review engine

`code-review` runs Go reviewers (`.claude/skills/code-review/reviewers/*.md`) — as read-only
parallel subagents on Claude (`.claude/agents/go-*.md`), or sequentially elsewhere. Two modes:
**comprehensive** (all 6: quality, smells, simplification, testing, implementation, documentation)
and **critical-only** (quality + implementation). Every finding is **verified at file:line**
(CONFIRMED / FALSE POSITIVE) before any fix. Severity **critical/major/minor**; over-engineering
effort **trivial/small/medium/large**.

## 9. Editing skills

Edit skills in **`.claude/skills/` only**, then run `make sync-skills` to regenerate the
`agents/skills/` mirror. Never hand-edit `agents/skills/`.
