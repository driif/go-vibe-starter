# Middleware API Reference

Package: `github.com/driif/go-vibe-starter/internal/server/middleware`
Import alias used in `server.go`: `srvmiddleware`

All middleware follows the standard chi/net/http signature:
```go
func(http.Handler) http.Handler
```

---

## Noop

File: `middleware/noop.go`

Passes the request through unchanged. Use as a placeholder when toggling middleware on/off programmatically.

```go
func Noop() func(http.Handler) http.Handler
```

**Usage:**
```go
r.Use(srvmiddleware.Noop())
```

---

## NoCache

File: `middleware/no_cache.go`

Prevents caching by proxies and clients. Sets response headers and deletes ETag-related request headers.

```go
func NoCache(next http.Handler) http.Handler
func NoCacheWithSkipper(skip func(*http.Request) bool) func(http.Handler) http.Handler
```

**Response headers set:**

| Header | Value |
|---|---|
| `Expires` | `Thu, 01 Jan 1970 00:00:00 UTC` |
| `Cache-Control` | `no-cache, private, max-age=0` |
| `Pragma` | `no-cache` |
| `X-Accel-Expires` | `0` |

**Request headers deleted:**
`ETag`, `If-Modified-Since`, `If-Match`, `If-None-Match`, `If-Range`, `If-Unmodified-Since`

**Usage:**
```go
// Default — no skipper
r.Use(srvmiddleware.NoCache)

// With skipper — skip health check endpoint
r.Use(srvmiddleware.NoCacheWithSkipper(func(r *http.Request) bool {
    return r.URL.Path == "/healthz"
}))
```

---

## CacheControl

File: `middleware/cache_control.go`

Sets `Cache-Control: no-store` on every response. Simpler than `NoCache` — use this for API routes where you want no caching at all.

```go
func CacheControl(next http.Handler) http.Handler
```

**Response headers set:**

| Header | Value |
|---|---|
| `Cache-Control` | `no-store` |

**Usage:**
```go
r.Use(srvmiddleware.CacheControl)
```

Enabled by default. Toggle: `SERVER_ENABLE_CACHE_CONTROL_MIDDLEWARE=false`

---

## Secure

File: `middleware/secure.go`

Sets security-related response headers. Controlled by `SecureConfig`.

```go
type SecureConfig struct {
    XSSProtection         string
    ContentTypeNosniff    string
    XFrameOptions         string
    HSTSMaxAge            int
    HSTSExcludeSubdomains bool
    ContentSecurityPolicy string
    CSPReportOnly         bool
    HSTSPreloadEnabled    bool
    ReferrerPolicy        string
}

func Secure(cfg SecureConfig) func(http.Handler) http.Handler
```

**Field → HTTP header mapping:**

| Field | HTTP Header | Default (from env) | Notes |
|---|---|---|---|
| `XSSProtection` | `X-XSS-Protection` | `1; mode=block` | Empty string = header not set |
| `ContentTypeNosniff` | `X-Content-Type-Options` | `nosniff` | Empty string = header not set |
| `XFrameOptions` | `X-Frame-Options` | `SAMEORIGIN` | `DENY` or `SAMEORIGIN` |
| `HSTSMaxAge` | `Strict-Transport-Security` | `0` (disabled) | `0` = header not set |
| `HSTSExcludeSubdomains` | (modifier) | `false` | If false, adds `; includeSubDomains` |
| `HSTSPreloadEnabled` | (modifier) | `false` | If true, adds `; preload` |
| `ContentSecurityPolicy` | `Content-Security-Policy` | `""` (disabled) | Empty string = header not set |
| `CSPReportOnly` | (modifier) | `false` | If true, uses `Content-Security-Policy-Report-Only` |
| `ReferrerPolicy` | `Referrer-Policy` | `""` (disabled) | Empty string = header not set |

**Usage in server.go:**
```go
sc := s.Config.Server.SecureMiddleware
r.Use(srvmiddleware.Secure(srvmiddleware.SecureConfig{
    XSSProtection:         sc.XSSProtection,
    ContentTypeNosniff:    sc.ContentTypeNosniff,
    XFrameOptions:         sc.XFrameOptions,
    HSTSMaxAge:            sc.HSTSMaxAge,
    HSTSExcludeSubdomains: sc.HSTSExcludeSubdomains,
    ContentSecurityPolicy: sc.ContentSecurityPolicy,
    CSPReportOnly:         sc.CSPReportOnly,
    HSTSPreloadEnabled:    sc.HSTSPreloadEnabled,
    ReferrerPolicy:        sc.ReferrerPolicy,
}))
```

