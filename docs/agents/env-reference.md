# Environment Variable Reference

All configuration is loaded in `internal/server/config/server_config.go` →
`DefaultServiceConfigFromEnv()`, which returns a `config.App`.

No config files. No CLI flags for runtime config. Set these in your shell, in `.env.local`, or in the
container environment. `.env.local` is read from the project root at startup and overrides the
process environment; it is skipped while tests run (`CI` set, or the binary ends in `.test`).

`.env.example` lists the same variables with the same defaults — change both together.

---

## App

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `APP_ENVIRONMENT` | `App.Environment` | `string` | `development` | Environment name. `production` enables the strict startup checks below |
| `PROJECT_ROOT_DIR` | — | `string` | directory of the binary | Root used to locate `.env.local` and `migrations/` |

---

## Auth

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `AUTH_PROVIDER` | `App.Auth.Provider` | `string` | `none` | `none` runs every request as a static dev principal; `keycloak` verifies bearer tokens |
| `AUTH_DEV_SUBJECT` | `App.Auth.DevSubject` | `string` | `local-dev` | Subject of the dev principal injected when `AUTH_PROVIDER=none` |
| `AUTH_DEV_ROLES` | `App.Auth.DevRoles` | `[]string` | `admin` | Comma-separated roles of that dev principal, whitespace trimmed |

`App.Validate()` returns an error — it never panics — when:

- `AUTH_PROVIDER` is anything other than `none` or `keycloak`.
- `AUTH_PROVIDER=none` while `APP_ENVIRONMENT=production`.

The bootstrap calls `Validate()` before building the server and aborts on error.

---

## Server

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `SERVICE_PORT` | `Server.ListenAddr` | `string` | `:8080` | HTTP listen address |

---

## Middleware toggles

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `SERVER_ENABLE_CORS_MIDDLEWARE` | `Server.EnableCORSMiddleware` | `bool` | `true` | CORS headers |
| `SERVER_ENABLE_LOGGER_MIDDLEWARE` | `Server.EnableLoggerMiddleware` | `bool` | `true` | Request/response logger |
| `SERVER_ENABLE_RECOVER_MIDDLEWARE` | `Server.EnableRecoverMiddleware` | `bool` | `true` | Panic recovery, returns 500 |
| `SERVER_ENABLE_REQUEST_ID_MIDDLEWARE` | `Server.EnableRequestIDMiddleware` | `bool` | `true` | `X-Request-Id` header |
| `SERVER_ENABLE_TRAILING_SLASH_MIDDLEWARE` | `Server.EnableTrailingSlashMiddleware` | `bool` | `true` | Strip trailing slashes from URLs |
| `SERVER_ENABLE_SECURE_MIDDLEWARE` | `Server.EnableSecureMiddleware` | `bool` | `true` | Security headers |
| `SERVER_ENABLE_CACHE_CONTROL_MIDDLEWARE` | `Server.EnableCacheControlMiddleware` | `bool` | `true` | `Cache-Control: no-store` |

---

## Security headers

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `SERVER_SECURE_XSS_PROTECTION` | `SecureMiddleware.XSSProtection` | `string` | `1; mode=block` | `X-XSS-Protection` |
| `SERVER_SECURE_CONTENT_TYPE_NOSNIFF` | `SecureMiddleware.ContentTypeNosniff` | `string` | `nosniff` | `X-Content-Type-Options` |
| `SERVER_SECURE_X_FRAME_OPTIONS` | `SecureMiddleware.XFrameOptions` | `string` | `SAMEORIGIN` | `X-Frame-Options` |
| `SERVER_SECURE_HSTS_MAX_AGE` | `SecureMiddleware.HSTSMaxAge` | `int` | `0` | `Strict-Transport-Security` max-age; `0` disables the header |
| `SERVER_SECURE_HSTS_EXCLUDE_SUBDOMAINS` | `SecureMiddleware.HSTSExcludeSubdomains` | `bool` | `false` | Omit `includeSubDomains` from HSTS |
| `SERVER_SECURE_HSTS_PRELOAD_ENABLED` | `SecureMiddleware.HSTSPreloadEnabled` | `bool` | `false` | Add `preload` to HSTS |
| `SERVER_SECURE_CONTENT_SECURITY_POLICY` | `SecureMiddleware.ContentSecurityPolicy` | `string` | `""` | `Content-Security-Policy`; empty disables the header |
| `SERVER_SECURE_CSP_REPORT_ONLY` | `SecureMiddleware.CSPReportOnly` | `bool` | `false` | Send the policy as `Content-Security-Policy-Report-Only` |
| `SERVER_SECURE_REFERRER_POLICY` | `SecureMiddleware.ReferrerPolicy` | `string` | `""` | `Referrer-Policy`; empty disables the header |

---

