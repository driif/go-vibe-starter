---
name: add-middleware
description: Use when adding a custom chi middleware — scaffolds internal/server/middleware and shows registration in server.go.
---

# Add middleware

Ask for the middleware name (e.g. `RateLimiter`, `Tenant`) and a one-line description of what it
does, if not already given. Then:

## 1. Create the file

`internal/server/middleware/<snake_case_name>.go`. Use the config form when it needs options:

```go
package middleware

import "net/http"

// <Name>Config configures the <Name> middleware.
type <Name>Config struct {
	// options
}

// <Name> returns the middleware with default settings.
func <Name>(next http.Handler) http.Handler {
	return <Name>WithConfig(<Name>Config{})(next)
}

// <Name>WithConfig returns the middleware with the given config.
func <Name>WithConfig(cfg <Name>Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// logic here
			next.ServeHTTP(w, r)
		})
	}
}
```

If no config is needed, use the single-function form:

```go
func <Name>(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// logic here
		next.ServeHTTP(w, r)
	})
}
```

## 2. Register it

In `internal/server/server.go` → `Initialize()`, after the existing middleware, in the correct
order:

```go
r.Use(srvmiddleware.<Name>)
// or: r.Use(srvmiddleware.<Name>WithConfig(srvmiddleware.<Name>Config{...}))
```

## 3. Optional env toggle

If it should be switchable, add to `internal/server/config/server_config.go`:

```go
Enable<Name>Middleware bool
// in DefaultServiceConfigFromEnv():
Enable<Name>Middleware: env.GetEnvAsBool("SERVER_ENABLE_<NAME>_MIDDLEWARE", true),
```

and guard registration in `Initialize()`.

## Rules

- Signature is always `func(http.Handler) http.Handler`.
- `log/slog` only — never fmt.Println or another logger.
- Don't import echo or any other HTTP framework.
- Storing values in context: use a typed key, not a plain string, to avoid collisions.
- Keep it minimal — implement only what was asked.