Toggle: `SERVER_ENABLE_SECURE_MIDDLEWARE=false`

---

## Logger

File: `middleware/logger.go`

Structured request/response logger using `log/slog`. Logs two events per request:
- `"request received"` — before the handler runs
- `"response sent"` — after the handler returns

```go
func Logger() func(http.Handler) http.Handler
func LoggerWithConfig(cfg LoggerConfig) func(http.Handler) http.Handler
```

### LoggerConfig

```go
type LoggerConfig struct {
    Level                     slog.Level
    LogRequestBody            bool
    LogRequestHeader          bool
    LogRequestQuery           bool
    LogResponseBody           bool
    LogResponseHeader         bool
    Skipper                   func(*http.Request) bool
    RequestBodyLogSkipper     RequestBodyLogSkipper     // func(*http.Request) bool
    RequestHeaderLogReplacer  HeaderLogReplacer          // func(http.Header) http.Header
    ResponseBodyLogSkipper    ResponseBodyLogSkipper    // func(*http.Request, int) bool
    ResponseHeaderLogReplacer HeaderLogReplacer
}
```

**Fields:**

| Field | Type | Default | Behavior |
|---|---|---|---|
| `Level` | `slog.Level` | `slog.LevelDebug` | Log level for both events |
| `LogRequestBody` | `bool` | `false` | Read + log request body (restored for handler) |
| `LogRequestHeader` | `bool` | `false` | Log sanitized request headers |
| `LogRequestQuery` | `bool` | `false` | Log URL query parameters |
| `LogResponseBody` | `bool` | `false` | Log response body (JSON only by default) |
| `LogResponseHeader` | `bool` | `false` | Log sanitized response headers |
| `Skipper` | `func(*http.Request) bool` | `nil` | Return `true` to skip logging for a request |
| `RequestBodyLogSkipper` | `RequestBodyLogSkipper` | `DefaultRequestBodyLogSkipper` | Skip body logging for form/multipart |
| `RequestHeaderLogReplacer` | `HeaderLogReplacer` | `DefaultHeaderLogReplacer` | Redact sensitive headers |
| `ResponseBodyLogSkipper` | `ResponseBodyLogSkipper` | `DefaultResponseBodyLogSkipper` | Only log JSON response bodies |
| `ResponseHeaderLogReplacer` | `HeaderLogReplacer` | `DefaultHeaderLogReplacer` | Redact sensitive response headers |

### slog attributes emitted

**"request received" event:**

| Attribute | Source |
|---|---|
| `id` | `X-Request-Id` header |
| `host` | `r.Host` |
| `method` | `r.Method` |
| `url` | `r.URL.String()` |
| `bytes_in` | `Content-Length` header |
| `req_body` | Request body (if `LogRequestBody=true`) |
| `req_header` | Sanitized headers (if `LogRequestHeader=true`) |
| `req_query` | URL query (if `LogRequestQuery=true`) |

**"response sent" event:**

| Attribute | Source |
|---|---|
| `status` | HTTP status code |
| `bytes_out` | Bytes written to response |
| `duration_ms` | Handler execution time |
| `res_body` | Response body (if `LogResponseBody=true` and content is JSON) |
| `res_header` | Sanitized response headers (if `LogResponseHeader=true`) |

### Default skippers

**`DefaultRequestBodyLogSkipper`** — returns `true` (skip) for:
- `Content-Type: application/x-www-form-urlencoded`
- `Content-Type: multipart/form-data`

**`DefaultResponseBodyLogSkipper`** — returns `false` (do not skip) always; the middleware additionally checks `Content-Type: application/json` before logging the response body.

**`DefaultHeaderLogReplacer`** — replaces values with `*****REDACTED*****` for:
- `Authorization`
- `X-CSRF-Token`
- `Proxy-Authorization`

### Env vars that map to LoggerConfig

These are read in `DefaultServiceConfigFromEnv()` and passed to `LoggerWithConfig` in `server.Initialize()`:

| Env var | LoggerConfig field |
|---|---|
| `SERVER_LOGGER_REQUEST_LEVEL` | `Level` |
| `SERVER_LOGGER_LOG_REQUEST_BODY` | `LogRequestBody` |
| `SERVER_LOGGER_LOG_REQUEST_HEADER` | `LogRequestHeader` |
| `SERVER_LOGGER_LOG_REQUEST_QUERY` | `LogRequestQuery` |
| `SERVER_LOGGER_LOG_RESPONSE_BODY` | `LogResponseBody` |
| `SERVER_LOGGER_LOG_RESPONSE_HEADER` | `LogResponseHeader` |

Toggle: `SERVER_ENABLE_LOGGER_MIDDLEWARE=false`
