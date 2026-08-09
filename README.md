# Go Vibe Starter

A GitHub **template** for starting a Go backend fast, with an agent skill pack that knows how to
build features in it.

Click **Use this template**, run `make init`, and you have a running HTTP server with config, auth,
migrations, type-safe SQL, an OpenAPI-generated API layer, CI, and a set of skills that turn an idea
into shipped code. The point is to spend your time on business logic, not on the fifteenth
re-implementation of graceful shutdown.

---

## Quickstart

```bash
# 1. Create a repo from this template, then clone it
git clone https://github.com/<you>/<your-project> && cd <your-project>

# 2. Rename the module, name the app, and strip the demo slice
make init

# 3. Start Postgres (and Keycloak, if you want it)
make db-up

# 4. Run
make run
```

The server listens on `:9880` by default. Check it:

```bash
curl localhost:9880/health   # {"status":"ok","version":"..."}
curl localhost:9880/ready    # 200 once the database answers
```

`AUTH_PROVIDER` defaults to `none`, so a fresh clone boots with no identity provider and every
request runs as a local dev principal. Set it to `keycloak` when you want real tokens. Starting in
production with auth disabled is refused.

**Prerequisites:** Go 1.25+, Docker, and `make tools` for the pinned code generators.

---

## Why this stack

**No ORM.** ORMs add abstraction that hurts both humans and LLMs. SQL is explicit and models write it
well, so schema lives in [goose](https://github.com/pressly/goose) migrations and queries are
hand-written and compiled to type-safe Go by [sqlc](https://sqlc.dev).

**chi for routing.** ~1000 lines, fully `net/http`-compatible, no magic. Middleware is just
`func(http.Handler) http.Handler`.

**OpenAPI-first.** The spec is the source of truth; `oapi-codegen` produces the types and the server
interface. Your clients get a contract for free.

**PostgreSQL.** Reliable, flexible, and `pgvector` is there when you need embeddings.

**Keycloak, optionally.** Real user management with organizations and tenants when you need it,
entirely out of the way when you don't.

**LLM-first.** Explicit over clever, SQL over abstraction, one worked example in the repo for agents
to copy, and a skill pack that encodes how work actually flows.

---

## How you build a feature

Every feature is a **vertical slice** — schema to HTTP, in one pass. The repo ships one as a worked
example (`notes`, deleted by `make init`):

```
migrations/0001_notes.sql             1. schema        goose
sql/queries/notes.sql                 2. queries       -> make sqlc
oapi/openapi.yaml                     3. API surface   -> make gen-oapi
internal/api/handlers/notes.go        4. handler
internal/api/handlers/notes_test.go   5. tests
internal/api/router/routes_notes.go   6. route
```

Ask an agent for a feature and the `go-feature` skill walks exactly this path. One
`routes_<domain>.go` per domain means adding a feature never edits a shared file.

---

## The AI workflow

Guidance for agents lives in **[`AGENTS.md`](AGENTS.md)** — the single source of truth for stack,
conventions, and workflow. `CLAUDE.md` imports it. The domain model lives in
[`CONTEXT.md`](CONTEXT.md).

Work moves through two substrates:

- **GitHub Issues hold intent** — `spec`, `ticket`, `decision`, `bug`, with `Blocked by #N` edges.
  `make labels` creates the vocabulary.
- **`docs/plans/*.md` holds execution** — a plan file in [ralphex](https://github.com/umputun/ralphex)
  format, run unattended and archived when done.

```
loose idea ─▶ /grill-me ─▶ /to-spec ─▶ /to-tickets ─┬─▶ /implement            (you're watching)
                                                     └─▶ /to-plan + ralphex   (walk away)
```

24 skills ship in `.claude/skills/`: twelve you invoke by name (the flow above, plus `/triage`,
`/wayfinder`, `/handoff`, `/improve-architecture`, `/setup-project`) and twelve the agent reaches for
on its own (`go-feature`, `tdd`, `code-review`, `diagnosing-bugs`, `codebase-design`, and friends).
Not sure which? Run **`/vibe`** — it's the map.

Skills are portable: `.claude/skills/` is canonical, `agents/skills/` is a mirror for Codex, Pi, and
opencode. Edit the canonical copy, then `make sync-skills`.

Structure and writing discipline of the pack owe a lot to
[mattpocock/skills](https://github.com/mattpocock/skills); the review engine and plan format come
from [umputun/ralphex](https://github.com/umputun/ralphex). Both MIT.

---

## Make targets

```
make help         list every target
make init         bootstrap a new project from this template
make build/run    compile to bin/app / build and start
make test         go test -race ./...
make cover        coverage profile + summary
make lint         golangci-lint run
make fmt          gofmt + goimports
make gen          gen-oapi + sqlc
make migrate-up   apply pending migrations
make db-up/down   docker compose up -d / down
make labels       create the GitHub issue labels
make sync-skills  mirror .claude/skills -> agents/skills
make tools        install the pinned codegen and lint tools
```

## Documentation

- [`AGENTS.md`](AGENTS.md) — agent guidance, conventions, skill index
- [`CONTEXT.md`](CONTEXT.md) — ubiquitous language and module map
- [`docs/agents/go-style.md`](docs/agents/go-style.md) — the Go house style in depth
- [`docs/agents/env-reference.md`](docs/agents/env-reference.md) — every environment variable
- [`docs/middleware.md`](docs/middleware.md) — the middleware stack and how to tune it
- [`docs/production.md`](docs/production.md) — hardening checklist before going live
- [`docs/adr/`](docs/adr/) — architecture decision records

## License

MIT
