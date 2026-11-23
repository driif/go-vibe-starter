# Production deployment

This template supplies a backend foundation. Deployment owns TLS, secret distribution, database
backups, Keycloak operations, network policy and resource limits. See [authentication](auth.md).

## Shipped behavior and checks

| Behavior | Where | Proven by |
|---|---|---|
| Connect, gRPC, gRPC-Web, JSON POST and safe GET; HTTP/1.1 and h2c | server / generated contract | protocol integration tests |
| Fail-closed bearer auth; explicit local bypass | config / identity / HTTP middleware | auth tests, real Keycloak smoke, image smoke |
| Domain-error mapping, panic recovery, input validation | RPC interceptors | error tests; validation failure once a message carries a rule |
| JSON access logs with request id and RPC result | request logging middleware | access-log tests |
| pgx pool, embedded migrations, startup schema check | database package / CLI | migration and schema integration tests |
| Liveness/readiness, graceful shutdown, 4 MiB message limits | server | health, shutdown and body-limit tests |
| Exact-origin CORS | HTTP middleware | CORS tests |
| Duration histogram and pprof on opt-in private listener | management server | metrics and listener tests |
| Static non-root image, version and built-in health probe | Dockerfile / CLI | `make image-smoke` |
| Drift, lint, race tests, advisories and skills mirror | CI / Make | `make check`, `make vuln` |

## Required configuration

```sh
APP_ENVIRONMENT=production
AUTH_PROVIDER=keycloak
KEYCLOAK_ISSUER_URL=https://identity.example.com/realms/company
KEYCLOAK_AUDIENCE=api
DATABASE_URL=postgres://api:REPLACE@db.example.com/api?sslmode=verify-full&pool_max_conns=20
LISTEN_ADDR=:8080
```

No auth provider or database URL is built in. Local bypass is refused outside the development
allow-list. Process environment overrides `.env.local`; images contain no env file. Never deploy
the local compose credentials. Configure trust roots for your database and identity provider.

## Migrations

Migrations are embedded in the binary. Run the image with `db migrate up` as one deployment job
before rolling out replicas. The migration provider takes a Postgres session lock: connect directly
to Postgres rather than through a transaction-mode pooler. Schema mismatch refuses startup. Back up
before destructive schema changes; deploy backward-compatible changes before removing old fields.

## Image and probes

`make image` builds the image; `VERSION` sets its version stamp. It runs as uid/gid 65532 and includes
only a static binary and the distroless runtime. `probe` checks `/readyz`. Kubernetes should use
`/healthz` for liveness and `/readyz` for readiness. Neither endpoint reveals connection details.

## Listeners, timeouts and shutdown

Expose port 8080 through a TLS-terminating reverse proxy with HTTP/2 support. h2c is for a trusted
private hop only. `MANAGEMENT_ADDR` is disabled by default. If enabled, it exposes unauthenticated
metrics and pprof: bind a private interface or loopback and enforce network policy. Do not publish it.

Header and idle timeouts are set; read/write timeouts are intentionally absent because global
HTTP write deadlines break streaming RPCs. Handlers have no automatic wall-time bound. Apply unary
application timeouts with contexts; for future streams enforce maximum duration, receive limits and
per-message deadlines using `http.NewResponseController` where appropriate. Clients can request a
shorter deadline. Input/output messages are limited to 4 MiB. Auth runs once per stream.

Shutdown marks readiness false and drains both listeners. After 25 seconds remaining streams are cut
and the process exits nonzero. Give Kubernetes enough termination grace and a `preStop` sleep to
let load balancers observe readiness before SIGTERM. No deployment manifests ship.

## Pool and readiness

pgx defaults pool size to max(4, CPU count). Set `pool_max_conns` and other pgx options in the URL
for deployment capacity. Readiness shares the application pool; saturation can make it fail. It
pings Postgres and requires the verifier to have loaded keys once. Later identity outages do not
remove readiness; monitor `unavailable` responses as well. Backups and restore drills are deployment work.

## CORS, logs and metrics

CORS is closed by default. Set exact frontend origins, including scheme/port, when separate origins
are needed. No wildcard or credentialed cookie flow is supplied. The access log records request id,
method, path/procedure, status/code and elapsed time without tokens, bodies or query strings.
Metrics use bounded procedure and code labels. Alert on availability, latency and RPC
`code="unavailable"`; provide your own Prometheus backend, dashboards and retention.

## Operational additions

Configure TLS and trusted reverse proxy handling for the actual deployment. Add request rate limits
at the edge or in the app according to measured abuse patterns. Tracing is not enabled by default;
add it when requests cross enough services to justify it. Registry publishing, release automation,
image scanning and deployment manifests belong to your delivery platform. Run `buf breaking` against
the base branch when evolving a public contract. Keep dependencies, tool pins and base images current.

No frontend, browser e2e suite, htmx variant, REST variant, tenancy, authorization framework, blob
storage, background jobs or automatic update mechanism for derived projects ships here.
