# Go Vibe Starter - Project Guidelines for Claude Code

## Project Context
Go Vibe Starter is a production-ready Go web server foundation for rapid prototyping with Claude Code. The focus is on explicit, understandable infrastructure with minimal boilerplate.

### Tech Stack
- **Language**: Go 1.25
- **CLI Framework**: Cobra
- **RPC**: Connect RPC (protobufs as source of truth)
- **Authentication**: Keycloak (OIDC/JWT)
- **Database**: PostgreSQL
- **SQL**: SQLC (no ORM - SQL is explicit)
- **Linting**: golangci-lint

### Project Philosophy
This project is designed for LLM-first development:
- SQL over ORMs - LLMs write SQL well, ORMs add unnecessary abstraction
- Explicit code over magic - clarity for both humans and LLMs
- Minimal boilerplate - focus on business logic

## Core Development Principles

### Code Philosophy: Minimal & Pragmatic
**CRITICAL**: Always write minimal, pragmatic code that solves the problem at hand.
- **DO**: Write the simplest code that works correctly
- **DO**: Add complexity only when explicitly needed
- **DO**: Favor straightforward solutions over clever ones

### Command Execution
**Use Makefile for all build/run operations** - Only execute commands when explicitly asked.
- **DON'T**: Run `make build`, `make run`, linters, or tests unless explicitly requested
- **DO**: Use Makefile targets when asked to build or run
- **DO**: Assume the user runs checks automatically

### Available Makefile Targets
```bash
make build    # Build the binary to bin/app
make run      # Build and run the server
```

## Project Structure

```
.
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command setup
│   ├── run.go             # Server run command
│   ├── db.go              # Database commands
│   └── db_seed.go         # Database seeding
├── internal/
│   └── server/
│       ├── server.go      # Connect RPC server setup
│       └── config/        # Configuration from env
├── pkg/
│   ├── oidc/              # Generic OIDC utilities
│   └── keycloak/          # Keycloak-specific integration
├── main.go                # Entry point
├── Makefile               # Build targets
└── docker-compose.yaml    # Local development services
```

### Directory Conventions
- `cmd/` - CLI commands only, minimal logic
- `internal/` - Private application code
- `pkg/` - Reusable packages (can be imported by other projects)

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENVIRONMENT` | `development` | Environment name |
| `SERVICE_PORT` | `:8443` | Connect RPC server port |
| `KEYCLOAK_URL` | `http://localhost:8080` | Keycloak base URL |
| `KEYCLOAK_REALM` | `myrealm` | Keycloak realm |
| `KEYCLOAK_CLIENT_ID` | `myclient` | Keycloak client ID |

## Go Best Practices

### Error Handling
```go
// Keep it simple - return errors, don't wrap excessively
if err != nil {
    return err
}

// Add context only when it helps debugging
if err != nil {
    return fmt.Errorf("failed to connect to database: %w", err)
}
```

### Logging
Use `log/slog` for structured logging:
```go
slog.Info("Starting server", "port", port)
slog.Error("Failed to connect", "error", err)
```

### Interfaces
- Define interfaces where they're used, not where they're implemented
- Keep interfaces small (1-3 methods)
- Don't create interfaces for single implementations

## Database Patterns (SQLC)

### Why No ORM
- SQL is explicit and LLMs write it well
- SQLC generates type-safe Go code from SQL
- No magic, no hidden queries

### SQLC Workflow
1. Write SQL queries in `sql/queries/`
2. Run `sqlc generate`
3. Use generated Go code

## Authentication

### Keycloak Integration
- JWT tokens validated via OIDC
- Organizations/tenants supported via Keycloak organizations
- Token claims extracted via `pkg/keycloak`

### Local Development
Keycloak runs via docker-compose on `http://localhost:8080`

## Common Patterns

### DO: Keep It Simple
```go
// Simple config loading
cfg := config.DefaultServiceConfigFromEnv()

// Direct error returns
if err != nil {
    return err
}
```

### DON'T: Over-Engineer
```go
// Don't create elaborate abstractions
// ConfigLoader interface with multiple implementations

// Don't wrap errors excessively
// fmt.Errorf("layer1: %w", fmt.Errorf("layer2: %w", err))

// Don't add unused flexibility
// NewServerWithOptions(opts ...ServerOption)
```

## Testing
- Tests should be added when explicitly requested
- Keep tests simple and focused
- Use table-driven tests for multiple cases
- Don't mock unnecessarily - test real behavior when practical

## Questions & Ambiguity
- **Ask First**: If requirements are unclear, ask before implementing
- **Prefer Simple**: When multiple approaches exist, choose the simpler one
- **Check Existing Code**: Follow patterns already established in the codebase

## Helpful Reminders
1. **No automatic command execution** - User handles builds, tests, linting
2. **Use Makefile** - When asked to run something, use make targets
3. **Minimal code** - Only write what's necessary
4. **Standard library first** - Use stdlib before adding dependencies
5. **SQL over ORM** - Write explicit SQL, generate with SQLC
6. **Environment config** - All config via env vars
7. **Ask when unsure** - Better to clarify than over-engineer

## Endpoints
- **Connect RPC**: `http://localhost:8443` (default)
- **Keycloak**: `http://localhost:8080`

---

**Remember**: Pragmatic, minimal, idiomatic Go code that solves the problem at hand. Nothing more, nothing less.
