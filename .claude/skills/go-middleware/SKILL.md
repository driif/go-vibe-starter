---
name: go-middleware
description: Adding or changing a chi middleware in this repo. Use when a concern has to run on every request before the handler — rate limiting, tenant resolution, security or cache headers, request-scoped logging or context values — or when a ticket says "add middleware X".
---

A middleware here is a `func(http.Handler) http.Handler` living in
`internal/server/middleware/<snake_name>.go`, registered in `internal/server/server.go` behind an
`Enable*Middleware` config flag. Worked examples, smallest first:
`cache_control.go` · `no_cache.go` · `secure.go` · `logger.go`.

Before writing one, check the concern is global. A rule for one domain belongs on that domain's
route group in `internal/api/router/routes_<domain>.go` (`r.Use(...)` inside the group, the way
`s.Authenticator()` is applied), and never reaches the other domains.

## 1. Write it

`internal/server/middleware/<snake_name>.go`, `package middleware`. Take the plain form when there
is nothing to configure:

```go
// CacheControl sets Cache-Control: no-store on every response.
func CacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
```

Take the Config form the moment it has options — a `Config` struct, a constructor that fills its
defaults, and one exported default entry point, as `logger.go` and `secure.go` do:

```go
// RateLimitConfig configures the RateLimit middleware.
type RateLimitConfig struct {
	RequestsPerMinute int
	Skipper           func(*http.Request) bool
}

// RateLimit returns the middleware with DefaultRateLimitConfig.
func RateLimit() func(http.Handler) http.Handler {
	return RateLimitWithConfig(DefaultRateLimitConfig)
}

// RateLimitWithConfig returns the middleware with the given config.
func RateLimitWithConfig(cfg RateLimitConfig) func(http.Handler) http.Handler {
	if cfg.RequestsPerMinute <= 0 {
		cfg.RequestsPerMinute = 60
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skipper != nil && cfg.Skipper(r) {
				next.ServeHTTP(w, r)
				return
			}
			// decide, then either write the rejection and return, or continue
			next.ServeHTTP(w, r)
		})
	}
}
```

Build everything that outlives a request once, in the constructor, outside the returned closure —
it is shared by every concurrent request, so keep it read-only there (`auth.DevPrincipal` builds its
principal this way).

Reading the response needs a wrapper: `chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)` gives
`Status()`, `BytesWritten()` and `Tee()`, as `logger.go` shows.

Passing a value down the chain uses a typed context key and an exported reader, never a bare string
key — `internal/server/auth/context.go` is the pattern to copy.

Reject with `errs.Write(w, status, err)` so the body matches every other error in the API. Log with
`log/slog`.

**Done when** `go build ./...` is clean and the middleware calls `next.ServeHTTP` on every path it
does not deliberately terminate.

## 2. Register it

Add the flag to `Server` in `internal/server/config/server_config.go` and read it in
`DefaultServiceConfigFromEnv`:

```go
EnableRateLimitMiddleware:  env.GetEnvAsBool("SERVER_ENABLE_RATE_LIMIT_MIDDLEWARE", true),
RateLimitRequestsPerMinute: env.GetEnvAsInt("SERVER_RATE_LIMIT_PER_MINUTE", 60),
```

Then guard registration in `Initialize()` in `internal/server/server.go`, matching the surrounding
idiom — a `slog.Warn` on the disabled branch, so a stripped-down server says so at startup:

```go
if s.Config.Server.EnableRateLimitMiddleware {
	r.Use(srvmiddleware.RateLimitWithConfig(srvmiddleware.RateLimitConfig{
		RequestsPerMinute: s.Config.Server.RateLimitRequestsPerMinute,
	}))
} else {
	slog.Warn("rate limit middleware disabled")
}
```

Registration order is execution order. The current chain is StripSlashes → Recoverer → Secure →
RequestID → Logger → CORS → CacheControl. Place a new one by what it needs: after `Recoverer` to be
covered by it, after `RequestID` to log the id, before `Logger` to have its effect logged.

Document the new variable in `.env.example` and `docs/agents/env-reference.md` with its default.

**Done when** `SERVER_ENABLE_<NAME>_MIDDLEWARE=false` starts the server without the middleware and
logs the warning, and the same variable appears in `.env.example`.

## 3. Test it

`internal/server/middleware/<snake_name>_test.go`. Wrap a handler that records what reached it,
drive it with `httptest.NewRequest` and `httptest.NewRecorder`, and assert both directions — what
the middleware did to the response, and what it passed down. `internal/server/auth/dev_test.go` is
the shape to copy.

**Done when** `make test` and `make lint` both exit 0, with a case for the pass-through path, the
terminating path, and the skipper or disabled path if the middleware has one.
