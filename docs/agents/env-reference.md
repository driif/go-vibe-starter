# Environment Variable Reference

All configuration is loaded in `internal/server/config/server_config.go` → `DefaultServiceConfigFromEnv()`.

No config files. No CLI flags for runtime config. Set these in your shell, `.env.local`, or container environment.

---

## Server

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `APP_ENVIRONMENT` | `App.Environment` | `string` | `development` | Environment name (used for logging context) |
| `SERVICE_PORT` | `Server.ListenAddr` | `string` | `:8080` | Listen address. Prefix `:` added automatically if missing |

---

## Middleware toggles

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `SERVER_ENABLE_CORS_MIDDLEWARE` | `Server.EnableCORSMiddleware` | `bool` | `true` | Enable CORS headers |
| `SERVER_ENABLE_LOGGER_MIDDLEWARE` | `Server.EnableLoggerMiddleware` | `bool` | `true` | Enable request/response logger |
| `SERVER_ENABLE_RECOVER_MIDDLEWARE` | `Server.EnableRecoverMiddleware` | `bool` | `true` | Enable panic recovery (returns 500) |
| `SERVER_ENABLE_REQUEST_ID_MIDDLEWARE` | `Server.EnableRequestIDMiddleware` | `bool` | `true` | Add `X-Request-Id` header |
| `SERVER_ENABLE_TRAILING_SLASH_MIDDLEWARE` | `Server.EnableTrailingSlashMiddleware` | `bool` | `true` | Strip trailing slashes from URLs |
| `SERVER_ENABLE_SECURE_MIDDLEWARE` | `Server.EnableSecureMiddleware` | `bool` | `true` | Set security headers |
| `SERVER_ENABLE_CACHE_CONTROL_MIDDLEWARE` | `Server.EnableCacheControlMiddleware` | `bool` | `true` | Set `Cache-Control: no-store` |

---

## Security headers (Secure middleware)

| Env var | Go field | Type | Default | HTTP header |
|---|---|---|---|---|
| `SERVER_SECURE_XSS_PROTECTION` | `SecureMiddleware.XSSProtection` | `string` | `1; mode=block` | `X-XSS-Protection` |
| `SERVER_SECURE_CONTENT_TYPE_NOSNIFF` | `SecureMiddleware.ContentTypeNosniff` | `string` | `nosniff` | `X-Content-Type-Options` |
| `SERVER_SECURE_X_FRAME_OPTIONS` | `SecureMiddleware.XFrameOptions` | `string` | `SAMEORIGIN` | `X-Frame-Options` |
| `SERVER_SECURE_HSTS_MAX_AGE` | `SecureMiddleware.HSTSMaxAge` | `int` | `0` | `Strict-Transport-Security` max-age (0 = disabled) |
| `SERVER_SECURE_HSTS_EXCLUDE_SUBDOMAINS` | `SecureMiddleware.HSTSExcludeSubdomains` | `bool` | `false` | Omit `includeSubDomains` from HSTS |
| `SERVER_SECURE_CONTENT_SECURITY_POLICY` | `SecureMiddleware.ContentSecurityPolicy` | `string` | `""` | `Content-Security-Policy` (empty = disabled) |
| `SERVER_SECURE_CSP_REPORT_ONLY` | `SecureMiddleware.CSPReportOnly` | `bool` | `false` | Use `Content-Security-Policy-Report-Only` instead |
| `SERVER_SECURE_HSTS_PRELOAD_ENABLED` | `SecureMiddleware.HSTSPreloadEnabled` | `bool` | `false` | Add `preload` to HSTS header |
| `SERVER_SECURE_REFERRER_POLICY` | `SecureMiddleware.ReferrerPolicy` | `string` | `""` | `Referrer-Policy` (empty = disabled) |

---

## Logger

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `SERVER_LOGGER_LEVEL` | `Logger.Level` | `slog.Level` | `DEBUG` | Global log level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `SERVER_LOGGER_REQUEST_LEVEL` | `Logger.RequestLevel` | `slog.Level` | `DEBUG` | Level for request/response log events |
| `SERVER_LOGGER_LOG_REQUEST_BODY` | `Logger.LogRequestBody` | `bool` | `false` | Log request body (skips form/multipart) |
| `SERVER_LOGGER_LOG_REQUEST_HEADER` | `Logger.LogRequestHeader` | `bool` | `false` | Log request headers (Authorization redacted) |
| `SERVER_LOGGER_LOG_REQUEST_QUERY` | `Logger.LogRequestQuery` | `bool` | `false` | Log URL query parameters |
| `SERVER_LOGGER_LOG_RESPONSE_BODY` | `Logger.LogResponseBody` | `bool` | `false` | Log response body (JSON only) |
| `SERVER_LOGGER_LOG_RESPONSE_HEADER` | `Logger.LogResponseHeader` | `bool` | `false` | Log response headers |

---

## Database

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `PGHOST` | `Database.Host` | `string` | `postgres` | PostgreSQL host |
| `PGPORT` | `Database.Port` | `int` | `5432` | PostgreSQL port |
| `PGDATABASE` | `Database.Database` | `string` | `development` | Database name |
| `PGUSER` | `Database.Username` | `string` | `dbuser` | Database user |
| `PGPASSWORD` | `Database.Password` | `string` | `dbpass` | Database password (sensitive) |
| `PGSSLMODE` | `Database.AdditionalParams["sslmode"]` | `string` | `disable` | SSL mode (`disable`, `require`, `verify-full`) |
| `DB_MAX_OPEN_CONNS` | `Database.MaxOpenConns` | `int` | `NumCPU*2` | Max open DB connections |
| `DB_MAX_IDLE_CONNS` | `Database.MaxIdleConns` | `int` | `1` | Max idle DB connections |
| `DB_CONN_MAX_LIFETIME_SEC` | `Database.ConnMaxLifetime` | `time.Duration` | `60s` | Max connection lifetime in seconds |

---

## Keycloak

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `KEYCLOAK_ISSUER_URL` | `Keycloak.IssuerURL` | `string` | `http://localhost:8080/realms/myrealm` | OIDC issuer URL used for discovery and issuer validation |
| `KEYCLOAK_AUDIENCE` | `Keycloak.Audience` | `string` | `api` | Expected access-token audience/client ID |
| `KEYCLOAK_HTTP_TIMEOUT_SEC` | `Keycloak.HTTPTimeout` | `time.Duration` | `5s` | Timeout for discovery and JWKS HTTP requests |
| `KEYCLOAK_CLOCK_SKEW_SEC` | `Keycloak.ClockSkew` | `time.Duration` | `30s` | Allowed clock skew when validating token times |

Legacy fallback env vars still read: `KEYCLOAK_ISS`, `KEYCLOAK_CLIENT_ID`

---

## Pprof

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `SERVER_PPROF_ENABLE` | `Pprof.Enable` | `bool` | `false` | Mount `/debug/pprof` routes |
| `SERVER_PPROF_ENABLE_MANAGEMENT_KEY_AUTH` | `Pprof.EnableManagementKeyAuth` | `bool` | `true` | Guard pprof with `?mgmt-secret=` query param |
| `SERVER_PPROF_RUNTIME_MUTEX_PROFILE_FRACTION` | `Pprof.RuntimeMutexProfileFraction` | `int` | `0` | `runtime.SetMutexProfileFraction` value (0 = off) |

---

## Management

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `SERVER_MANAGEMENT_SECRET` | `Management.Secret` | `string` | `""` | Secret for pprof key auth (`?mgmt-secret=<value>`) |
