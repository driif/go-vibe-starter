# Production Configuration Guide

This guide walks through every setting you should review before deploying to production.
The defaults are safe for local development — not for production.

---

## 1. Required env vars (no safe defaults)

Set these before anything else. The server will start without them but will behave incorrectly.

```env
# Database
PGHOST=your-db-host
PGDATABASE=your-db-name
PGUSER=your-db-user
PGPASSWORD=<strong-random-password>
PGSSLMODE=require

# Keycloak
KEYCLOAK_ISSUER_URL=https://your-keycloak-domain/realms/your-realm
KEYCLOAK_AUDIENCE=your-backend-client
KEYCLOAK_HTTP_TIMEOUT_SEC=5
KEYCLOAK_CLOCK_SKEW_SEC=30

# Management (if pprof is enabled)
SERVER_MANAGEMENT_SECRET=<strong-random-secret>
```

---

## 2. Server port

```env
SERVICE_PORT=:8080
```

The server does not terminate TLS — run it behind a reverse proxy (nginx, Caddy, Traefik).
The proxy should handle HTTPS on 443 and forward to this port over plain HTTP.

---

## 3. Security headers

The defaults are a reasonable starting point. For production, tune these:

```env
# Already on by default — keep them
SERVER_SECURE_XSS_PROTECTION=1; mode=block
SERVER_SECURE_CONTENT_TYPE_NOSNIFF=nosniff
SERVER_SECURE_X_FRAME_OPTIONS=DENY              # stricter than SAMEORIGIN if no iframes needed

# Enable HSTS — only do this when you are committed to HTTPS
SERVER_SECURE_HSTS_MAX_AGE=31536000             # 1 year
SERVER_SECURE_HSTS_EXCLUDE_SUBDOMAINS=false     # false = includeSubDomains
SERVER_SECURE_HSTS_PRELOAD_ENABLED=false        # true only if you submit to HSTS preload list

# Content Security Policy — tune to your actual needs
SERVER_SECURE_CONTENT_SECURITY_POLICY=default-src 'self'
SERVER_SECURE_CSP_REPORT_ONLY=false             # set true first to test without breaking things

# Referrer Policy
SERVER_SECURE_REFERRER_POLICY=strict-origin-when-cross-origin
```

> **HSTS warning**: Once a browser sees `max-age=31536000`, it will refuse HTTP connections
> to your domain for a year. Only set this after your TLS setup is confirmed stable.

---

## 4. Logger

In development, DEBUG level with no body/header logging is fine.
In production, switch to INFO and be careful about body logging — it can log PII.

```env
SERVER_LOGGER_LEVEL=INFO
SERVER_LOGGER_REQUEST_LEVEL=INFO

# Disable body/header logging unless actively debugging
SERVER_LOGGER_LOG_REQUEST_BODY=false
SERVER_LOGGER_LOG_REQUEST_HEADER=false
SERVER_LOGGER_LOG_REQUEST_QUERY=false
SERVER_LOGGER_LOG_RESPONSE_BODY=false
SERVER_LOGGER_LOG_RESPONSE_HEADER=false
```

What is always logged (not configurable):
- HTTP method, URL, status code, response size, duration
- Request ID (from `X-Request-Id`)

---

## 5. Database connection pool

The defaults (`MaxOpenConns = NumCPU*2`, `MaxIdleConns = 1`) are conservative.
For a production API under sustained load, tune based on your database's `max_connections` and
the number of app instances you run.

A practical starting point for a single instance against Postgres with `max_connections=100`:

```env
DB_MAX_OPEN_CONNS=20
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME_SEC=300    # 5 minutes — prevents stale connections
```

Rule of thumb: `(max_connections - 5 reserved for admin) / number_of_app_instances`.

---

## 6. CORS

The default uses `cors.AllowAll()`, which allows any origin. This is fine for development and
internal APIs. For a public API, replace it in `server.go`:

```go
// Replace:
r.Use(cors.AllowAll().Handler)

// With:
r.Use(cors.New(cors.Options{
    AllowedOrigins:   []string{"https://your-frontend.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           300,
}).Handler)
```

---

## 7. Pprof

Disabled by default. Do not enable it in production unless you are actively profiling,
and always gate it with a management secret.

```env
SERVER_PPROF_ENABLE=false                            # keep off unless profiling
SERVER_PPROF_ENABLE_MANAGEMENT_KEY_AUTH=true         # always true if enabled
SERVER_MANAGEMENT_SECRET=<strong-random-secret>      # required when key auth is on
```

When enabled, pprof is available at `/debug/pprof?mgmt-secret=<secret>`.

---

## 8. Health check

The starter does not include a `/healthz` route by default. Add one early in your route setup:

```go
r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
})
```

For a deeper check (e.g., database ping), inject `s.DB` and add a ping:

```go
r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
    if err := s.DB.PingContext(r.Context()); err != nil {
        http.Error(w, "db unavailable", http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
})
```

---

## 9. TLS / reverse proxy

The server listens on plain HTTP. Put it behind a reverse proxy that handles TLS.

**Caddy (simplest):**
```
your-domain.com {
    reverse_proxy localhost:8080
}
```

**Nginx:**
```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    ssl_certificate     /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Request-Id $request_id;
    }
}
```

Pass `X-Request-Id` from the proxy so request IDs are consistent across logs.

---

## 10. Docker

The included `Dockerfile` is a placeholder. A minimal production Dockerfile for this project:

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bin/app ./

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/app .
EXPOSE 8080
CMD ["./app", "run"]
```

Set all env vars via your orchestrator (Kubernetes secrets, Docker secrets, etc.).
Never bake secrets into the image.

---

## Minimal production `.env` example

```env
APP_ENVIRONMENT=production
SERVICE_PORT=:8080

SERVER_LOGGER_LEVEL=INFO
SERVER_LOGGER_REQUEST_LEVEL=INFO

SERVER_SECURE_HSTS_MAX_AGE=31536000
SERVER_SECURE_X_FRAME_OPTIONS=DENY
SERVER_SECURE_REFERRER_POLICY=strict-origin-when-cross-origin
SERVER_SECURE_CONTENT_SECURITY_POLICY=default-src 'self'

SERVER_PPROF_ENABLE=false

PGHOST=db
PGDATABASE=myapp
PGUSER=myapp_user
PGPASSWORD=<from-secrets>
PGSSLMODE=require
DB_MAX_OPEN_CONNS=20
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME_SEC=300

KEYCLOAK_ISSUER_URL=https://auth.myapp.com/realms/myapp
KEYCLOAK_AUDIENCE=backend
```
