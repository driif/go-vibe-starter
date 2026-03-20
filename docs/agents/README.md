# Agent Context: go-vibe-starter

This file is the primary orientation for LLM agents working in this repository.
Read this before making any changes.

---

## Module

```
github.com/driif/go-vibe-starter
```

Go version: **1.25**

---

## Directory layout

```
.
├── cmd/                         CLI commands (Cobra)
│   ├── root.go                  Root command, global flags
│   ├── run.go                   `app run` — starts the HTTP server
│   ├── db.go                    `app db migrate` and related
│   └── db_seed.go               `app db seed`
│
├── internal/
│   └── server/
│       ├── server.go            Server struct, Initialize(), Start(), Shutdown()
│       ├── config/
│       │   ├── server_config.go App config struct + DefaultServiceConfigFromEnv()
│       │   ├── db_config.go     Database config + ConnectionString()
│       │   └── env/             Low-level env helpers (GetEnv, GetEnvAsBool, etc.)
│       └── middleware/          All custom chi middleware (net/http, slog)
│           ├── noop.go
│           ├── no_cache.go
│           ├── cache_control.go
│           ├── secure.go
│           └── logger.go
│
├── pkg/
│   ├── dotenv/                  .env file loader
│   ├── oidc/                    Generic OIDC/JWT utilities
│   ├── keycloak/                Keycloak-specific JWT validation
│   └── tests/                  Test helpers (RunningInTest())
│
├── main.go                      Entry point — executes root Cobra command
├── Makefile                     make build / make run / make clean
└── docker-compose.yaml          Local postgres + keycloak
```

---

## Key files and what they own

| File | Owns |
|---|---|
| `internal/server/server.go` | Router init, middleware stack order, pprof mount |
| `internal/server/config/server_config.go` | ALL config structs + env var mappings |
| `internal/server/middleware/logger.go` | Structured slog request/response logger |
| `internal/server/middleware/secure.go` | Security headers middleware |
| `cmd/run.go` | Server startup, config loading, graceful shutdown |

---

## Config system

All configuration comes from environment variables. No config files, no flags (except `--help`).

Flow:
```
env vars
  → internal/server/config/server_config.go DefaultServiceConfigFromEnv()
  → config.App struct
  → server.NewWithConfig(config)
  → server.Initialize() reads fields and wires middleware
```

Accessing config in a handler: pass what you need explicitly. Do not use global state.

The full env var reference is in [`env-reference.md`](./env-reference.md).

---

## Middleware stack (in order)

Registered in `server.Initialize()`:

1. `chimiddleware.StripSlashes` — strips trailing slashes from URL
2. `chimiddleware.Recoverer` — catches panics, returns 500
3. `srvmiddleware.Secure(cfg)` — sets security headers (X-Frame-Options, HSTS, CSP, etc.)
4. `chimiddleware.RequestID` — adds X-Request-Id header if absent
5. `srvmiddleware.LoggerWithConfig(cfg)` — structured slog request/response logging
6. `cors.AllowAll().Handler` — CORS headers (replace in production)
7. `srvmiddleware.CacheControl` — sets `Cache-Control: no-store`

Each can be toggled via its `SERVER_ENABLE_*_MIDDLEWARE` env var.
The full middleware API is in [`middleware-api.md`](./middleware-api.md).

---

## Adding a route

1. Create a handler file: `internal/api/<domain>/<handler>.go`
2. Handler signature: `func(w http.ResponseWriter, r *http.Request)`
3. Mount in `server.Initialize()` or in a separate `mountRoutes()` method:

```go
r.Get("/users", handlers.ListUsers)
r.Route("/users", func(r chi.Router) {
    r.Get("/", handlers.ListUsers)
    r.Post("/", handlers.CreateUser)
    r.Get("/{id}", handlers.GetUser)
})
```

---

## Database pattern

No ORM. Use SQLC:

1. Write SQL in `sql/queries/<domain>.sql`
2. Run `sqlc generate`
3. Use the generated Go code from `db/` (or wherever sqlc output is configured)

Access DB via `server.DB *sql.DB` passed to handlers.

---

## Logging

Always use `log/slog`. Never use `fmt.Println` or zerolog or logrus.

```go
slog.Info("user created", "user_id", id)
slog.Error("failed to query", "error", err)
slog.Warn("cache miss", "key", k)
```

The logger middleware adds request context (method, url, status, duration) automatically.

---

## Error handling

Return errors up the call stack. Wrap with context only when it helps debugging:

```go
// OK
return err

// OK when context helps
return fmt.Errorf("create user: %w", err)

// Too much
return fmt.Errorf("layer1: %w", fmt.Errorf("layer2: %w", err))
```

---

## Rules for agents

**DO:**
- Follow patterns already in the codebase
- Use stdlib before adding a dependency
- Write minimal code — only what the task requires
- Use `slog` for all logging
- Use chi middleware signature: `func(http.Handler) http.Handler`
- Read existing files before proposing changes

**DO NOT:**
- Import `github.com/labstack/echo` — this project uses chi
- Import `github.com/rs/zerolog` — this project uses slog
- Add an ORM — use SQLC + raw SQL
- Create interfaces for single implementations
- Add error wrapping layers without clear benefit
- Run `make build` or `make run` unless asked
- Create files not needed for the task
