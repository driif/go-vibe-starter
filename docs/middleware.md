# Middleware Configuration

This document explains every middleware layer in the stack: what it does, when to turn it off, and how to tune it.

Middleware is registered in `internal/server/server.go` → `Initialize()`, in this order:

1. StripSlashes
2. Recoverer
3. Secure
4. RequestID
5. Logger
6. CORS
7. CacheControl

---

## 1. StripSlashes

**Package:** `go-chi/chi/v5/middleware`
**Toggle:** `SERVER_ENABLE_TRAILING_SLASH_MIDDLEWARE` (default: `true`)

Strips trailing slashes from request URLs before routing.
`GET /users/` becomes `GET /users`.

**When to disable:** Never, unless you have routes that intentionally differ on the trailing slash.

---

## 2. Recoverer

**Package:** `go-chi/chi/v5/middleware`
**Toggle:** `SERVER_ENABLE_RECOVER_MIDDLEWARE` (default: `true`)

Catches panics in handlers and returns `500 Internal Server Error` instead of crashing the server.

**When to disable:** Only in tests where you want panics to propagate (set `SERVER_ENABLE_RECOVER_MIDDLEWARE=false` in test config).

---

## 3. Secure

**Package:** `internal/server/middleware`
**Toggle:** `SERVER_ENABLE_SECURE_MIDDLEWARE` (default: `true`)

Sets security-related HTTP response headers.

### Configuration

All fields are set via env vars. An empty string or zero value means the header is not sent.

| Header | Env var | Dev default | Prod recommendation |
|---|---|---|---|
| `X-XSS-Protection` | `SERVER_SECURE_XSS_PROTECTION` | `1; mode=block` | `1; mode=block` |
| `X-Content-Type-Options` | `SERVER_SECURE_CONTENT_TYPE_NOSNIFF` | `nosniff` | `nosniff` |
| `X-Frame-Options` | `SERVER_SECURE_X_FRAME_OPTIONS` | `SAMEORIGIN` | `DENY` |
| `Strict-Transport-Security` | `SERVER_SECURE_HSTS_MAX_AGE` | `0` (off) | `31536000` |
| `Content-Security-Policy` | `SERVER_SECURE_CONTENT_SECURITY_POLICY` | `""` (off) | `default-src 'self'` |
| `Referrer-Policy` | `SERVER_SECURE_REFERRER_POLICY` | `""` (off) | `strict-origin-when-cross-origin` |

### HSTS

HSTS (`Strict-Transport-Security`) tells browsers to only connect over HTTPS.
Configure it once TLS is stable — it cannot be easily undone.

```env
SERVER_SECURE_HSTS_MAX_AGE=31536000           # 1 year in seconds
SERVER_SECURE_HSTS_EXCLUDE_SUBDOMAINS=false   # false = includeSubDomains
SERVER_SECURE_HSTS_PRELOAD_ENABLED=false      # only set true for HSTS preload list submission
```

### CSP

Start in report-only mode to audit violations before enforcing:

```env
SERVER_SECURE_CONTENT_SECURITY_POLICY=default-src 'self'
SERVER_SECURE_CSP_REPORT_ONLY=true    # observe violations first
```

Then switch to enforcing:

```env
SERVER_SECURE_CSP_REPORT_ONLY=false
```

---

## 4. RequestID

**Package:** `go-chi/chi/v5/middleware`
**Toggle:** `SERVER_ENABLE_REQUEST_ID_MIDDLEWARE` (default: `true`)

Adds `X-Request-Id` to the response. If the request already has an `X-Request-Id` header (set by a proxy), it is preserved.

The logger middleware reads this value and includes it in every log line as `id`.

**When to disable:** Never in production. Useful for correlating logs to specific requests.

---

## 5. Logger

**Package:** `internal/server/middleware`
**Toggle:** `SERVER_ENABLE_LOGGER_MIDDLEWARE` (default: `true`)

Structured `slog` logger that emits two log events per request.

### Default behavior (no extra config needed)

Every request logs:
```
level=DEBUG msg="request received" id=abc123 host=localhost method=GET url=/users bytes_in=0
level=DEBUG msg="response sent" status=200 bytes_out=512 duration_ms=1.23ms
```

### Tuning for development

Enable more detail when debugging:

```env
SERVER_LOGGER_LOG_REQUEST_BODY=true     # log JSON request bodies
SERVER_LOGGER_LOG_REQUEST_HEADER=true   # log request headers (Authorization redacted)
SERVER_LOGGER_LOG_REQUEST_QUERY=true    # log URL query params
SERVER_LOGGER_LOG_RESPONSE_BODY=true    # log JSON response bodies
```

### Tuning for production

```env
SERVER_LOGGER_LEVEL=INFO
SERVER_LOGGER_REQUEST_LEVEL=INFO

# Disable all body/header logging to avoid logging PII
SERVER_LOGGER_LOG_REQUEST_BODY=false
SERVER_LOGGER_LOG_REQUEST_HEADER=false
SERVER_LOGGER_LOG_REQUEST_QUERY=false
SERVER_LOGGER_LOG_RESPONSE_BODY=false
SERVER_LOGGER_LOG_RESPONSE_HEADER=false
```

### Sensitive header redaction

The following headers are always redacted before logging, replaced with `*****REDACTED*****`:
- `Authorization`
- `X-CSRF-Token`
- `Proxy-Authorization`

### Skipping specific routes

To skip logging for a route (e.g., health checks), use a custom skipper when wiring the middleware:

```go
r.Use(srvmiddleware.LoggerWithConfig(srvmiddleware.LoggerConfig{
    Skipper: func(r *http.Request) bool {
        return r.URL.Path == "/healthz"
    },
}))
```

---

## 6. CORS

**Package:** `go-chi/cors`
**Toggle:** `SERVER_ENABLE_CORS_MIDDLEWARE` (default: `true`)

Currently wired as `cors.AllowAll()` which permits any origin. This is fine for development and APIs consumed only by trusted clients.

### Production: restrict to your frontend

In `server.go`, replace `cors.AllowAll().Handler` with:

```go
cors.New(cors.Options{
    AllowedOrigins:   []string{"https://your-app.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-Id"},
    AllowCredentials: true,
    MaxAge:           300,
}).Handler
```

---

## 7. CacheControl

**Package:** `internal/server/middleware`
**Toggle:** `SERVER_ENABLE_CACHE_CONTROL_MIDDLEWARE` (default: `true`)

Sets `Cache-Control: no-store` on every response. Prevents proxies and browsers from caching API responses.

### vs. NoCache middleware

`CacheControl` sets one header (`no-store`). It is appropriate for most API routes.

`NoCache` sets four headers (`Expires`, `Cache-Control`, `Pragma`, `X-Accel-Expires`) and also deletes ETag request headers. Use it for routes that must defeat aggressive proxy caching (legacy proxies, nginx proxy_cache, etc.).

To use `NoCache` on specific routes instead of the global `CacheControl`:

```go
// Remove global CacheControl, add NoCache per-group
r.Group(func(r chi.Router) {
    r.Use(srvmiddleware.NoCache)
    r.Get("/feed", handlers.Feed)
})
```

Or with a skipper:

```go
r.Use(srvmiddleware.NoCacheWithSkipper(func(r *http.Request) bool {
    // skip for static assets served from /static/
    return strings.HasPrefix(r.URL.Path, "/static/")
}))
```