## Logger

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `SERVER_LOGGER_LEVEL` | `Logger.Level` | `slog.Level` | `DEBUG` | Global log level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `SERVER_LOGGER_REQUEST_LEVEL` | `Logger.RequestLevel` | `slog.Level` | `DEBUG` | Level of request/response log records |
| `SERVER_LOGGER_LOG_REQUEST_BODY` | `Logger.LogRequestBody` | `bool` | `false` | Log request bodies (form and multipart skipped) |
| `SERVER_LOGGER_LOG_REQUEST_HEADER` | `Logger.LogRequestHeader` | `bool` | `false` | Log request headers (`Authorization` redacted) |
| `SERVER_LOGGER_LOG_REQUEST_QUERY` | `Logger.LogRequestQuery` | `bool` | `false` | Log URL query parameters |
| `SERVER_LOGGER_LOG_RESPONSE_BODY` | `Logger.LogResponseBody` | `bool` | `false` | Log response bodies (JSON only) |
| `SERVER_LOGGER_LOG_RESPONSE_HEADER` | `Logger.LogResponseHeader` | `bool` | `false` | Log response headers |

An unparseable level falls back to `DEBUG` instead of failing startup.

---

## Database

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `PGHOST` | `Database.Host` | `string` | `postgres` | PostgreSQL host; use `localhost` outside docker compose |
| `PGPORT` | `Database.Port` | `int` | `5432` | PostgreSQL port |
| `PGDATABASE` | `Database.Database` | `string` | `development` | Database name |
| `PGUSER` | `Database.Username` | `string` | `dbuser` | Database user |
| `PGPASSWORD` | `Database.Password` | `string` | `dbpass` | Database password (sensitive) |
| `PGSSLMODE` | `Database.AdditionalParams["sslmode"]` | `string` | `disable` | `sslmode` connection parameter (`disable`, `require`, `verify-full`) |
| `DB_MAX_OPEN_CONNS` | `Database.MaxOpenConns` | `int` | `NumCPU * 2` | Max open connections |
| `DB_MAX_IDLE_CONNS` | `Database.MaxIdleConns` | `int` | `1` | Max idle connections |
| `DB_CONN_MAX_LIFETIME_SEC` | `Database.ConnMaxLifetime` | `time.Duration` | `60s` | Max connection lifetime, in seconds |

---

## Keycloak (token verification)

Read whenever the config is built; only used when `AUTH_PROVIDER=keycloak`.

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `KEYCLOAK_ISSUER_URL` | `Keycloak.IssuerURL` | `string` | `http://localhost:8080/realms/myrealm` | OIDC issuer used for discovery and `iss` validation |
| `KEYCLOAK_AUDIENCE` | `Keycloak.Audience` | `string` | `api` | Expected access-token audience |
| `KEYCLOAK_HTTP_TIMEOUT_SEC` | `Keycloak.HTTPTimeout` | `time.Duration` | `5s` | Timeout for discovery and JWKS requests, in seconds |
| `KEYCLOAK_CLOCK_SKEW_SEC` | `Keycloak.ClockSkew` | `time.Duration` | `30s` | Allowed clock skew when validating token times, in seconds |
| `KEYCLOAK_ISS` | `Keycloak.IssuerURL` | `string` | — | Legacy fallback, used only when `KEYCLOAK_ISSUER_URL` is unset |
| `KEYCLOAK_CLIENT_ID` | `Keycloak.Audience` | `string` | — | Legacy fallback, used only when `KEYCLOAK_AUDIENCE` is unset |

---

## Keycloak admin API

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `KEYCLOAK_ADMIN_BASE_URL` | `KeycloakAdmin.BaseURL` | `string` | `http://localhost:8080` | Keycloak base URL for admin calls |
| `KEYCLOAK_ADMIN_REALM` | `KeycloakAdmin.Realm` | `string` | `myrealm` | Realm managed through the admin API |
| `KEYCLOAK_ADMIN_CLIENT_ID` | `KeycloakAdmin.ClientID` | `string` | `""` | Service-account client id |
| `KEYCLOAK_ADMIN_CLIENT_SECRET` | `KeycloakAdmin.ClientSecret` | `string` | `""` | Service-account client secret (sensitive) |

---

## Pprof and management

| Env var | Go field | Type | Default | Description |
|---|---|---|---|---|
| `SERVER_PPROF_ENABLE` | `Pprof.Enable` | `bool` | `false` | Mount `/debug/pprof` routes |
| `SERVER_PPROF_ENABLE_MANAGEMENT_KEY_AUTH` | `Pprof.EnableManagementKeyAuth` | `bool` | `true` | Guard pprof with `?mgmt-secret=` |
| `SERVER_PPROF_RUNTIME_MUTEX_PROFILE_FRACTION` | `Pprof.RuntimeMutexProfileFraction` | `int` | `0` | `runtime.SetMutexProfileFraction` value; `0` is off |
| `SERVER_MANAGEMENT_SECRET` | `Management.Secret` | `string` | `""` | Value expected in `?mgmt-secret=` (sensitive) |
