Add a new custom middleware to this project.

Ask the user for the following if not already provided:
1. Middleware name (e.g. `RateLimiter`, `Auth`, `Tenant`)
2. What it should do (brief description)

Then:

1. Create `internal/server/middleware/<snake_case_name>.go` with this structure:

```go
package middleware

import "net/http"

// <Name>Config configures the <Name> middleware.
type <Name>Config struct {
    // TODO: add config fields
}

// <Name> returns the middleware with default settings.
func <Name>(next http.Handler) http.Handler {
    return <Name>WithConfig(<Name>Config{})(next)
}

// <Name>WithConfig returns the middleware with the given config.
func <Name>WithConfig(cfg <Name>Config) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // TODO: implement middleware logic

            next.ServeHTTP(w, r)
        })
    }
}
```

If no config is needed, simplify to a single function:

```go
func <Name>(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // TODO: implement middleware logic
        next.ServeHTTP(w, r)
    })
}
```

2. Show the user where to register it in `internal/server/server.go` → `Initialize()`:

```go
// Add after the existing middleware registrations, in the correct order:
r.Use(srvmiddleware.<Name>)
// or with config:
r.Use(srvmiddleware.<Name>WithConfig(srvmiddleware.<Name>Config{...}))
```

3. If the middleware needs a toggle, show the pattern to add to `server_config.go`:

```go
// In Server struct:
Enable<Name>Middleware bool

// In DefaultServiceConfigFromEnv():
Enable<Name>Middleware: env.GetEnvAsBool("SERVER_ENABLE_<NAME>_MIDDLEWARE", true),
```

And in `Initialize()`:
```go
if s.Config.Server.Enable<Name>Middleware {
    r.Use(srvmiddleware.<Name>)
} else {
    slog.Warn("<name> middleware disabled")
}
```

Rules to follow:
- Middleware signature is always `func(http.Handler) http.Handler`
- Use `log/slog` — never zerolog or fmt.Println
- Do not import echo or any other HTTP framework
- Keep it minimal — only implement what was asked
- If storing values in context, define a typed key (not a plain string) to avoid collisions
