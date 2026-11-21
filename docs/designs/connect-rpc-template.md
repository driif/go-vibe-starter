# Connect-RPC template refactor

*   **Author:** driif
*   **Status:** Approved
*   **Date:** 2026-11-21

--------------------------------------------------------------------------------

## Overview

go-vibe-starter is a GitHub template for Go backends. Today it serves REST through chi and
oapi-codegen, uses lib/pq, ships its own Keycloak token verifier, and has an init script that no
test runs. This refactor turns it into a Go-only template for Connect-RPC backends: stdlib
`http.ServeMux` with go-pkgz/routegroup, pgx/v5 + sqlc + goose, buf + connect-go v2 with
protovalidate, Keycloak through `github.com/coreos/go-oidc/v3`, `AUTH_PROVIDER=none` for local
work, and an in-process OIDC provider for tests.

The product is the tree that exists after `make init`. The refactor defines that tree and keeps a
permanent `WhoAmIService` in it. After the demo is removed, that service still exercises buf, the
interceptor chain except the failing path of validation, the token verifier and the test harness;
sqlc and a non-empty migration run only while the demo is present. A template-only CI job proves
the tree: it copies the repository, runs init with a different module path, then generates,
builds, lints and tests, and fails on residue. The definition of done is in Goals.

OpenForms is a project that was created from this template and later moved to Connect-RPC; it is
called the reference project below. Its code is reused where it is generic; section 21 of the
Technical Plan lists what is ported and what is not.

The work lands in four stages on one branch and is merged as one pull request: stage 0 tags the
old stack, stage 1 swaps the stack, stage 2 rewrites the docs and skills, stage 3 rewrites init
and CI. No stage is released on its own.

## The Problem

A review of the first plan for this refactor found six blockers. Each names the Technical Plan
section that answers it.

*   **B1. The tree after init is untested and init breaks under buf.** CI never runs init. Init
    renames the module path with `sed` over `*.go` and `*.md` only. Generated `.pb.go` files hold
    the Go package path inside a length-prefixed descriptor; a spike on 2026-11-20 showed that a
    `sed` rename to a path of another length builds and then panics at package load. Init
    therefore has to skip generated code and regenerate, which makes the generators
    prerequisites of init. (Sections 14, 15, 18.)
*   **B2. The tree without the demo is undefined.** Removing the `notes` demo removes the only
    proto file, query file and migration. sqlc fails with no queries, buf fails with no `.proto`
    file, `//go:embed migrations/*.sql` fails with no match, and no service is left to exercise
    the interceptor chain, the verifier or the harness. (Sections 10, 11, 12.)
*   **B3. Auth fails open when no environment is set.** `AUTH_PROVIDER` defaults to `none`,
    `APP_ENVIRONMENT` to `development` and the dev principal to role `admin`. A container started
    with no environment serves every request as admin. (Section 8.)
*   **B4. "Modelled on the reference project" is not a specification.** The reference project
    has no routegroup, no `AUTH_PROVIDER=none` and no init script; its auth interceptor rejects a
    request with no bearer header before the provider is asked; tenancy is in its auth, database,
    principal, logging and fixtures; its `apperr` imports Connect. (Sections 7, 21.)
*   **B5. The quickstart fails on a fresh clone.** Config defaults do not match the compose file,
    the README names port 9880 while the code listens on 8080, Keycloak publishes the port the
    server listens on, and the quickstart has no env or migrate step. (Sections 8, 15, 17.)
*   **B6. After init the skills and `AGENTS.md` point at files init deleted.** They tell the
    agent to copy the `notes` slice. The `setup-project` skill uses the template module path as
    its "done" sentinel, which init rewrites, so the check can never pass. (Sections 14, 19.)

Further findings the design closes:

*   **F1. Verifier.** The template's own verifier checks claims before the signature, refetches
    keys without limit for an unknown `kid`, never compares the discovery issuer, supports RSA
    PKCS#1 v1.5 only, fails every request when its key cache expires during an outage, and
    accepts an empty `sub`. The reference project's copy has fixes that were not applied to the
    template: copied code has no update channel. (Section 7.)
*   **F2. Realm.** The shipped realm export has no `oidc-audience-mapper` that adds `api`, so it
    cannot satisfy the verifier; nothing tests it and there is no documented way to get a token.
    (Section 17.)
*   **F3. Dead surface.** Configuration carries seven middleware enable flags, nine header
    knobs, five body-logging knobs and legacy fallbacks. The Keycloak admin client, four role and
    scope guards, a seed command and a test harness have no caller. (Sections 8, 22.)
*   **F4. Unary-only interceptors.** The reference project's interceptors cover unary calls
    only. In a copied template the first streaming RPC would skip error mapping without a
    warning. (Section 6.)
*   **F5. No stopping rule.** "Near production ready" has no stopping rule, and the sample
    environment in `docs/production.md` sets `APP_ENVIRONMENT=production` and no
    `AUTH_PROVIDER`, a combination the server refuses to start with. (Section 20.)
*   **F6. Stale agent material.** The words chi, OpenAPI or `database/sql` occur in 28 of the 44
    files under `.claude/skills/` (run at `b33836c` on 2026-11-21: `git grep -liIE -e
    '(^|[^A-Za-z])chi([^A-Za-z]|$)' -e 'openapi' -e 'database/sql' -- .claude/skills`), and
    stage 2 changes 39 of the 44. The layout is written down in five places: `AGENTS.md`,
    `CONTEXT.md`, `README.md`, `docs/agents/README.md` and the `go-feature` skill. (Section 19.)

## Goals

*   A Go-only template for Connect-RPC backends on the stdlib mux with routegroup, pgx/v5 + sqlc
    + goose, buf + connect-go v2.
*   A defined tree after init that builds, lints and tests, proven by a template-only CI job
    that runs init.
*   Init that excludes generated files from the rename, regenerates, and checks for its tools
    first.
*   Tests that, after the demo is removed, still exercise HTTP, the interceptor chain except the
    failing path of validation, token verification and the Postgres harness.
*   Keycloak as the only real provider, token verification with fail-closed defaults, and a test
    provider used by Go integration tests only.
*   A dev env file that matches the shipped compose file, no built-in default for connection
    values, and a compose file whose containers, network and volume do not collide between two
    derived projects on one machine.
*   A non-root image that passes `make image-smoke`, embedded migrations with a migrate
    subcommand, a request size limit, a CORS allow-list.
*   Deletion of the middleware toggles, unused guards, legacy env fallbacks and the other dead
    surface listed in F3.
*   `AGENTS.md`, `docs/` and the skills pack rewritten for the new stack, with a mirror check in
    `make check` and a banned-term check in `scripts/template-smoke.sh`.
*   A "ships / documented / not included" table in place of the phrase "near production ready".
*   An explicit include/exclude list for code from the reference project.

**Definition of done.** `template-smoke.yml` is green on its three legs (`stripped`,
`keep-demo`, `macos`), and `ci.yml` and `keycloak.yml` are green, all on the pull request that
carries the four stages. Every other mention of "done" in this document refers to this
paragraph.

## Non-Goals

*   Any frontend skeleton; browser e2e tests.
*   An htmx template repository and any shared-layer design for it.
*   A shared Go framework module.
*   LDAP access from the Go app.
*   The reference project's tenancy, RLS, authz / CEL, blob storage, plugins, runs, retention,
    SPA embedding, docs site, deploy manifests, release automation, rate limiting.
*   Metrics backend, tracing, TLS, secret store, backups, deploy-time migration orchestration.
    These are documented as per-project work.
*   A maintained REST / OpenAPI variant. The old stack is kept as a tag only.
*   Init-time variant selection.
*   An authorization framework. A role check is one helper call inside a handler.
*   Streaming RPCs in the template, and a server-side time bound on handlers.
*   Removing a kept demo with init. `--keep-demo` is final as far as init is concerned.
*   An update mechanism from the template to derived projects.

## Open Questions

*   **`actions/setup-go` and the `toolchain` line.** Whether `go-version-file: go.mod` selects
    the version on the `toolchain` line (`go1.27.4`) or the one on the `go` line (`1.26.0`) was
    not checked. Check: the Go version printed by the first run of `ci.yml`. Fallback: if it
    prints 1.26.x, the three workflows set `go-version: "1.27"` and the Go version is then
    stated in a third place next to `go.mod` and the `Dockerfile`.

## Critical User Journeys (CUJs)

### CUJ 1: Create a project from the template and initialise it

*   **Given**: a repository created with "Use this template", cloned, with Go, Docker, Make, git,
    curl and a C compiler installed, and `make tools` done.
*   **When**: the user runs `scripts/init.sh --module github.com/me/billing --name billing --yes`
    (or `make init` and answers the prompts).
*   **Then**: init exits 0 and prints its next steps; `go.mod`, imports and `buf.gen.yaml` carry
    the new path; the binary is `bin/billing` and the compose project is `billing`; no demo file
    exists; generated code is regenerated, never edited; `LICENSE` and the template-only files
    are gone; `make check` passes without a commit in between; a second identical run changes no
    file. With `--module foo` or `--name My_App` init exits 2 and changes nothing. With a tool
    missing it stops before changing anything and prints
    `tools missing or outdated: run 'make tools'`.

### CUJ 2: First run with no external dependency beyond Postgres

*   **Given**: a fresh clone (initialised or not), Docker running, no `.env.local`.
*   **When**: the developer runs `make tools`, `make dev-up`, `make migrate-up`, `make run`, then
    `curl localhost:8080/readyz` and
    `curl -X POST -H 'Content-Type: application/json' -d '{}' localhost:8080/api.v1.WhoAmIService/WhoAmI`.
*   **Then**: only the Postgres container starts; `.env.local` is created from `.env.example`;
    the server logs `server started` with `addr=127.0.0.1:8080`, `auth_provider=none` and one
    WARN `authentication disabled: every request runs as the dev principal`; `/readyz` answers
    200; `WhoAmI` returns subject `local-dev` with role `admin`.

### CUJ 3: A container started without configuration refuses to start

*   **Given**: the built image.
*   **When**: it is started with no environment, or with only `AUTH_PROVIDER=none`, or with
    `AUTH_PROVIDER=none` and `APP_ENVIRONMENT` set to `prod`, `Production` or `staging`.
*   **Then**: it exits 1 before opening a socket, with one JSON record that lists every
    violation. The first case names `AUTH_PROVIDER is required` and `DATABASE_URL is required`;
    the others name the dev allow-list.

### CUJ 4: Switch to Keycloak

*   **Given**: CUJ 2 works.
*   **When**: the developer runs `make keycloak-up`, sets `AUTH_PROVIDER=keycloak` in
    `.env.local`, runs `make run`, then `TOKEN=$(make -s token)` and calls `WhoAmI` with
    `Authorization: Bearer $TOKEN`; then runs `make smoke`.
*   **Then**: startup logs `identity provider ready`; the response is username `alice`, name
    `Alice Admin`, roles `admin` and `user` (`KC_USER=bob` gives `user` only); a call with no
    token, an expired token or an ID token gets HTTP 401 with
    `{"code":"unauthenticated",...}` and `WWW-Authenticate`; `make smoke` passes against the
    shipped realm. With a trailing slash on `KEYCLOAK_ISSUER_URL` the config is rejected; with
    another host name than Keycloak's `KC_HOSTNAME` the server exits at startup and prints both
    URLs.

### CUJ 5: Add a new RPC domain

*   **Given**: the template tree or a stripped tree.
*   **When**: the developer adds `proto/api/v1/tasks.proto`, `sql/migrations/0000N_tasks.sql`,
    `sql/queries/tasks.sql`, runs `make gen`, writes `internal/domain/tasks`, `internal/rpc/tasks`,
    `internal/server/routes_tasks.go` with `mountTasks`, and adds one line to
    `internal/server/routes.go`.
*   **Then**: `make check` passes; the service is behind auth, validation, error mapping, the
    access log and the duration histogram with no other edit in `internal/server`; in a stripped
    tree `internal/db/gen` appears with the first query file and the migration applies as
    version 1.

### CUJ 6: Write an integration test

*   **Given**: a handler for `XService`.
*   **When**: the developer writes, in an external test package, `s := test.NewServer(t)` and
    calls `apiv1connect.NewXServiceClient(s.Client(s.Token("alice", "admin")))`, and again with a
    token minted from `s.Provider.Claims("alice")` with `exp` in the past.
*   **Then**: the first call passes the production middleware, the go-oidc verifier and the
    interceptors and hits a private migrated database, in parallel with other tests; the second
    fails with `connect.CodeUnauthenticated`. No container other than Postgres is involved.

### CUJ 7: Build and run the container

*   **Given**: a derived project.
*   **When**: the operator runs `make image VERSION=v1.2.0`, runs the image with `db migrate up`
    as a job, then starts it with `AUTH_PROVIDER=keycloak` and the production variables.
*   **Then**: `--version` prints `v1.2.0`; the process runs as uid 65532; the container reports
    healthy when `/readyz` is 200 (`/app probe`).

### CUJ 8: An agent builds a feature after init

*   **Given**: a stripped tree and a request for a feature.
*   **When**: the agent loads `go-feature`.
*   **Then**: the skill carries its own snippets and points at the `whoami` files, which are
    present; no skill, `AGENTS.md` section or doc names a path that init deleted (the residue
    and banned-term steps of `scripts/template-smoke.sh` proved that in CI on the stripped
    tree); `setup-project` detects that init has run because `scripts/template-smoke.sh` is
    absent.

### CUJ 9: Run before migrating; migrate and roll back

*   **Given**: a tree with migrations and a database that was never migrated.
*   **When**: the user runs `make run`; then `make migrate-up`; later the operator runs
    `db migrate down` and `db migrate status`.
*   **Then**: `run` exits 1 with `database schema is at version 0, this binary needs 1: run "db
    migrate up"`; `up` applies pending migrations and prints one line each; `down` rolls back
    exactly the most recent one and `status` shows it as `pending`. With no migration embedded
    each command prints `no migrations` and exits 0.

### CUJ 10: A handler returns a domain error or guards by role

*   **Given**: a handler that returns `apperr.New(apperr.NotFound, "NOTE_NOT_FOUND", ...)`, and one
    that starts with `identity.FromContext(ctx).HasRole("admin")`.
*   **When**: a client calls them on Connect, gRPC or gRPC-Web; bob calls the guarded one.
*   **Then**: the first returns `not_found` with `ErrorInfo.reason = NOTE_NOT_FOUND` and logs
    nothing at ERROR; a raw driver error returns `internal: internal error` and the log has the
    cause under the request id; bob gets `permission_denied` with reason `ADMIN_REQUIRED`.

### CUJ 11: Keycloak outage

*   **Given**: `AUTH_PROVIDER=keycloak`.
*   **When**: Keycloak is unreachable when the API starts; separately, Keycloak stops while the
    API runs.
*   **Then**: started during the outage, the API logs one WARN, `/readyz` answers 503 with the
    fixed body `identity provider unavailable`, and every RPC gets `unavailable` (HTTP 503) with
    one ERROR record; once Keycloak answers, the next readiness probe or token completes
    discovery within 5 s and `/readyz` turns 200 without a restart. When Keycloak stops later,
    `/readyz` stays 200, tokens signed by a cached key keep working, an RPC that needs a key
    fetch gets `unavailable`, and `rpc_server_duration_seconds_count{code="unavailable"}` rises.

### CUJ 12: Rolling restart

*   **Given**: requests in flight.
*   **When**: the orchestrator sends SIGTERM.
*   **Then**: the listeners stop accepting; requests in flight get 25 s to finish; the process
    logs `server stopped` and exits 0. Anything still open after 25 s, a stream included, is
    cut and the exit code is 1. A second signal ends the process immediately.

### CUJ 13: Database outage

*   **Given**: a running API.
*   **When**: Postgres becomes unreachable.
*   **Then**: `/healthz` stays 200 and `/readyz` answers 503 with the fixed body
    `database unavailable`; the driver error is in the log only.

### CUJ 14: Trace a request, scrape metrics, take a profile

*   **Given**: `MANAGEMENT_ADDR=:9090`.
*   **When**: a caller sends `X-Request-Id: abc-123`; Prometheus scrapes `:9090/metrics`; an
    operator runs `go tool pprof http://<pod>:9090/debug/pprof/profile?seconds=30`.
*   **Then**: every log record of that request, the access line included, has
    `request_id=abc-123`, and RPC records have `procedure`; metrics and pprof answer without a
    token; the same paths on the API port answer 404.

### CUJ 15: A pull request forgets to regenerate

*   **Given**: a change to a `.proto` or query file without the regenerated code.
*   **When**: CI runs `make check`.
*   **Then**: the job fails at the drift step, lists the paths that differ and prints
    `generated files or go.mod are stale: commit the output of 'make gen' and 'go mod tidy'`.

### CUJ 16: The template maintainer changes init

*   **Given**: a change to `scripts/init.sh`, staged.
*   **When**: the maintainer runs one leg locally with its command from section 18, or opens the
    pull request, which runs all three.
*   **Then**: each leg (`stripped`, `keep-demo`, `macos`) passes or names the failing step.

## Technical Plan (Architecture)

Each contract is stated once. This section gives behaviour; the Detailed Implementation Plan
gives the literals (signatures, messages, command lines, file contents) under the entry of the
file that holds them. "Section N" means a numbered section of this Technical Plan;
"`AGENTS.md` §N" means a section of that file.

**Fixed requirements.** These come from the author and are not design choices of this document:

*   Go only. No frontend, no second variant.
*   chi is removed. The router is the stdlib `http.ServeMux`. Accepted cost: no verb helpers;
    path values come from `r.PathValue`.
*   go-pkgz/routegroup stays on top of the mux. Its three jobs here are the root middleware
    stack, the public group for probes and the authed group for RPC services and plain routes.
    Accepted cost: one dependency where a Connect-only server could use the bare mux and a
    short wrap function, as the reference project does.
*   Keycloak is the only real identity provider. `AUTH_PROVIDER=none` stays as the dev mode.
*   `connectrpc.com/connect/v2`, the current stable major.
*   `go.mod` declares `go 1.26.0`, the floor the dependencies force, and `toolchain go1.27.4`,
    the toolchain everything is built and tested with.

**Evidence.** Library versions and API facts were checked on 2026-11-20 and 2026-11-21. Nothing
from these runs is committed; the design does not depend on their files.

| What | How it was checked |
|---|---|
| Core wiring: buf v2 managed mode with `go tool` plugins, `Handler` on routegroup, auth as HTTP middleware on four client configurations, the go-oidc verifier with the key-set wrapper and replay transport, the `RequestLog` and `Errors` interceptors with validation, streaming through the whole stack on HTTP/1.1 and h2c, pgxpool + goose over embedded files with zero migrations, the Postgres harness, the `sed` failure on generated code | A spike module: `go test -race ./...` passed on 2026-11-20 with go1.27.4, buf 1.75.0, sqlc v1.30.0, `postgres:17-alpine` |
| Hand-registered test procedures on all four client configurations; auth, a plain route with the body cap and an uncapped Connect route side by side; clearing the read and write deadline for a stream | A side probe: `go test` passed on 2026-11-20 |
| The realm file with the verifier, middleware, handler and harness against real Keycloak | Run on 2026-11-20 against `quay.io/keycloak/keycloak:26.8.2`: two tests, nine cases, all passed. That run published Keycloak on host port 8280 |
| The compose service shape with `--http-port=8181` | Started on 2026-11-20: healthy in 16 s; discovery issuer `http://localhost:8181/realms/app`; a password-grant token carried `aud: "api"`, claim `typ: "Bearer"` and `realm_access.roles` |
| `CheckSchema` cases; the uniform mount signature under `unparam`; `EmptyDatabase`; sqlc and buf on a tree without the demo | Run on 2026-11-20 in a copy of the spike |
| The readiness latch: priming go-oidc's key cache without a token | A test on 2026-11-21 against go-oidc v3.21.0 and the in-process provider: after priming, with the provider down and no earlier token, a valid token verified; discovery plus key fetch took two requests |
| API signatures quoted in the plan (`connect.NewServer`, `connecthttp.Mount`, `connecthttp.NewErrorWriter`, `connecthttp.WithReadMaxBytes`, `validate.NewServerInterceptor`, `routegroup.Bundle`, `pgx.BeginFunc`, `goose.WithSessionLocker`, `lock.NewPostgresSessionLocker`) | `go doc` on 2026-11-21 at the versions of section 22 |
| Tool behaviour init and the Makefile rely on (`go mod edit -module` as validator, `git grep --untracked`, stamp files and `awk` help under GNU Make 3.81, `golangci-lint config verify` with the three added linters) | Run on 2026-11-20 on macOS with bash 3.2 and BSD userland |

Designed here and expected to work, but not run: `cmd/` (the commands, signal handling, exit
codes), `internal/server/config`, `identity.DevVerifier`, calling the verifier with an empty
token, the two `WWW-Authenticate` forms, the duration histogram in `AccessLog`, `/readyz`
consulting the verifier, the goose session locker, the copy-compare drift check, the rewritten
init, `scripts/template-smoke.sh` and `scripts/image-smoke.sh`, the image, and the pinned sqlc v1.31.1 and govulncheck v1.10.0.
Where a single claim below rests on inference it says "expected; not run".

### 1. Package layout and import rules

```text
main.go                          cmd.Execute() only
cmd/                             root.go, run.go, db.go (db migrate up / down / status), probe.go
internal/server/                 server.go (Handler, New, Serve), routes.go (hub),
                                 routes_<domain>.go (one mount file per domain), health.go, management.go
internal/server/config/          flat env reader + Validate, .env.local parser
internal/server/middleware/      RequestID, AccessLog, Recover, SecurityHeaders, CORS, BodyLimit, Auth
internal/rpc/interceptor/        RequestLog, Errors (validation is connectrpc.com/validate)
internal/rpc/<domain>/           Connect handlers (whoami permanent, notes DEMO)
internal/domain/<domain>/        logic + store (notes DEMO)
internal/reqlog/                 request-log carrier and slog handler
internal/apperr/                 error type with its own Kind, no Connect import
internal/identity/               Principal, context helpers, ErrUnauthenticated, ErrUnavailable, HasRole, DevVerifier
internal/identity/keycloak/      verifier on go-oidc, replay transport
internal/identity/testprovider/  in-process discovery + JWKS + token minting for tests
internal/db/                     Open, Migrate, CheckSchema; never imports internal/db/gen
internal/db/gen/                 sqlc output, committed; absent in a stripped tree
internal/gen/api/v1/             buf output (protobuf + connect), committed
internal/test/                   Postgres harness, test server + client, Keycloak smoke test (build tag)
proto/api/v1/                    whoami.proto (permanent), notes.proto (DEMO)
sql/migrations/, sql/queries/, sql/embed.go
```

Terms used below: the **template tree** is the repository before init; the **stripped tree** is
the result of init without `--keep-demo`; the **keep-demo tree** is the result with it; a
**derived project** is a repository created from the template. The **hub** is
`internal/server/routes.go`; a **mount file** is `internal/server/routes_<domain>.go`.
**Integration tests** run in `make test`; **smoke** means `make smoke` against compose Keycloak;
the other two scripts are always written out as `template-smoke` and `image-smoke`.

Import rules:

*   `internal/server` does not import `internal/server/config`. `cmd/run.go` reads the config and
    fills `server.Deps`, so tests build the same handler without touching the environment.
*   Only `internal/domain/*` imports `internal/db/gen`. `internal/db` never does.
*   `internal/domain/*` imports no transport package (`internal/rpc`, `internal/gen`,
    `connectrpc.com/*`, `net/http`) and not `internal/identity`; the owner arrives as a `string`.
*   `internal/apperr` and `internal/identity` do not import Connect. Only
    `internal/identity/keycloak` imports go-oidc.
*   Nothing under `cmd/` or `internal/server` imports `internal/identity/testprovider`; the
    binary carries no token minting. Tests in those directories reach the provider through
    `internal/test`.
*   Interfaces are declared by their consumer: `middleware.Verifier` in
    `internal/server/middleware/auth.go`, the two one-method readiness interfaces in
    `internal/server/health.go`.

| Layer | Owns | Must not |
|---|---|---|
| `proto/api/v1/<domain>.proto` | RPC names, messages, field-shape rules, `idempotency_level` | — |
| `validate` interceptor | rejects a request that breaks a field rule before the handler runs | — |
| `internal/rpc/<domain>` | read the principal, parse ids, one store call per RPC, map domain type to proto, role check | SQL, transactions, business rules, import `internal/db/gen` |
| `internal/domain/<domain>` | rules that are not field shape, defaults, mapping `pgx.ErrNoRows` and constraint errors to `apperr`, transactions | import a transport package |
| `sql/queries/<domain>.sql` | the SQL; owner predicate in every statement | — |
| `internal/server/routes_<domain>.go` | construct store and handler, register the service | logic |

### 2. Request path

```text
client
  │
  ▼  http.Server on LISTEN_ADDR     HTTP/1.1 + unencrypted HTTP/2, one port for Connect, gRPC, gRPC-Web
  │                                 ReadHeaderTimeout 10s · IdleTimeout 120s · no read or write timeout
  ▼
routegroup root   (wraps the mux: runs for every request, 404 and 405 included)
  1 RequestID → 2 AccessLog (access line + duration histogram) → 3 Recover → 4 SecurityHeaders
  → 5 CORS (answers preflights of listed origins)
  │
  ├── public group (no middleware of its own)
  │     GET /healthz      GET /readyz
  │
  └── authed group: 6 Auth(Deps.Verifier)      answers before the body is read
        │   token = bearer(Authorization), "" when absent → Verifier.Verify(ctx, token)
        │     AUTH_PROVIDER=none      identity.DevVerifier        fixed principal
        │     AUTH_PROVIDER=keycloak  keycloak.Verifier           go-oidc → key set (lazy discovery) → replay transport (5 s) → Keycloak
        │     make test               keycloak.Verifier           → testprovider (httptest)
        │   *identity.Principal → identity.WithPrincipal(ctx)
        │   identity.ErrUnauthenticated → unauthenticated (401 / grpc 16) + WWW-Authenticate
        │   identity.ErrUnavailable     → unavailable     (503 / grpc 14)
        │
        ├── /api.v1.<Service>/<Method>      one route per procedure, registered by connecthttp.Mount
        │      connect.Server: 7 RequestLog → 8 Errors → 9 validate
        │        → internal/rpc/<domain>.Handler      owner = principal.Subject; proto ↔ domain type
        │        → internal/domain/<domain>.Store     rules, pgx.ErrNoRows → apperr
        │        → internal/db/gen                    every statement carries owner = @owner
        │        → Postgres
        ├── /api.v1.<Service>/              unknown method: 501 unimplemented, no interceptor runs
        └── plain routes added by mount files: BodyLimit(4 MiB) → handler

error path back:  Store returns *apperr.Error or a raw error → Handler returns it unchanged →
                  Errors maps Kind to a Connect code and attaches ErrorInfo{Reason, Domain: "api"} →
                  RequestLog writes procedure and code to the request-log carrier →
                  AccessLog reads the carrier, writes one line and observes the histogram

management http.Server on MANAGEMENT_ADDR (started only when set; no middleware, no auth)
  GET /metrics      GET /debug/pprof/…
```

The split between the two layers: HTTP middleware carries what needs only headers, applies to
every route, or must act before a body is read. Connect interceptors carry what needs the
procedure, the typed message or a Connect error. connect v2 has one interceptor type,
`connect.ServerInterceptor`, which wraps the whole call of any RPC, so every interceptor covers
unary and streaming by construction (F4).

### 3. Composition root, hub and mount files

`server.Handler(Deps)` is the composition root. `Deps` has four fields: the pool, the verifier,
the CORS origin list and an optional Prometheus registerer. `Handler` returns an error when the
pool or the verifier is nil. It is `build(d, routes)`; `build` takes the hub as a function so a
test can add routes. Steps, in order:

1.  `router := routegroup.New(http.NewServeMux())`;
    `router.Use(RequestID, AccessLog(d.Metrics, "/healthz", "/readyz"), Recover, SecurityHeaders, CORS(d.CORSOrigins))`.
2.  `public := router.Group()`; register `GET /healthz` and `GET /readyz`.
3.  `authed := router.Group()`; `authed.Use(middleware.Auth(d.Verifier))`. `Use` comes before any
    route on the group; routegroup panics otherwise.
4.  `plain := authed.With(middleware.BodyLimit(maxRequestBytes))`. It inherits auth and adds the
    body cap. This is the bundle the hub and every mount file receive.
5.  `rpc := connect.NewServer(RequestLog, Errors, validate.NewServerInterceptor())`.
6.  `routes(rpc, plain, d)`: the hub.
7.  `connecthttp.Mount(authed, rpc, connecthttp.WithReadMaxBytes(maxRequestBytes))`. Mount runs
    once and last: it reads the procedures registered at that moment and panics on a second
    call.

`maxRequestBytes` is 4 MiB, a constant in `server.go`. It is the per-message limit of every RPC
(answered `resource_exhausted`) and the body limit of every plain route. A stream has no total
limit. It is not an environment variable.

The hub is one function that calls one mount function per domain. Each mount function has the
same three parameters, `(rpc *connect.Server, plain *routegroup.Bundle, d Deps)`; unused ones
are named `_`. With per-function signatures, removing the demo line left the hub with unused
parameters and `unparam` failed; with the uniform signature the stripped hub lints clean.
Mount-file rules:

*   It builds the domain's store and handler from `d` and calls the generated
    `apiv1connect.Register<Service>Handler(rpc, h)`.
*   It may add plain routes with `plain.HandleFunc("GET /path", fn)`; they are behind auth and
    the body cap.
*   It never calls `Use`, `connecthttp.Mount` or `connect.NewServer`, never registers a
    method-qualified catch-all such as `"GET /"` (panics next to a Connect service), and never
    registers a path under `/api.v1.`.

Removing a domain is: delete its mount file and its hub line. A service without auth is a second
`connect.NewServer(...)` mounted on `public`; the template ships none, and adding one is an edit
to `server.go`. `connect.Server.SetUnknownHandler` must not be used: it would run the
interceptors for method names a caller chooses, and those names would become metric label
values.

### 4. HTTP middleware (`internal/server/middleware`)

All are `func(http.Handler) http.Handler` or return one. No enable flags, no options from the
environment. `go-pkgz/rest` is not used: two of its middlewares fit and would save 39 lines, but
their defaults are unsuitable for an API (CORS origin `*`, the retired `X-XSS-Protection`
header, no CSP).

| Middleware | Where | Behaviour |
|---|---|---|
| `RequestID` | root 1 | Keeps an inbound `X-Request-Id` that is 1 to 64 characters of letters, digits, `.`, `_`, `-`; otherwise generates 32 hex characters. Sets the response header before calling next. Creates the request-log carrier with the id. |
| `AccessLog(reg, skip...)` | root 2 | After the handler returns: one INFO record `request` with `method`, `path` (never the query string), `status`, `duration_ms` (integer milliseconds), and `code` when the carrier has one. When `reg` is not nil and the carrier has a code, one observation of the histogram `rpc_server_duration_seconds{procedure, code}` with both labels read from the carrier. No line and no observation for a path in `skip`. Its response writer implements `Flush` through `http.NewResponseController` and `Unwrap`; Connect type-asserts `http.Flusher`, and a writer with only `Unwrap` fails every server stream. |
| `Recover` | root 3 | Recovers a panic of a plain route, re-panics `http.ErrAbortHandler`, logs ERROR `panic` with the stack, answers a plain-text 500. A panic in an RPC handler is recovered by the `Errors` interceptor instead. |
| `SecurityHeaders` | root 4 | Constants: `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`; `Strict-Transport-Security: max-age=31536000` only when the request arrived over TLS or `X-Forwarded-Proto` is `https`. |
| `CORS(origins)` | root 5 | Exact match of `Origin` against the list. Match: `Access-Control-Allow-Origin`, `Vary: Origin`, exposed headers from `connectrpc.com/cors` plus `X-Request-Id`. A preflight from a listed origin is answered 204 here with the Connect method and header lists plus `Authorization`, and never reaches auth. No credentials, no wildcard. Empty list: does nothing. |
| `Auth(v)` | authed group | Section 7. |
| `BodyLimit(n)` | plain routes | `http.MaxBytesHandler`. Connect routes are not wrapped: an HTTP-level cap would bound a whole client stream. |

The histogram lives in `AccessLog` and not in an interceptor so that requests stopped by `Auth`
are counted. A request that never reached an interceptor has an empty `procedure` label: `Auth`
sets only the code. A request with no code on the carrier (a plain route, a 404, an unknown
method of a known service) is not observed, so a caller cannot create label values. The
duration includes the middleware inside `AccessLog`. A stream is observed once, when it ends.
Its `_count` series is the request count; there is no separate counter.

Order consequences, from the spike's routing and CORS tests: the request id is on every
response and log record; security headers are on 404, 405, 401 and 500; a 401 for a listed
origin carries the CORS headers; a preflight from an unlisted origin falls through to auth and
gets 401; auth does not run for a path that matches no route.

### 5. Server, protocols, timeouts

`server.New` sets `Protocols` (HTTP/1.1 and unencrypted HTTP/2), `ReadHeaderTimeout` 10 s,
`IdleTimeout` 120 s, and an `ErrorLog` that writes net/http's own messages as JSON records.
There is no TLS: it terminates at the proxy, which must speak h2c upstream for gRPC clients. One
port serves three protocols (Connect, gRPC, gRPC-Web).

`ReadTimeout` and `WriteTimeout` are 0. `WriteTimeout` is counted from the start of the
response and cuts a streaming RPC: in the spike a 1 s server stream under a 300 ms
`WriteTimeout` ended at 300 ms on HTTP/1.1 and on h2c. `ReadTimeout` covers reading the whole
request body, so it would bound a client stream the same way: the side probe saw a 1 s client
stream fail with both timeouts at 300 ms; that `ReadTimeout` alone cuts it is expected; not
run. Setting either would break the first streaming RPC a derived project adds, with no
warning.

What this leaves unbounded: the server sets no time limit on a handler, unary or streaming. On
HTTP/1.1 a `WriteTimeout` would not have bounded the handler either; it only fails the write
after the handler returns (the side probe saw a 700 ms handler under a 300 ms `WriteTimeout`
fail after the handler ended). A handler ends when it returns, when the client's deadline
(`Connect-Timeout-Ms`, `grpc-timeout`) expires its context, when the client disconnects, or
when the shutdown grace runs out. `docs/production.md` states this and gives the recipe for a
project that adds a streaming RPC and wants a bound on everything else: set the two timeouts in
`server.New`, and register one HTTP middleware on the streaming procedure's path that clears
both deadlines through `http.NewResponseController` (the side probe ran this shape: with the
deadlines cleared, a 1 s server stream and a 1 s client stream completed under 300 ms
timeouts).

### 6. Connect interceptors and the error path

Order, outermost first: `RequestLog`, `Errors`, `validate`.

| Interceptor | Behaviour |
|---|---|
| `RequestLog()` | Sets `procedure` on the request-log carrier before the call and `code` after it. Outermost, so it records the code after `Errors` mapped it. `nil` is recorded as `ok`. |
| `Errors()` | Maps what the handler returned and recovers panics. connect v2 ships no recovery; recovered here, clients on every protocol get `internal: internal error`. |
| `validate.NewServerInterceptor()` | Innermost. Checks protovalidate rules on each received message; failure is `invalid_argument` with a `buf.validate.Violations` detail. |

Not seen by any interceptor: a request stopped by `Auth` (the middleware sets the code on the
carrier itself), and a call to an unknown method of a known service (Connect answers 501
`unimplemented`; the carrier has no code).

`Errors` mapping:

| Handler returns | Client gets | Log |
|---|---|---|
| `*apperr.Error` anywhere in the chain | code from `Kind`, its message, `google.rpc.ErrorInfo{Reason, Domain: "api"}` | ERROR `rpc failed` only for `apperr.Internal` |
| `*connect.Error` made in this process (validation errors included) | unchanged | none |
| `*connect.Error` with `IsRemote()` | `internal`, `internal error` | ERROR `rpc failed` |
| `context.Canceled` | `canceled` | DEBUG `rpc canceled` |
| `context.DeadlineExceeded` | `deadline_exceeded` | DEBUG `rpc deadline exceeded` |
| any other error | `internal`, `internal error` | ERROR `rpc failed` with the error |
| panic | `internal`, `internal error` | ERROR `rpc panic` with the stack |

Text a client can receive is limited to an `apperr` message, a message a handler put in its own
`connect.Error`, protovalidate violation text and the fixed strings above. Clients switch on
`ErrorInfo.reason`. The domain `"api"` is a constant; init has nothing to rewrite.

`internal/apperr` has its own `Kind` and no Connect import. Kinds map to Connect codes in the
interceptor, the only place that knows both vocabularies:

| `apperr.Kind` | Connect code | HTTP status of a Connect unary response |
|---|---|---|
| `Internal` (zero value) | `internal` | 500 |
| `InvalidArgument` | `invalid_argument` | 400 |
| `NotFound` | `not_found` | 404 |
| `AlreadyExists` | `already_exists` | 409 |
| `PermissionDenied` | `permission_denied` | 403 |
| `FailedPrecondition` | `failed_precondition` | 400 |
| any value outside the enum | `internal` | 500 |

`Unauthenticated` and `Unavailable` are not kinds: only the auth middleware produces them.
Domain packages declare sentinels and return them directly or wrapped with `%w`; a handler
never calls `connect.NewError` for a domain condition and never inspects a store error. A
reason is a stable `UPPER_SNAKE` string of the form `<NOUN>_<CONDITION>`.

`internal/reqlog` is the request-log carrier: a pointer to a mutex-guarded `{id, procedure,
code}` stored in the request context by `RequestID`. The pointer lets an interceptor or `Auth`
write a value that the outer `AccessLog` reads after the handler returned. Its slog handler
appends `request_id` and `procedure` to every record logged with the request context. It
carries no subject, token or payload.

### 7. Identity

`internal/identity` holds the provider-neutral principal (subject, username, email, name, realm
roles), the context helpers, `HasRole`, the two sentinels `ErrUnauthenticated` (the token is not
accepted) and `ErrUnavailable` (the token could not be checked), and `DevVerifier`. It imports
`context`, `errors` and `slices` only. There are no organizations, client roles, scopes, groups,
raw claims or raw token. `DevVerifier` returns a new principal per call with subject and
username `local-dev` and the configured roles; it ignores the token and never fails.

A role check is one line at the top of a handler:
`if !identity.FromContext(ctx).HasRole("admin") { return nil, apperr.New(apperr.PermissionDenied, "ADMIN_REQUIRED", "admin role required") }`.
There is no `Require*` middleware, no proto option and no interceptor for it.

**Auth middleware.** `middleware.Auth(v Verifier)` on the authed group:

*   The token is the part after a case-insensitive `Bearer ` scheme in the first `Authorization`
    value, otherwise `""`. Never from a query parameter, cookie or body.
*   `Verify` is always called, also with `""`. "No token" is the verifier's decision:
    `keycloak.Verifier` rejects it, `DevVerifier` accepts it. This makes the dev principal a
    second implementation of the same interface (B4).
*   `errors.Is(err, identity.ErrUnavailable)` → `unavailable`, one ERROR record. Any other
    error, or a nil principal → `unauthenticated`, one DEBUG record with the reason. Unknown
    errors fail closed.
*   It writes the code to the request-log carrier, so the access line and the histogram carry
    `unauthenticated` or `unavailable`.
*   The client learns two fixed messages only. The token, the header and claims are never
    logged. The subject is not added to the access line.
*   It runs once per HTTP request, before the body is read. A stream is authenticated when it
    opens and is not re-checked when its token expires.

Failures are written with one `connecthttp.NewErrorWriter()`, which picks the protocol from the
method and `Content-Type` and does not read the body. Auth on the wire, measured by the spike's
wire test on every row:

| Client | Bad or missing token | Provider unavailable | Where the error travels | `WWW-Authenticate` |
|---|---|---|---|---|
| Connect unary (`application/proto`, `application/json`) | HTTP 401 | HTTP 503 | JSON body `{"code":…,"message":…}` | on the 401 |
| Connect streaming (`application/connect+proto`) | HTTP 200 | HTTP 200 | end-stream message `{"error":{"code":…}}` | on the 200 of a bad or missing token |
| gRPC (`application/grpc`) | HTTP 200, `Grpc-Status: 16` | HTTP 200, `Grpc-Status: 14` | HTTP trailer | on the 200 of a bad or missing token |
| gRPC-Web (`application/grpc-web+proto`) | HTTP 200, `Grpc-Status: 16` | HTTP 200, `Grpc-Status: 14` | HTTP header | on the 200 of a bad or missing token |
| plain route on the authed group | HTTP 401 | HTTP 503 | Connect JSON body | on the 401 |

`WWW-Authenticate` is never set for `unavailable`. Its value is `Bearer` when the request had
no bearer token and `Bearer error="invalid_token"` when it had one. The spike sent `Bearer` in
both cases; the second form is expected; not run.

**Keycloak verifier.** `internal/identity/keycloak` wraps go-oidc. `New(issuer, audience)` makes
no request and cannot fail. Checks, in order:

| # | Check | By |
|---|---|---|
| 0 | token not empty | template, before go-oidc, no request |
| 1 | JWS shape and `alg` in the fixed list RS256/384/512, ES256/384/512, PS256/384/512, EdDSA; `HS*` and `none` refused before any request | go-oidc |
| 2 | signature, key chosen by `kid`, before any claim | go-oidc |
| 3 | `iss` equal to the configured issuer | go-oidc |
| 4 | `aud` contains the configured audience | go-oidc |
| 5 | `exp` present and not passed; no leeway | go-oidc |
| 6 | `nbf` when present, fixed 5 min leeway | go-oidc |
| 7 | token type: claim `typ` must be `Bearer`; when absent, header `typ` must be `at+jwt` or `application/at+jwt`. Rejects ID, refresh, offline, logout and DPoP-bound tokens | template |
| 8 | `sub` non-empty | template |

Not checked: `azp`, `iat`, JWK `use`. Claims map to the principal: `sub`, `preferred_username`,
`email`, `name`, `realm_access.roles`; a missing `realm_access` gives no roles; nothing is
filtered. There is no expiry leeway because go-oidc has none; API hosts need synchronized
clocks.

*   **Discovery** is lazy and retried. The package's own key set runs `oidc.NewProvider` on first
    use and on every later attempt until it succeeds, then keeps one `oidc.RemoteKeySet` for the
    `jwks_uri` it found. Discovery uses `context.Background()` with a 5 s client timeout, so one
    caller hanging up cannot fail the fetch others wait for.
*   **Outage versus bad token.** go-oidc flattens every `Verify` error to text. `Verify` puts a
    `*error` in the context; the key set fills it when discovery fails or a key fetch fails.
    Filled → `ErrUnavailable`; otherwise `ErrUnauthenticated`. No string matching. The verifier
    tests pin this against a go-oidc upgrade.
*   **Key caching.** go-oidc keeps the last JWKS with no TTL, refetches when no cached key
    verifies a token, and keeps the old keys when a fetch fails. Tokens signed by a cached key
    verify through an outage; a new key is picked up by the first token that carries it.
*   **Refetch guard.** Without a guard every forged token costs one JWKS request (measured: 20
    for 20). The `replay` round-tripper answers a GET for a URL it fetched under 5 s ago from
    memory, failures included, and reads at most 1 MiB. Result: at most one outbound request per
    URL per 5 s whatever the inbound load.
*   **Readiness latch.** `Ready` reports whether this verifier has completed discovery and
    loaded the provider's keys at least once. While it has not, each call tries: it runs
    discovery, then asks the key set to verify a fixed, well-formed JWS that no key signs, which
    makes go-oidc fetch and cache the JWKS. Once both have succeeded the result is latched: it
    never resets, and later calls return without I/O. `/readyz` and startup call it (section 9).
*   Both 5 s values are constants. There is no timeout, clock-skew or JWKS-URL setting. The
    issuer URL must equal `iss` in tokens and be reachable from the server as written.

Three limits of this wrapper, also stated in `docs/auth.md`:

1.  **A key removed at the provider stays trusted.** go-oidc has no key TTL. A replica drops a
    removed key only when it restarts or when it next refetches, and it refetches only when a
    token arrives that no cached key verifies. After deleting a signing key in Keycloak, restart
    every API replica.
2.  **Refetch stall.** The replay transport holds one lock across an outbound fetch. While the
    provider hangs, each request that needs a fetch waits up to 5 s per window, and recovery
    after an outage and pickup of a rotated key under forged traffic are up to 5 s late.
    Unauthenticated callers can trigger the wait. Each such request writes one ERROR record, so
    during an outage a caller sending unverifiable tokens controls the ERROR log volume.
3.  **No update channel.** The key-set wrapper, the fetch-failure out-parameter, the replay
    transport and the token-type rule are about 180 lines of template code. They are copied into
    every derived project and receive no updates from the template. The outage split depends on
    go-oidc wrapping only key-fetch failures inside the key set; the outage tests fail if an
    upgrade changes that.

`internal/identity/testprovider` serves discovery and one RSA key from an `httptest.Server` and
signs with the standard library. It imports neither go-oidc nor the verifier, so the verifier is
tested against independent code. It has no authorize, token, userinfo or logout endpoint.

### 8. Configuration and logging

One flat struct, one reader, no library. `Load` builds a lookup: process environment first, then
`.env.local`, then the built-in default where one exists. An empty value counts as unset. `Load`
never calls `os.Setenv`. `Load` (syntax) and `Validate` (presence and cross-variable rules) each
collect every violation and return them joined, one per line, each naming its variable. A
malformed value never falls back to a default.

The complete set of variables the binary reads:

| Variable | Type | Built-in default | Required | Syntax and meaning |
|---|---|---|---|---|
| `APP_ENVIRONMENT` | string | `production` | no | Trimmed, lower-cased. Only use: gate for `AUTH_PROVIDER=none`. `development`, `dev`, `local`, `test` form the dev allow-list; everything else is a deployed environment. |
| `AUTH_PROVIDER` | enum | none | by `run` | `keycloak` or `none`, exact. |
| `AUTH_DEV_ROLES` | comma list | empty | no | Roles of the dev principal. Read only under `none`. |
| `KEYCLOAK_ISSUER_URL` | URL | none | when `AUTH_PROVIDER=keycloak` | Absolute `http` or `https` URL with a host, no query, fragment or trailing `/`. Must equal `iss` in tokens. |
| `KEYCLOAK_AUDIENCE` | string | none | when `AUTH_PROVIDER=keycloak` | Required audience of access tokens. |
| `DATABASE_URL` | URL | none | by `run` and `db migrate` | Scheme `postgres` or `postgresql`. Pool size and lifetimes are pgxpool's `pool_*` query parameters. |
| `LISTEN_ADDR` | `host:port` | `:8080` | no | `net.SplitHostPort` succeeds, port 1–65535. |
| `MANAGEMENT_ADDR` | `host:port` | empty = no management listener | no | As `LISTEN_ADDR`; must differ from it. |
| `CORS_ALLOWED_ORIGINS` | comma list | empty = no origin allowed | no | Each item `scheme://host[:port]`, `http` or `https`, no path, no `*`. |
| `LOG_LEVEL` | enum | `INFO` | no | `DEBUG`, `INFO`, `WARN`, `ERROR`, case-insensitive. Applied to every record. |

The three connection values (`DATABASE_URL`, `KEYCLOAK_ISSUER_URL`, `KEYCLOAK_AUDIENCE`) have no
built-in default. A deployed binary that is missing one refuses to start; it cannot fall back to
a localhost value. The compose values are in no file the binary is built from. `.env.example`
sets all three. Dev tooling repeats them in three places, which a project that changes a
compose port or credential edits too: the `Makefile` default of `KEYCLOAK_ISSUER_URL` for
`make token`, the issuer fallback in the smoke test, and the database credentials that
`scripts/image-smoke.sh` passes to the container. There is no rule about the
issuer scheme: an `http` issuer inside a cluster is legitimate. `MANAGEMENT_ADDR` is empty by
default because the listener serves pprof without auth; metrics are off until an operator sets
it.

`Validate` rules:

1.  `AUTH_PROVIDER` unset → refused.
2.  `AUTH_PROVIDER=none` and `APP_ENVIRONMENT` not in the dev allow-list → refused, naming the
    allow-list. Because `APP_ENVIRONMENT` defaults to `production`, `none` with no environment
    set is refused; so are `prod`, `staging`, `Production` and typos.
3.  `AUTH_PROVIDER=keycloak` and `KEYCLOAK_ISSUER_URL` or `KEYCLOAK_AUDIENCE` unset → refused.
4.  `DATABASE_URL` unset → refused.
5.  `MANAGEMENT_ADDR` equal to `LISTEN_ADDR` → refused.

`db migrate` calls `Load` and requires only `DATABASE_URL`, so a migration job needs one
variable.

`.env.local` is read from the working directory; there is no project-root setting. A missing
file is not an error; a malformed line is. Format: `KEY=value`, `#` comments, blank lines, one
pair of matching quotes removed; no `export`, no interpolation. It only fills variables the
process environment leaves unset. The image contains no such file. Under `go test` the working
directory is the package directory, so the repository's file is not read.

`.env.example` is the dev env file, copied to `.env.local` by the Makefile. Every variable
appears once, set or commented. It sets the dev allow-list environment, `AUTH_PROVIDER=none`,
the dev role, the three compose connection values, and loopback addresses for both listeners,
so the dev API, where every request is admin, is not reachable from the network. No Go code
defaults to `none`: the value is set in this file and in the dev-environment case of
`scripts/image-smoke.sh`, nowhere else. A test keeps `.env.example` and
`docs/agents/env-reference.md` equal to the reader in both directions.

Removed: every other variable in today's `.env.example`, among them the seven
`SERVER_ENABLE_*_MIDDLEWARE` flags, nine `SERVER_SECURE_*` knobs, two logger levels and five
`SERVER_LOGGER_LOG_*` knobs, `SERVICE_PORT`, `PROJECT_ROOT_DIR`, six `PG*`, three `DB_*`,
`KEYCLOAK_HTTP_TIMEOUT_SEC`, `KEYCLOAK_CLOCK_SKEW_SEC`, `KEYCLOAK_ISS`, `KEYCLOAK_CLIENT_ID`, four
`KEYCLOAK_ADMIN_*`, four pprof and management-secret variables, and `AUTH_DEV_SUBJECT` (fixed to
`local-dev`).

Logging: JSON to stdout through `log/slog`, one global logger wrapped by the `reqlog` handler,
call sites use `slog.*Context`, keys are snake_case. Installed at INFO before the configuration
is read and again with `LOG_LEVEL`. No text handler, no per-environment switch, no header or
body logging. Access line of an RPC:

```text
{"time":"…","level":"INFO","msg":"request","method":"POST","path":"/api.v1.WhoAmIService/WhoAmI","status":200,"duration_ms":1,"code":"ok","request_id":"…","procedure":"/api.v1.WhoAmIService/WhoAmI"}
```

A failed stream or gRPC call has `status: 200` and the failure in `code`; the line carries both.

### 9. Commands and process lifecycle

`Execute` installs the logger, builds a context cancelled by the first SIGINT or SIGTERM (a
second signal kills the process), runs the root command, and on error logs ERROR `command
failed` and exits 1. The root command takes its name from `os.Args[0]`, so no Go file holds the
project name. The version string is one variable in `cmd/root.go`, set at link time.

| Command | Does | Needs |
|---|---|---|
| `run` | Starts the server. Never migrates. | a configuration that passes `Validate` |
| `db migrate up` / `down` / `status` | Runs the embedded migrations through `db.Migrate`. Calls `Load`, not `Validate`. | `DATABASE_URL` only |
| `probe` | `GET /readyz` on the host and port of `LISTEN_ADDR` (an empty or unspecified host becomes `127.0.0.1`), 2 s timeout, exit 0 on 200, else 1. It is the image `HEALTHCHECK`: the image has no shell and no curl. | `LISTEN_ADDR` |

`run`, in order:

1.  `Load` and `Validate`. A failure exits 1 before any network I/O.
2.  `db.Open` with a 10 s timeout; `defer pool.Close()`.
3.  `db.CheckSchema`.
4.  Verifier. `none`: WARN `authentication disabled: every request runs as the dev principal`
    with `subject`, `roles`, `app_environment`; the verifier is `identity.DevVerifier`.
    `keycloak`: `keycloak.New(issuer, audience)`, then `Ready` once: nil → INFO `identity
    provider ready`; `ErrIssuerMismatch` → exit 1 printing both URLs, because the configured
    issuer can never work; any other error → WARN `identity provider not reachable at startup`,
    the server starts, `/readyz` answers 503 and requests get `unavailable` until `Ready`
    succeeds.
5.  A Prometheus registry with the Go and process collectors, when `MANAGEMENT_ADDR` is set.
6.  `server.Handler(Deps{...})`.
7.  Bind both listeners before serving; a bind failure exits 1.
8.  INFO `server started` with `addr`, `management_addr`, `environment`, `auth_provider`, `version`.
9.  `server.Serve(ctx, 25*time.Second, servers)`.
10. INFO `server stopped`; return Serve's error.

**Shutdown.** On SIGTERM the listeners stop accepting, requests in flight get the 25 s grace
period, and anything still open at the deadline, streams included, is cut. The process exits 0
when everything drained inside the grace period and 1 otherwise. There is no mechanism that
tells a stream to end early. `pool.Close()` runs after the last request. The grace fits inside
Kubernetes' default 30 s termination period. Listener close is not delayed in code; under
Kubernetes, endpoint removal lags SIGTERM, and `docs/production.md` documents a `preStop`
`sleep` for that window (the `sleep` action needs no shell in the image).

**Probes.** `GET /healthz` answers 200 with an empty body and checks nothing. `GET /readyz`
answers 200 with an empty body when both hold: the database answers a ping within 2 s, and the
verifier is ready. A verifier is ready when it has no `Ready` method (`DevVerifier`) or its
`Ready` returns nil (the latch of section 7). Failure is 503 with a fixed body, `database
unavailable` or `identity provider unavailable`, and a WARN record with the real error. Because
the latch never resets, an identity-provider outage after start does not take replicas out of
rotation: they keep serving tokens signed by cached keys. A replica started during an outage
stays out of rotation until it has keys, so a rolling restart cannot replace replicas that
serve with replicas that cannot. Both probes are outside auth and write no access line.

**Management.** `server.Management` serves `GET /metrics` and the `net/http/pprof` handlers on
its own mux with no middleware and no auth. The API port answers 404 for both. This replaces
the pprof secret in the query string.

### 10. API contract and code generation

*   `proto/api/v1/<domain>.proto`, one service per file, all in `package api.v1`. The proto
    package is neutral and permanent; init does not rename it. No `go_package` option; buf
    managed mode sets it. Messages are `<Rpc>Request` / `<Rpc>Response`. Read-only RPCs carry
    `idempotency_level = NO_SIDE_EFFECTS` (Connect then also serves them on GET). Timestamps are
    `google.protobuf.Timestamp` named `create_time` / `update_time`; ids are `string`.
*   `whoami.proto` is permanent: one RPC that returns the caller's principal, no validation
    import. It replaces `GET /v1/users/me`.
*   `notes.proto` is the demo: four RPCs with protovalidate rules (section 12).

buf runs in v2 managed mode with one generated-package prefix, `<module>/internal/gen`, which is
the one line of `buf.gen.yaml` init rewrites. The two plugins run as `go tool` and are pinned by
`go.mod`. `buf.lock` is committed. `buf breaking` is configured (`FILE` rule set) and not run by
any target or CI step: a prototype's API is expected to break; `go-feature` names the command.

| Source | Tool | Output (committed, never edited) |
|---|---|---|
| `proto/api/v1/*.proto` | `buf generate` with the two `go tool` plugins | `internal/gen/api/v1/<file>.pb.go` (package `apiv1`), `internal/gen/api/v1/apiv1connect/<file>.connect.go` |
| `sql/queries/*.sql` + `sql/migrations/*.sql` | `sqlc generate` | `internal/db/gen/{db.go,models.go,<file>.sql.go}` |

Nothing hand-written lives under `internal/gen` or `internal/db/gen`: buf's `clean: true`
deletes everything under `internal/gen` on each run, and `make sqlc` deletes `internal/db/gen`
before it regenerates, because sqlc never removes the output of a deleted query file.

connect v2 facts the code builds on, each checked in the spike or with `go doc`:
`Register<Service>Handler(server *connect.Server, svc)` has no return value; handler methods are
`(ctx, *Req) (*Resp, error)`; clients are `New<Service>Client(*connect.Client)` over
`connecthttp.NewTransport`; errors are `connect.NewError(code, message)`; there is no
`WithInterceptors`, `UnaryInterceptorFunc` or `WithRecover`. `docs/agents/stack-notes.md`
carries the full list for agents.

### 11. Persistence

`internal/db` has three functions: `Open`, `Migrate`, `CheckSchema`.

*   `Open` creates a `pgxpool.Pool` and pings it. It sets nothing in code: the pool uses pgxpool
    defaults, and `pool_max_conns` and the other `pool_*` query parameters of `DATABASE_URL`
    change them. The default maximum is the greater of 4 and the CPU count, which is 4 in a
    typical small container. `/readyz` pings through the same pool. `Deps.DB` is the pool; there
    is no wrapper type.
*   `Migrate` opens its own `*sql.DB` through `pgx/v5/stdlib` (goose needs one), builds a goose
    provider over the embedded files with goose's Postgres session locker, runs the command and
    closes the connection. Version table: goose's default `goose_db_version`. The locker makes
    two concurrent `db migrate up` runs against one database wait for each other on a session
    advisory lock (the option exists in goose v3.30.1; the behaviour is expected; not run). With
    zero migration files `goose.NewProvider` returns `goose.ErrNoMigrations`; `Migrate` prints
    `no migrations`, pings and returns, so `up`, `down` and `status` all succeed and no version
    table is created.
*   `CheckSchema` compares the highest embedded version with the highest applied version in
    `goose_db_version` (table missing = 0). With no embedded migration it returns nil without a
    query. A schema older than the binary is an error that names both versions and `db migrate
    up`. A newer schema passes, so a binary rollback can run. It is read-only and reads goose's
    table directly, because `run` must not write schema and every goose provider method creates
    the version table.
*   No transaction helper: a multi-statement use case calls `pgx.BeginFunc(ctx, pool, fn)` in
    the domain package. A wrapper in `internal/db` that handed out generated queries would have
    to import `internal/db/gen`.
*   `sql/embed.go` embeds the migrations directory with `all:`, so the directive compiles with
    only `.gitkeep` present.
*   Migrations are `sql/migrations/NNNNN_snake_name.sql`, five digits, both goose sections,
    written by hand. No goose CLI is installed.
*   sqlc generates for `pgx/v5` with `uuid` and `timestamptz` mapped to `uuid.UUID` and
    `time.Time`. There is no generated interface: tests use Postgres, not a fake.

### 12. Demo slice, markers and the stripped tree

The notes slice is today's demo ported at its current surface: `CreateNote`, `GetNote`,
`ListNotes`, `DeleteNote`. There is no pagination and no update; `ListNotes` returns the
caller's newest 100 notes. The slice consists of one proto file, one migration, one query file,
`internal/domain/notes` (a `Store` over the pool), `internal/rpc/notes` (a `Handler` that
implements the generated interface with no `Unimplemented` embed, so a new RPC fails the build
until it is implemented), one mount file and one hub line. Every statement carries `owner =
@owner`. A row of another owner is indistinguishable from a missing row: `not_found`, never
`permission_denied`.

Paging appears in one place only, and on purpose: a note in the `go-feature` contract reference
names the conventional request and response fields. No token format is prescribed, and the tree
holds no paged query and no paged example.

Marker contract. This table is the only statement of the forms:

| Form | Where | Effect of the strip |
|---|---|---|
| Line 1 is exactly `// DEMO: notes`. In Go, line 2 is blank so the marker stays out of the package doc. | every Go and proto file that belongs to the demo, tests included | file deleted |
| Line 1 is exactly `-- DEMO: notes` | every SQL file that belongs to the demo | file deleted |
| `// DEMO:line` at the end of a line | a shared file; only the hub line | line deleted |
| generated output of a demo source | `internal/gen`, `internal/db/gen` | never scanned; regenerated |

Init matches the forms, not the domain name: line 1 against `^(//|--) DEMO: `, and lines that
end with `// DEMO:line`. A demo file is a whole file; no non-demo file imports a demo package;
demo tests live in demo-marked files. The demo's documentation is one demo-marked file,
`internal/domain/notes/doc.go`: its package comment lists the files of the slice, the marker
forms and the manual removal steps. Permanent docs and skills never name a demo path or
identifier, so nothing dangles after the strip (B6).

State of the tree:

| Thing | Template tree and keep-demo tree | Stripped tree |
|---|---|---|
| `proto/api/v1/` | `whoami.proto`, `notes.proto` | `whoami.proto` |
| `internal/gen/api/v1/` | whoami and notes output | whoami output |
| `sql/migrations/`, `sql/queries/` | `.gitkeep` + the notes file | `.gitkeep` |
| `internal/db/gen/` | `db.go`, `models.go`, `notes.sql.go` | absent |
| `internal/domain/`, `internal/rpc/notes/` | present | absent |
| `internal/server/routes.go` | two mount lines | the `mountWhoAmI` line |
| `make gen` | `buf generate`, `sqlc generate` | `buf generate`; prints `sqlc: no query files, skipped` |
| `make build`, `make lint` | pass | pass: `all:` embeds `.gitkeep`; nothing imports `internal/db/gen`; uniform mount signature |
| `db migrate up` / `down` / `status` | goose output | `no migrations`, exit 0 |
| `run` | `CheckSchema` compares versions | `CheckSchema` returns nil without a query |
| `go.mod` | `uuid` and the protovalidate generated module are direct requires | `go mod tidy` moves `uuid` to indirect (run); the protovalidate generated module is expected to follow (expected; not run) |

No placeholder table, query or migration ships. After the strip no single request reaches a
table: `WhoAmI` reads no data, and Postgres is exercised through the harness (container,
migrate, clone, pool) and the `/readyz` ping. The failing path of the validation interceptor has
no test after the strip, because no permanent message has a rule.

`--keep-demo` is final as far as init is concerned. The demo migration is version 1 of every
database it was applied to; removing the demo later means deleting the marked files and the hub
line by hand, regenerating, and either recreating the database or writing a migration that
drops the table. Init prints this when `--keep-demo` is used. For the same reason the template
README tells a user who ran `make migrate-up` on the uninitialised template to run `make
dev-down` before init: init renames the compose project, and the old `app` project's Postgres
container would otherwise keep port 5434.

### 13. Test harness and shipped tests

Three kinds of test exist: **integration tests** in `make test`; the Keycloak **smoke** test in
`make smoke`; browser e2e tests are not in this template.

`internal/test` is the harness:

*   One Postgres container per test process (`postgres:17-alpine`; the tag is one constant with
    the same major as compose; fsync off). After start it creates the database `app_template`
    and runs the production `db.Migrate` on it. `Database(t)` clones the template into a fresh
    database and drops it in cleanup; `EmptyDatabase(t)` creates one with no migration applied.
    No mutex is needed around the copy (160 concurrent copies succeeded on Postgres 17). Docker
    is a prerequisite of `make test`; there is no `-short` gate.
*   `NewHandler` builds `server.Deps` with a pool on a fresh database, the real go-oidc verifier
    pointed at an in-process `testprovider`, and a metrics registry, then calls the production
    `server.Handler`. `NewServer` serves it on HTTP/1.1 and h2c through `httptest`. `NewDeps`
    returns the same `Deps` without building a handler; the `internal/server` suite uses it to
    build the stack with a hub that adds its test procedures.
*   `Client` adds the token in a `RoundTripper`; its options select one of four client
    configurations (Connect, Connect JSON, gRPC, gRPC-Web).
*   Tests that import `internal/test` are in external packages, because `internal/test` imports
    `internal/server`.

Permanent tests, present in every tree: `TestWhoAmI` (`/readyz` against Postgres, a generated
client with a minted token, no token); the `internal/server` suite, which registers its own
unary, server-stream and client-stream test procedures by hand with `rpc.Register`, so it needs
no `.proto` file and does not mention the demo; the middleware, interceptor, reqlog, config,
cmd, identity, keycloak, `internal/db` and harness tests. Demo tests, deleted with the slice:
`internal/rpc/notes/notes_test.go` and `internal/domain/notes/notes_test.go`. Bidi streaming is
not tested.

The smoke test (`internal/test/smoke_test.go`, build tag `smoke`) starts the real handler with a
verifier pointed at compose Keycloak, obtains tokens from the shipped realm and calls `WhoAmI`
in six cases. `.golangci.yml` sets the `smoke` build tag so the file is type-checked without
Keycloak.

### 14. Init (`scripts/init.sh`)

Init stays in bash: the script exists today, Go would have to exclude its own source from the
rename, and the `macos` leg of `template-smoke.yml` covers bash 3.2 and BSD userland. It uses no
`sed -i` and no python. It stays in the derived project as an idempotent repair tool: a
failed run is completed by running it again, and a second run with the same arguments changes
no file.

Init substitutes two values and no others:

| Value | Template value | Where it is a literal |
|---|---|---|
| module path | `github.com/driif/go-vibe-starter` | `go.mod`, imports in `*.go` outside generated directories, `buf.gen.yaml` (one line), `README.md`. `Makefile` and `Dockerfile` use `$(go list -m)`, so neither holds it. |
| project name | `app` | `Makefile` line `APP_NAME := app`; `docker-compose.yaml` line `name: app`. Binary path, image name and compose network derive from these. |

Neutral and never rewritten: database, role and realm `app`, audience `api`, proto package
`api.v1`, `ErrorInfo` domain `api`, the metric name, the test template database, the binary
inside the image (`/app`).

Steps:

1.  Parse flags, read state, prompt when stdin is a terminal and `--yes` is absent.
2.  Validate the module path and the name before anything changes.
3.  Preflight: a git work tree with the expected root files; `git`, `go`, `make` present; `make
    tools-check`.
4.  **First run only.** A first run is one where `scripts/template-smoke.sh` exists. Delete the
    other template-only files (`.github/workflows/template-smoke.yml`, `SECURITY.md`, `LICENSE`,
    `docs/designs/`), delete `docs/plans/**/*.md` and `docs/adr/*.md`, replace `README.md` with
    a stub. Unless `--keep-demo`: strip by markers (section 12). Delete
    `scripts/template-smoke.sh` last, so an interrupted first run repeats this step.
5.  **When the module path changes.** Rewrite the files that contain the old path, excluding
    the generated directories, with a boundary-aware substitution. `go mod edit -module` runs
    last, so an interrupted run still finds the old path.
6.  **When the name changes.** Rewrite the two anchored lines; a missing anchor is a failure.
7.  **Always.** `make gen` → `go mod tidy` → `make sync-skills` → `go build ./...` →
    `go vet ./...` → `make build`.
8.  **Always.** Residue check over all files except `scripts/init.sh`: the old module path when
    the path changed, and, when no demo-marked file is left, the string `DEMO:` and the path of
    every file the strip deleted in this run. Fails with `file:line`. Init does not search for
    a single element of the old path, such as the owner: in a later rename that element is an
    arbitrary word (`platform`, `google`) that occurs in unrelated lines.
9.  Print the result and the next steps. Init runs no git write command and creates no labels.

Init deletes `LICENSE` because the file would declare every derived project MIT under the
template author's name; the next steps say to add one. The template README states that the
author asks nothing of generated projects. There is no origin record: a derived project has no
update mechanism, and nothing would read one. Init contains no demo domain name and no template
name; the literal checks for both are in the template-only `scripts/template-smoke.sh` (section 18).

### 15. Tools, Makefile, the check gate

| Tool | Pin lives in | Installed by | Invoked as |
|---|---|---|---|
| `protoc-gen-go`, `protoc-gen-connect-go` | `go.mod` `tool (...)` block (adds no require) | nothing | `go tool ...` from `buf.gen.yaml` |
| buf, sqlc, golangci-lint, govulncheck | four `*_VERSION` variables at the top of the `Makefile`; these are the only pins for these four tools | `make tools` into `bin/tools`, one stamp file per tool and version | by full path |

`make tools-check` tests the stamps; the targets that need a tool depend on it. Nothing installs
implicitly. No goose CLI, no goimports binary (`golangci-lint fmt` runs both formatters). sqlc
links its Postgres parser through cgo, so a C compiler is a prerequisite. The spike ran sqlc
v1.30.0; the committed output is regenerated with the pinned v1.31.1 (expected; not run). The Go
version is stated in `go.mod` and in the `Dockerfile` (`golang:1.27`); CI reads `go.mod`, with
the caveat in Open Questions.

`make check` is the gate and the single "done" criterion in `AGENTS.md` and every skill. In
order, stopping at the first failure:

1.  **Drift.** Copy `go.mod`, `go.sum`, `internal/gen` and `internal/db/gen` to a temporary
    directory, run `make gen` and `go mod tidy`, and compare the tree with the copies. The check
    reads no git state, so it passes right after init and fails in CI when a commit lacks
    regenerated output.
2.  `make lint`: `golangci-lint run`, `buf lint`, `buf format -d --exit-code`.
3.  `make test`: `go test -race ./...`.
4.  `scripts/sync-skills.sh --check`: the mirror equals `.claude/skills`.

`vuln` is not in `check`: it needs the network and fails on advisories unrelated to the change.
Skills and docs name Make targets, never `bin/app`, because the binary name changes at init.
`make dev-up` starts Postgres only; Keycloak starts with `make keycloak-up` or `make smoke`.

`.golangci.yml` keeps `unparam` and adds three linters that turn written conventions into
checks: `sloglint` (snake_case keys, no mixed arguments), `forbidigo` (`fmt.Print*`, `print`,
`println`; `fmt.Fprint*` to a writer stays allowed) and `errorlint`.

### 16. Container image

Two stages: `golang:1.27` builds a static binary (`CGO_ENABLED=0`, `-trimpath`, the version set
at link time through `$(go list -m)`); `gcr.io/distroless/static-debian12:nonroot` holds only
`/app`. The user is numeric (65532), the health check is `/app probe`, the entrypoint is `/app`
and the default command is `run`. Migrations are inside the binary, so the same image runs `db
migrate up` when the command is overridden. Only port 8080 is exposed; the management port is
not. No `.env*` file is in the image; `.dockerignore` is an allow-list.

`scripts/image-smoke.sh` backs the image claims with two cases: started with no environment the
image exits non-zero and names `AUTH_PROVIDER is required`; started with a dev environment
against the compose Postgres it migrates and becomes healthy, which means `/app probe` got 200
from `/readyz`. It also checks the image user and the version stamp. It obtains no token.

### 17. Dev compose and realm

`docker-compose.yaml` is dev only: throwaway credentials, Keycloak in `start-dev` mode, ports
bound to 127.0.0.1. Two services:

*   **postgres**: `postgres:17-alpine`, database, user and password `app`, host port 5434, a
    named volume, a `pg_isready` health check.
*   **keycloak**: `quay.io/keycloak/keycloak:26.8.2`, `start-dev --import-realm
    --http-port=8181`, host port 8181, the realm directory mounted read-only, a health check on
    the management port 9000 inside the container. `KC_HOSTNAME` as one full URL fixes the
    issuer to `http://localhost:8181/realms/app`, the value in `.env.example`.

Published compose ports are 5434 and 8181. There is no `container_name`, no bind-mounted data
and no custom network: the compose project name, set by init, separates the containers, network
and volume of two derived projects. Host ports are literals; two stacks running at the same
time need different ports, and the second project edits the literals in the compose file and
in the places that repeat them (section 8). No compose profile is used, so `make dev-down`
stops everything. Keycloak
has no database and no volume: `make keycloak-up` recreates the container, which imports the
realm file again, so the running realm always equals the file.

`keycloak/realms/app.json` is hand-written, about 100 lines, and replaces the 87 KB export,
which had no `oidc-audience-mapper` that adds `api` and shipped private keys, a client secret
and password hashes. It defines realm `app`, realm roles `admin` and `user`, three clients
(`app-web` for a browser, `app-service` for client credentials, `app-test` for `make smoke` and
`make token`), each with one audience mapper for `api`, and users `alice` and `bob`. It has no
`clientScopes`, `components` or `id` fields, so Keycloak creates the built-in scopes and its own
keys. The audience `api` is not the id of any client, so no ID token can carry it. `app-web`
stays although nothing in the template uses it: it is one client entry and the first thing a
separately started frontend needs. Init leaves the file alone: realm `app` and audience `api`
are neutral.

### 18. CI

All workflows: `permissions: contents: read`, a concurrency group, actions pinned by commit
SHA, triggers on pull requests, pushes to `main`, a weekly schedule and manual dispatch. Lint,
buf, govulncheck and the image build go through Make, so no workflow repeats a tool pin.

| Workflow / job | Ships to derived projects | Runs | Proves |
|---|---|---|---|
| `ci.yml` / `check` | yes | `make tools`, `make check`, `make vuln` | build, lint, buf lint and format, tests on Postgres, codegen and tidy drift, skills mirror, no known vulnerability |
| `ci.yml` / `image` | yes | `make image-smoke` | the image row of the ships table (section 20), including the fail-closed start |
| `keycloak.yml` / `smoke` | yes | `make smoke`, on changes to the realm, compose file, identity packages, auth middleware and harness, and weekly | realm, audience and token type against real Keycloak |
| `template-smoke.yml` / `init` | no | `scripts/template-smoke.sh`, three legs | the tree after init builds, lints and tests, and carries no residue |

`template-smoke.yml` is guarded by the template's repository name and deleted by init together
with its script. The script is the single implementation and runs locally too. One command per
leg:

| Leg | Runner | Command |
|---|---|---|
| `stripped` | ubuntu | `scripts/template-smoke.sh --module example.com/acme/smoke-svc --name smoke-svc` |
| `keep-demo` | ubuntu | `scripts/template-smoke.sh --module git.example.org/platform/teams/billing-service --name billing-service --keep-demo` |
| `macos` | macOS, bash 3.2 | `/bin/bash scripts/template-smoke.sh --module example.com/acme/smoke-svc --name smoke-svc --no-docker` |

The first module path is shorter than the template's and the second is longer, which catches
descriptor corruption in generated code. The script:

1.  Copies the index into a temporary directory and commits it there. This reproduces the file
    set that "Use this template" copies; that GitHub's copy also keeps file modes is expected;
    not run. The script passes its own author identity to every commit, because a hosted
    runner has none configured.
2.  Runs `make tools`.
3.  Runs init with `--yes` under the same bash; commits.
4.  Runs an independent residue check with literals, so a bug in init's own check is caught:
    the template module path and the owner name are absent; the template-only files are gone;
    without `--keep-demo`, `DEMO:` and the demo identifiers are absent; and the banned old-stack
    terms of the skills pack (section 19) are absent.
5.  Runs `make check`; the tree is then clean. With `--no-docker`: generate, tidy, build, vet
    and lint only.
6.  Runs init again with the same arguments; the tree is clean.
7.  Runs `make image-smoke` (not with `--no-docker`).

Step 7 starts the compose Postgres of the temporary project on host port 5434, and
`image-smoke` leaves it running. The script therefore removes that compose project, volume
included, when it exits, whether it passed or failed. It refuses a `--name` equal to the
template's own project name, so the teardown cannot reach the template tree's dev volume. Two
consequences for a local run: port 5434 must be free, so `make dev-down` runs in the template
tree first; and the legs run one after the other, not at the same time. In CI each leg has its
own runner.

Dependabot covers `gomod`, `github-actions`, `docker` and `docker-compose`. Not visible to it
and bumped by hand: the four Makefile pins, the `toolchain` line, the Postgres tag constant in
the harness.

### 19. Docs and skills pack

Rules the rewrite follows:

*   **One place for the layout.** `AGENTS.md` §3 is the only place the tree is written down.
    `README.md` and `CONTEXT.md` link to it.
*   **One done criterion.** `make check`, in `AGENTS.md` and in every skill that names one.
*   **No demo names in permanent files.** `AGENTS.md`, `CLAUDE.md`, `CONTEXT.md`, `docs/` and
    the skills never name a demo path or identifier and do not use the word `DEMO`. The worked
    example they point at is the permanent `whoami` slice. `go-feature` carries static snippets
    for the persistence half, which `whoami` does not have.
*   **No template name in permanent files.** Titles lose the repository name; the `handoff`
    skill derives the name from the working directory.
*   **Skills may name stack terms and paths.** There is no rule that confines stack facts to
    some files and no check of paths named in prose. A statement is edited when it is wrong for
    the new stack.
*   **Init edits no skill.** `setup-project` treats the presence of `scripts/template-smoke.sh`
    as "not yet initialised".

The skills pack is changed in three ways: a short list of files is rewritten in full or
created; every other file with a stale statement is edited by one mapping table of old term to
new term; and `agents/skills/` is regenerated by `make sync-skills`. The mapping table, the
file lists and the acceptance grep are in stage 2 of the implementation plan. The grep runs
once by hand as the stage 2 gate and then in every run of `scripts/template-smoke.sh`, which is
template-only; derived projects do not inherit it.

Docs: `README.md` is rewritten for the template and replaced by a stub at init;
`docs/production.md` is rewritten around the ships table; `docs/auth.md` is new;
`docs/agents/request-path.md` and `docs/agents/stack-notes.md` are new agent references;
`docs/agents/env-reference.md` is the table of section 8. `scripts/sync-skills.sh --check`
compares `.claude/skills` with `agents/skills` and exits 1 on a difference.

### 20. What ships

This table replaces the phrase "near production ready". It is the "What you get" section of the
template README, and `docs/production.md` expands each row.

| Ships | Proven by | Documented, not shipped | Not included |
|---|---|---|---|
| Connect, gRPC, gRPC-Web and JSON POST on one port, HTTP/1.1 and h2c | `TestServe`, `TestErrors` | TLS and reverse proxy | frontend, browser e2e tests |
| Bearer auth before the body is read; refuses to start without `AUTH_PROVIDER`; `none` only on the dev allow-list | `TestAuth_AnswersBeforeBody`, `TestValidate`, `make image-smoke` | metrics backend, dashboards, tracing | REST / OpenAPI variant (see the tag) |
| Keycloak token verification; a provider outage answers `unavailable`; readiness needs keys once | `TestVerify`, `TestReady`, `make smoke` | deploy manifests; who runs migrations at deploy; the `preStop` drain | htmx variant |
| Request validation, error mapping, panic recovery | `TestErrors_Mapping`, `TestErrors`; validation failure only with the demo (`TestNotes_Validation`) | release automation, registry push, image scanning | authorization framework, tenancy |
| One JSON access line per request with request id, procedure, code | `TestAccessLog` | rate limiting | Keycloak admin client, LDAP in the app |
| pgx pool, embedded migrations, `db migrate up` / `down` / `status`, schema check at start | `TestMigrate`, `TestMigrate_Embedded`, `TestCheckSchema` | secret store, backups | blob storage, background jobs |
| `/healthz`, `/readyz`; graceful shutdown; 4 MiB request limit | `TestHealthz`, `TestReadyz`, `TestServe`, `TestBodyLimit` | read and write timeouts; a time bound for streaming RPCs | a streaming RPC example |
| CORS allow-list, default none | `TestCORS` | browser login (the frontend's code flow) | an update mechanism for derived projects |
| RPC duration histogram and pprof on a separate listener, off by default | `TestMetrics`, `TestServe` | `buf breaking` against a base branch | |
| Static non-root image with version and health check | `make image-smoke` | | |
| CI: drift, lint, tests on Postgres, govulncheck, image smoke, Keycloak smoke; Dependabot | the workflows of section 18 | | |

### 21. Reference project: what is ported and what is not

Ported, stripped of tenancy: `Handler(Deps)` / `New` / `Serve`; the middlewares without toggles;
`identity` with the keycloak and test providers minus organizations; the `Errors` and
`RequestLog` interceptors; the per-process template-database Postgres harness with one role;
`db.Open` / `Migrate`; embedded migrations; health; the flat config reader; the Go stages of the
Dockerfile; `sync-skills.sh --check`; the proto-first `go-feature` and two-layer `go-middleware`
skills as the base for the rewrite.

Not ported: organizations, `X-Org`, org resolution, actor recording, `InOrg` / RLS, two database
roles, `versionGrant`; `internal/authz`, `internal/expr`, the proto permission option; blob
storage, plugins, runs, retention, imports, sheet, caluma, public links and their rate limiter;
SPA embedding, the node build stage, `site/`, TypeScript and docs codegen, Playwright; `deploy/`,
release-please, Trivy, load test, manual-only CI, extra skill mirrors. The auth interceptor is
not copied: authentication is rewritten as HTTP middleware. Its token verifier is not copied:
go-oidc replaces it. `apperr` is rewritten without the Connect import. The server-wide mount
function is replaced by one mount file per domain.

### 22. Dependencies and removals

Direct requires of `go.mod` after the refactor. Versions are the ones the spike resolved on
2026-11-20, except where noted:

| Module | Version | Status |
|---|---|---|
| `connectrpc.com/connect/v2` | v2.0.2 | added |
| `connectrpc.com/validate` | v0.9.0 | added |
| `connectrpc.com/cors` | v0.1.0 | added |
| `google.golang.org/protobuf` | v1.36.12 | added |
| `buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go` | v1.36.12-20260825204119-511051f7f437.2 | added; direct while the demo is present |
| `google.golang.org/genproto/googleapis/rpc` | v0.0.0-20261110000943-09e21b237f2c | added (for `errdetails.ErrorInfo`) |
| `github.com/go-pkgz/routegroup` | v1.6.2 | added |
| `github.com/coreos/go-oidc/v3` | v3.21.0 | added |
| `github.com/jackc/pgx/v5` | v5.13.0 | added |
| `github.com/prometheus/client_golang` | v1.25.0 | added; it brings seven modules that no other dependency needs (`client_golang`, `client_model`, `common`, `procfs`, `beorn7/perks`, `cespare/xxhash/v2`, `munnerz/goautoneg`) |
| `github.com/pressly/goose/v3` | v3.30.1 | raised from v3.27.0 |
| `github.com/stretchr/testify` | v1.12.1 | raised from v1.11.1 |
| `github.com/testcontainers/testcontainers-go` | v0.45.0 | raised from v0.41.0 |
| `github.com/google/uuid` | v1.6.0 | kept; direct while the demo is present |
| `github.com/spf13/cobra` | v1.10.2 | kept (not part of the spike) |

One indirect require is raised by hand: `github.com/go-jose/go-jose/v4` to v4.1.9 (go-oidc's
minimum is v4.1.4, which the spike ran; v4.1.9 is expected; not run). The `tool` block names
`google.golang.org/protobuf/cmd/protoc-gen-go` (v1.36.12) and
`connectrpc.com/connect/v2/cmd/protoc-gen-connect-go` (2.0.2); their versions are those of the
two runtime modules.

Removed requires: `github.com/go-chi/chi/v5`, `github.com/go-chi/cors`, `github.com/go-pkgz/rest`,
`github.com/lib/pq`, `github.com/getkin/kin-openapi`, `github.com/oapi-codegen/runtime`.

Tools and images:

| Pin | Value | Where |
|---|---|---|
| Go | `go 1.26.0`, `toolchain go1.27.4` | `go.mod` |
| Go build image | `golang:1.27` | `Dockerfile` |
| Runtime image | `gcr.io/distroless/static-debian12:nonroot` | `Dockerfile` |
| buf | v1.75.0 | `Makefile` |
| sqlc | v1.31.1 (raised from v1.30.0) | `Makefile` |
| golangci-lint | v2.15.0 (raised from v2.12.2) | `Makefile` |
| govulncheck | v1.10.0 (new) | `Makefile` |
| Postgres | `postgres:17-alpine` | `docker-compose.yaml`, harness constant |
| Keycloak | `quay.io/keycloak/keycloak:26.8.2` | `docker-compose.yaml` |

Removed pins: oapi-codegen v2.6.0, the goose CLI v3.27.0, `goimports@latest`.

Code and behaviour removed: `oapi/`, `internal/api/**`, `internal/server/errs`,
`internal/server/auth` (with the role, scope and organization guards), `internal/server/test`,
`pkg/**` (the template's own verifier, the admin client, `db`, `dotenv`, `tests`),
`cmd/db_seed.go`, `db migrate reset`, the `/v1/users` endpoints, the old realm export,
trailing-slash stripping, blanket `Cache-Control: no-store`, `X-XSS-Protection`, request and
response body logging, `cors.AllowAll()`, pprof on the API port and its query-string secret, the
version string in the health body, HTTP 400 for a malformed token (every token failure is 401),
`docs/superpowers/`.

## Alternatives Considered

Keeping chi and dropping routegroup are not listed: both are fixed requirements (Technical Plan,
"Fixed requirements"), stated there with their accepted cost.

### Alternative A: LDAP bind from the Go app

*   **Pros**: No identity provider to run.
*   **Cons**: The app would store or forward passwords and own sessions, lockout and password
    policy.
*   **Reason for Rejection**: Rejected by author: the cost is a second authentication code path
    with credential handling, for something Keycloak already does as user federation. Tokens
    keep their issuer, audience and claim layout, so the Go side does not change; `docs/auth.md`
    describes the Keycloak setup.

### Alternative B: One template with init-time variant selection (REST / Connect / SPA / htmx)

*   **Pros**: One repository; no drift between variants.
*   **Cons**: Init grows a branch per variant, and each combination needs its own leg in the
    `template-smoke.yml` or it is untested.
*   **Reason for Rejection**: Rejected by author: `template-smoke.yml` has three legs for one
    variant; four variants would need twelve. Accepted consequence: a later second template
    drifts from this one and is aligned by hand.

### Alternative C: Shipping a React + rsbuild frontend skeleton

*   **Pros**: A full-stack starting point; browser e2e tests become possible.
*   **Cons**: Adds a node toolchain, a build stage and a second dependency stream to every
    derived project, including those with another frontend or none.
*   **Reason for Rejection**: Rejected by author: the frontend stack differs per project, so
    most derived projects would delete the skeleton. The template ships the API side a frontend
    needs: the CORS allow-list, the `app-web` client and the JSON POST form of every RPC.

### Alternative D: The template's own token verifier instead of a maintained OIDC library

*   **Pros**: No new dependency; matches the reference project and the convention "stdlib
    before a dependency".
*   **Cons**: Derived projects have no update channel for copied code, and a verifier fix
    already failed to reach the template from the reference project. Minimum repair list:
    negative cache and single-flight for an unknown `kid`, the issuer comparison, the
    outage-versus-bad-token split, more algorithms.
*   **Reason for Rejection**: Rejected by author: with `github.com/coreos/go-oidc/v3`, signature
    and claim checks arrive as dependency updates through Dependabot. The cost is not zero:
    go-oidc needs the wrapper described in section 7, about 180 lines that are template code
    and have no update channel either. The trade is 180 lines of glue with tests against 570
    lines of cryptographic verification.

### Alternative E: Authentication as a Connect interceptor instead of HTTP middleware

*   **Pros**: Matches the reference project; less code to port; the procedure name is known when
    auth runs.
*   **Cons**: Runs after the unary body is decoded; a plain route added later is
    unauthenticated by default.
*   **Reason for Rejection**: Rejected by author: as group middleware, authentication runs
    before a body is read and covers unary RPCs, streams and plain routes alike. The cost is
    that auth failures have no procedure name in logs and metrics. `connectrpc.com/authn` is not
    needed: `connecthttp.NewErrorWriter` writes the failure in the caller's protocol.

### Alternative F: Renaming the proto package at init

*   **Pros**: Each derived project gets its own wire names.
*   **Cons**: Init would have to rewrite proto packages, directories, generated package names
    and every import of them.
*   **Reason for Rejection**: Rejected by author: the rename multiplies what init rewrites and
    what `scripts/template-smoke.sh` must prove, for a name clients see only in URLs. The proto package
    is `api.v1` in every derived project; init rewrites the Go module path and regenerates.

### Alternative G: An empty skeleton after init, or a permanent RPC plus a baseline migration

*   **Pros**: The empty skeleton ships no code a project did not ask for. A baseline migration
    would give the permanent test a table to query in one request.
*   **Cons**: The empty skeleton leaves B2 and B6 open: buf fails with no proto file and nothing
    exercises the chain. A baseline migration puts a placeholder table into every derived
    database.
*   **Reason for Rejection**: Rejected by author: a permanent `WhoAmIService` costs one proto
    file, one handler and one test, and the database side tolerates an empty tree. Accepted
    cost: after the strip no request reaches a table and the failing path of validation is
    untested.

### Alternative H: Keep the demo by default and strip it later (`make rm-demo`)

*   **Pros**: A worked example stays in the tree for agents.
*   **Cons**: A notes table reaches databases, and a later strip has to deal with a migration
    that is already applied as version 1.
*   **Reason for Rejection**: Rejected by author: a late strip would make init database-aware.
    The default is stripped, `--keep-demo` is final as far as init is concerned, and the worked
    example agents use is the permanent `whoami` slice plus the snippets in `go-feature`.

### Alternative I: AST-based import rewriting, or not committing generated code

*   **Pros**: Avoids descriptor corruption without regenerating at init.
*   **Cons**: AST rewriting cannot fix the length-prefixed descriptor in `.pb.go`. Without
    committed generated code a fresh clone cannot build without the tools, and there is no drift
    check.
*   **Reason for Rejection**: Rejected by author: regeneration is the only method the spike
    showed to repair the rename, and it needs the tools at init either way. Init rewrites the
    module path, never edits generated files, and regenerates.

### Alternative J: Go `tool` directives (or a tools module) for every tool

*   **Pros**: One pin location; Dependabot sees the pins.
*   **Cons**: buf, sqlc and golangci-lint in the `tool` block add their dependencies to the
    module graph of every derived project unless a separate module is used.
*   **Reason for Rejection**: Rejected by author for buf, sqlc, golangci-lint and govulncheck:
    they are pinned as Makefile variables and installed by `make tools`. Accepted cost:
    Dependabot does not see these four pins. The `tool` directive is used for the two protoc
    plugins, where it adds no require.

### Alternative K: buf remote plugins instead of local ones

*   **Pros**: No plugin installs.
*   **Cons**: The plugin version is chosen in `buf.gen.yaml`, apart from the runtime library
    version in `go.mod`, so the two can drift.
*   **Reason for Rejection**: Rejected by author: local plugins run through `go tool` take their
    version from `go.mod`, so generated code and runtime library always match. Network access
    is not the reason: the protovalidate dependency needs buf.build whenever the buf module
    cache is cold, with either kind of plugin.

### Alternative L: Start from the reference project and delete, instead of refactoring the template

*   **Pros**: Faster to working Connect code.
*   **Cons**: Loses init, the demo slice and dev auth; tenancy is in the reference project's
    auth, database, principal, logging and fixtures and would have to be removed from each.
*   **Reason for Rejection**: Rejected by author: deleting tenancy from five layers is harder to
    verify than porting the generic parts onto a tree that never had it. Section 21 is the
    include/exclude list.

### Alternative M: Stack-neutral skills as a separately installed pack

*   **Pros**: Removes skill drift between this template and any later one.
*   **Cons**: Loses the self-contained mirror other harnesses read; adds an install step to
    every derived project.
*   **Reason for Rejection**: Rejected by author: there is one template, so there is no drift to
    remove yet, and the install step would be paid by every derived project now.

### Alternative N: Removing `AUTH_PROVIDER=none`, as the reference project did

*   **Pros**: One auth code path; B3 cannot occur.
*   **Cons**: The first boot then needs Keycloak running.
*   **Reason for Rejection**: Rejected by author: a first run that needs only Postgres is a
    requirement. `none` is set only in the dev env file, has no built-in default and is refused
    outside the dev allow-list; `make image-smoke` proves the refusal on the built image.

### Alternative O: A dev-mode in-process issuer in the server binary instead of `none`

*   **Pros**: One code path for dev and production.
*   **Cons**: Ships token minting in the binary.
*   **Reason for Rejection**: Rejected by author: a binary that can mint tokens is a larger risk
    than a mode that is refused outside the allow-list. The in-process provider is for Go
    integration tests only; nothing under `cmd/` or `internal/server` imports it.

### Alternative P: Built-in defaults for the connection values, equal to the compose file

*   **Pros**: `bin/app run` works by hand with nothing set; `db migrate` needs no variable in
    development.
*   **Cons**: A deployment that sets `AUTH_PROVIDER=keycloak` and forgets the issuer starts and
    answers `unavailable` to every request; the values live in three places.
*   **Reason for Rejection**: Rejected by author: every dev path already gets its values from
    `.env.local`, so the built-in defaults would be reached only by a misconfigured deployment.
    A middle option, requiring the Keycloak variables only outside the dev allow-list, needs the
    reader to track whether a value came from the environment or a default.

### Alternative Q: Read and write timeouts with a per-stream exemption

*   **Pros**: Response delivery of unary calls is bounded; streams still work.
*   **Cons**: An adapter around `connecthttp.Mount` that depends on Mount's route patterns, a
    middleware, a scan of the registered procedures and a test, for a template that ships no
    streaming RPC. An exempt stream then has no bound at all.
*   **Reason for Rejection**: Rejected by author: the mechanism has no user in the template and
    its unary bound does not stop handler work on HTTP/1.1. Both timeouts are 0 and
    `docs/production.md` carries the recipe.

### Alternative R: The duration histogram as a Connect interceptor

*   **Pros**: The procedure is always known; the duration covers the RPC only.
*   **Cons**: Requests stopped by `Auth` never reach an interceptor, so an identity-provider
    outage or a burst of bad tokens moves no series.
*   **Reason for Rejection**: Rejected by author: observing in `AccessLog` is the same size and
    counts those requests. Accepted cost: they share one empty `procedure` label, and the
    duration includes middleware time. A separate counter in `Auth` would add a second metric
    family and thread a registerer into the auth middleware.

### Alternative S: Readiness on the database only, or exit on any discovery failure at start

*   **Pros**: Database-only readiness needs no state. Exit-on-failure needs no lazy discovery.
*   **Cons**: With database-only readiness, a replica started during a provider outage passes
    `/readyz` and answers `unavailable` to everything, so a rolling restart replaces replicas
    that serve with replicas that cannot. With exit-on-failure, a cold start during an outage
    crash-loops under the orchestrator's restart backoff.
*   **Reason for Rejection**: Rejected by author: the latch keeps a keyless replica out of
    rotation without restarting it and recovers within 5 s of the provider's return. It costs
    one method on the verifier and one optional interface in `health.go`. An issuer mismatch
    still exits at start, because waiting cannot fix it.

### Alternative T: Rewriting init in Go

*   **Pros**: Structured rewriting without `sed`; no bash 3.2 and BSD userland constraints.
*   **Cons**: The file must not import the module and must exclude itself from its own rename;
    skills and README that name `scripts/init.sh` need a new path.
*   **Reason for Rejection**: Rejected by author: a bash init already exists, and the `macos`
    leg of `template-smoke.yml` covers BSD userland.

### Alternative U: A generated example-slice document that survives the strip

*   **Pros**: A stripped project keeps a copy of the persistence half of the demo for agents.
*   **Cons**: A generator script, a Make target, a drift-check entry and exemptions in every
    residue check; after the strip the document is never regenerated and describes paths that
    no longer exist.
*   **Reason for Rejection**: Rejected by author: static snippets in the `go-feature` reference
    files carry the same content with no generator and no exemption.

### Alternative V: A reference checker and a rule confining stack facts to four locations

*   **Pros**: Every path and Make target named in agent-facing prose is checked in CI.
*   **Cons**: The checker ships to derived projects in `make check`, where a new skill that
    names a package fails the gate; its rules need a definition of "path" and "stack word" that
    prose does not have; the confinement vocabulary forces about 30 skill edits that only move
    words.
*   **Reason for Rejection**: Rejected by author: the one check that pays for itself, banned
    old-stack terms, runs in `scripts/template-smoke.sh`, which derived projects do not inherit.

### Alternative W: An origin record written by init (`.template`)

*   **Pros**: A derived project could diff against the template commit it came from.
*   **Cons**: No step reads the record; it must be exempt from the rename and from every residue
    check.
*   **Reason for Rejection**: Rejected by author: there is no update mechanism for it to serve.
    The presence of `scripts/template-smoke.sh` is the "not yet initialised" signal instead.

--------------------------------------------------------------------------------

## Non-Functional Requirements

**Security and privacy**

*   Fail-closed auth configuration: `AUTH_PROVIDER` has no default and the binary refuses to
    start without it; `APP_ENVIRONMENT` defaults to `production`; `none` is refused outside the
    allow-list `development`, `dev`, `local`, `test`. `make image-smoke` proves it on the built
    image.
*   Fail-closed connection configuration: `DATABASE_URL`, `KEYCLOAK_ISSUER_URL` and
    `KEYCLOAK_AUDIENCE` have no built-in default, so a deployed binary cannot fall back to a
    localhost value.
*   Fail-closed request handling: any verifier error that is not `ErrUnavailable`, and a nil
    principal, are `unauthenticated`. An identity-provider outage is `unavailable`, never a
    pass. Auth runs before the body is read, on every route of the authed group, streams and
    plain routes included. Handlers that find no principal return `unauthenticated`.
*   Token handling: signature before claims; fixed algorithm list without `HS*` and `none`;
    issuer, audience, expiry, token type and non-empty `sub` checked; ID and refresh tokens
    rejected. Outbound load from unauthenticated callers is bounded: at most one request per URL
    per 5 s reaches Keycloak.
*   Owner scoping is in the SQL: every statement carries the owner predicate, and another
    owner's row is `not_found`.
*   What is logged: one access line per request with method, path without query string, status,
    duration, code, request id, procedure. Never logged: tokens, the `Authorization` header,
    claims, request or response bodies, the subject in the access line, the query string (a
    Connect GET carries the message there). A token rejection reason is logged at DEBUG only.
*   What a client can learn: the fixed auth messages, `apperr` messages, validation text, and
    `internal error`. Driver errors, wrapping text and panic values go to the log only. `/readyz`
    returns fixed bodies.
*   Secrets: none in the repository beyond the dev-only compose credentials and the dev realm
    (`admin/admin`, `password`, `dev-only-secret`), each labelled dev-only; `docs/auth.md` says
    not to import the dev realm. The old realm export with private keys and hashes is deleted.
    The image holds no `.env*` file and no token-minting code.
*   Listeners: the dev env file binds the API and the management listener to 127.0.0.1. The
    management listener has no auth, is off by default and is not exposed by the image;
    `docs/production.md` says not to publish it. Probe paths should not be routed from the
    internet.
*   Supply chain: actions pinned by SHA, workflows with `contents: read`, Dependabot for Go
    modules, actions, Docker and compose images, `govulncheck` in CI, a distroless non-root
    image.

**Performance and latency**

Performance drives three decisions and nothing else is tuned:

*   The replay transport bounds outbound verifier traffic to one request per URL per 5 s.
*   One Postgres container per test process with a template database cloned per test: about
    1.3 s per package for the container, then tests run in parallel on private databases.
*   The server sets no read or write timeout, so streams are not cut; the bound on a handler is
    the client's deadline. Response delivery of a unary call is not bounded by the server. The
    per-message limit is 4 MiB.

**Rollout and rollback**

*   This is a breaking rewrite of a template. It has no deployed consumers: projects created
    from the template earlier never receive updates, so nothing migrates and no compatibility
    layer or feature flag is needed.
*   The work lands in four stages on one branch (Detailed Implementation Plan) and is merged as
    one pull request. Between stages the tree is knowingly inconsistent in the ways each stage
    names; no intermediate state is released. Rollback is a revert of the merge.
*   The old stack is kept as the annotated tag `rest-chi-openapi` on commit `b33836c`, the
    branch head before stage 1. The tag points at the commit directly, so the merge strategy of
    the pull request (merge commit or squash) does not affect it. The name is not `v*`, so it
    never becomes what `@latest` resolves to. The tagged tree is the old stack as it is, with
    its known defects; it is not maintained.
*   After the merge: `git push origin rest-chi-openapi` and `gh repo edit --template`. Both
    commands are in the README section "Maintaining this template".
*   Derived projects have no update mechanism and no origin record.
*   Operations of a derived project: migrations run as a job before the rollout; a new binary
    on an old schema refuses to start, so old replicas keep serving until the job has run; a
    newer schema passes the check, so a binary rollback works.

**Risks and limits**

Each item gives the limit, its consequence and what the design does about it.

*   **No server-side time bound on a handler.** A client that sends no deadline and a query
    that hangs hold a pool connection until the client disconnects. Mitigation: documented in
    `docs/production.md` with the timeout recipe; clients are expected to send deadlines.
*   **The pool is small by default and readiness shares it.** With pgxpool defaults the maximum
    is the greater of 4 and the CPU count. A few slow queries exhaust it, the readiness ping
    times out, and every replica leaves rotation together. Mitigation: the sample production
    `DATABASE_URL` in `docs/production.md` sets `pool_max_conns`.
*   **Streams.** The template ships none. A stream is authenticated when it opens and not
    re-checked: one opened with a short-lived token survives logout, user disablement and key
    removal. A stream has no server-side bound and is cut when the shutdown grace ends.
    Mitigation: stated in the `go-feature` reference for RPCs and in `docs/production.md`.
*   **Shutdown with a stream open.** Every rolling restart then takes the full 25 s and exits 1,
    which a supervisor that restarts on failure reads as a crash. Mitigation: documented;
    streams should end on a client deadline.
*   **No drain window in code.** Under Kubernetes, requests routed between SIGTERM and endpoint
    removal are refused. Mitigation: the `preStop` `sleep` in `docs/production.md`.
*   **Identity provider.** The three limits of section 7: a removed key stays trusted until
    restart or the next refetch; requests that need a fetch stall up to 5 s per window during a
    hang, and ERROR volume is caller-controlled during an outage; the wrapper has no update
    channel. Verification is local, so a logout or a disabled user takes effect when the token
    expires. There is no expiry leeway, so API hosts need synchronized clocks. Mitigation: the
    `docs/auth.md` checklist (restart after deleting a key; short access-token lifespan).
*   **A slow readiness probe during a provider hang.** A replica without keys runs discovery
    from `/readyz`; while the provider black-holes requests, that probe can take up to 10 s
    (two fetches of 5 s). The replica is not ready in that state, so a probe timeout gives the
    same result.
*   **Metrics.** Requests stopped by `Auth` share one empty `procedure` label, and
    unauthenticated noise moves the `unauthenticated` series. Plain routes and unknown methods
    are not measured.
*   **Migrations.** The session locker does not hold through a connection pooler in transaction
    mode. Mitigation: `docs/production.md` says to run the migrate job against Postgres
    directly, as one job before the rollout.
*   **After the strip.** No request reaches a table and the failing path of validation has no
    test until the project adds its first rule and query.
*   **`--keep-demo` is final for init.** Removing the demo later is manual and leaves a version
    1 migration behind in existing databases. Mitigation: init prints this; the steps are in
    `internal/domain/notes/doc.go`.
*   **Host ports are literals.** Two derived projects cannot run their dev stacks at the same
    time without editing ports.
*   **Network at generate time.** `buf generate` needs buf.build whenever the buf module cache
    is cold: every CI run and the first init on a machine.
*   **No rate limiting.** The only built-in bound on abuse is the outbound cap of the replay
    transport.
*   **Hand-bumped pins.** The four Makefile pins, the `toolchain` line and the harness Postgres
    tag are invisible to Dependabot. Mitigation: the weekly scheduled runs show when one stops
    building. GitHub disables scheduled workflows after 60 days without repository activity;
    the README says to re-enable them.
*   **Parts not run.** The Evidence paragraph of the Technical Plan lists what was designed
    here without a run. The stage gates are where each first runs.

--------------------------------------------------------------------------------

### Additional Resources

*   `AGENTS.md` and `docs/agents/go-style.md`: the project rules the design follows.
*   The tag `rest-chi-openapi`: the tree this refactor starts from.
*   Package documentation at the versions of section 22: `connectrpc.com/connect/v2` and its
    `connecthttp` package, `connectrpc.com/validate`, `github.com/go-pkgz/routegroup`,
    `github.com/coreos/go-oidc/v3`, `github.com/pressly/goose/v3` and its `lock` package,
    `github.com/jackc/pgx/v5`.
*   RFC 6750 section 3 (`WWW-Authenticate` for bearer tokens) and RFC 9068 (the `at+jwt` header
    type).
*   The spike module, the side probe, the Keycloak runs and the review reports of this design
    are not committed. The Evidence table in the Technical Plan records what each established.

### Future Improvements

*   **htmx template**: a separate repository, started after this template is merged and one
    real project has been created from it. `internal/identity`, the config reader, `internal/db`
    and the Postgres harness are the parts it can reuse; the API-only CSP and CORS are not.
*   **`buf breaking` in CI**: `buf.yaml` configures the `FILE` rule set; a derived project turns
    the check on once its API has consumers.
*   **A unary handler deadline**: one constant applied through `context.WithTimeout` for unary
    procedures; the `Errors` interceptor already maps `context.DeadlineExceeded`.
*   **A maximum key age** in the key-set wrapper, falling back to the cached keys when a
    refresh fails, to bound how long a removed key stays trusted.
*   **Expiry leeway**: add through `oidc.Config.Now` if clock skew shows up in practice.
*   **Validation test after the strip**: a permanent message with a rule would give the
    validation interceptor a failing-path test without the demo.

--------------------------------------------------------------------------------

## Detailed Implementation Plan

The work lands in four stages on one branch. Each file has one numbered entry, under the stage
that changes it, and the entry states the file's end state. No entry describes an intermediate
edit. "Section N" refers to the Technical Plan. `<module>` stands for the Go module path, which
is `github.com/driif/go-vibe-starter` in the template tree.

| Stage | Content | Gate |
|---|---|---|
| 0 | The tag `rest-chi-openapi`. No file changes. | the tag exists |
| 1 — stack swap | All Go code, proto, SQL, config, compose, env files, `Makefile`, `Dockerfile`, `.golangci.yml`, and the two scripts the Makefile calls. The old stack is deleted in the same stage. | `make check` passes in the template tree (demo present) |
| 2 — docs and skills | `README.md`, `AGENTS.md`, `CLAUDE.md`, `CONTEXT.md`, `docs/`, `.claude/skills/`, the regenerated mirror. | `scripts/sync-skills.sh --check` passes and the acceptance greps of entry 173 print nothing |
| 3 — template mechanics | `scripts/init.sh`, `scripts/template-smoke.sh`, `.github/workflows/`, repository hygiene files. | the `stripped` and `keep-demo` commands of section 18 pass locally, residue check included |

Knowingly inconsistent between stages, and not released in that state: after stage 1 the docs
and skills still describe the old stack and `make init` still runs the old script; after stage 2
`make init` still runs the old script and CI still runs the old workflow. Stage 3 ends with the
definition of done (Goals).

This document is committed as `docs/designs/connect-rpc-template.md`; init deletes
`docs/designs/` in derived projects. The draft and review files that sit untracked in the
repository root are not committed.

### Target Files to Create/Modify

**Stage 0 — tag.** `git tag -a rest-chi-openapi b33836c -m "last commit on chi + OpenAPI"`. The
tag marks the old stack as it is; nothing in the old stack is fixed first. As of 2026-11-21 the
annotated tag exists in the author's clone and points at `b33836c` (checked with `git
rev-parse`). It is pushed after the merge (Rollout). Stage 0 has no file entry.

**Stage 1 — stack swap.** Gate: `make tools && make check` in the template tree. By hand before
moving on: the quickstart of CUJ 2, `make smoke`, and `make image-smoke`.

#### 1. `go.mod` (Modify)

*   **Type / Scope**: Module file.
*   **Proposed Changes**: `go 1.26.0`, `toolchain go1.27.4`. Direct requires exactly as in the
    table of section 22; `github.com/go-jose/go-jose/v4` raised to v4.1.9 as an indirect
    require. A `tool (...)` block with `connectrpc.com/connect/v2/cmd/protoc-gen-connect-go` and
    `google.golang.org/protobuf/cmd/protoc-gen-go`, written by `go get -tool`. The six removed
    requires of section 22 are gone. `go mod tidy` leaves the file unchanged.

#### 2. `go.sum` (Modify)

*   **Type / Scope**: Module checksums.
*   **Proposed Changes**: Written by `go mod tidy` for entry 1.

#### 3. `Makefile` (Modify)

*   **Type / Scope**: Build and task runner.
*   **Proposed Changes**: Rewritten. Variables: `APP_NAME := app` (the line init rewrites),
    `BIN := bin/$(APP_NAME)`, `TOOLS := $(CURDIR)/bin/tools`, `BUF_VERSION := v1.75.0`,
    `SQLC_VERSION := v1.31.1`, `GOLANGCI_LINT_VERSION := v2.15.0`,
    `GOVULNCHECK_VERSION := v1.10.0`,
    `VERSION ?= $(shell git describe --tags --match 'v*' --always --dirty 2>/dev/null || echo dev)`,
    `IMAGE ?= $(APP_NAME):dev`, `LDFLAGS = -X $(shell go list -m)/cmd.version=$(VERSION)`
    (recursive, so `go` runs only when used), `GEN_PATHS := go.mod go.sum internal/gen internal/db/gen`,
    `KC_USER ?= alice`, `KEYCLOAK_ISSUER_URL ?= http://localhost:8181/realms/app`. `help` is the
    default goal and prints every `target: ## text` line with `awk`. Targets:

    | Target | Runs |
    |---|---|
    | `help` | the `awk` listing |
    | `init` | `bash scripts/init.sh` |
    | `tools` | for each of the four tools: `GOBIN=$(TOOLS) go install <package>@<version>`, then a stamp file `bin/tools/.<tool>-<version>` (for example `bin/tools/.buf-v1.75.0`); older stamps of that tool are removed. Packages: `github.com/bufbuild/buf/cmd/buf`, `github.com/sqlc-dev/sqlc/cmd/sqlc`, `github.com/golangci/golangci-lint/v2/cmd/golangci-lint`, `golang.org/x/vuln/cmd/govulncheck` |
    | `tools-check` | tests the four stamps; on failure prints `tools missing or outdated: run 'make tools'` and exits 1. `gen-proto`, `sqlc`, `lint`, `fmt`, `check`, `vuln` depend on it |
    | `build` | `go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) .` |
    | `.env.local` (file target) | `cp .env.example .env.local`; never overwrites |
    | `run` | deps `build .env.local`; `./$(BIN) run` |
    | `test` | `go test -race ./...` |
    | `cover` | `go test -race -coverprofile=profile.cov ./...`; prints the last line of `go tool cover -func` |
    | `lint` | `$(TOOLS)/golangci-lint run`; `$(TOOLS)/buf lint`; `$(TOOLS)/buf format -d --exit-code` |
    | `fmt` | `$(TOOLS)/golangci-lint fmt`; `$(TOOLS)/buf format -w` |
    | `gen` | `gen-proto sqlc` |
    | `gen-proto` | `$(TOOLS)/buf generate` |
    | `sqlc` | `rm -rf internal/db/gen`; `$(TOOLS)/sqlc generate` when `sql/queries/*.sql` exists, else prints `sqlc: no query files, skipped` |
    | `drift` | copies each path of `GEN_PATHS` that exists into `$(mktemp -d)`; runs `$(MAKE) gen` and `go mod tidy`; compares each path with its copy using `diff -r` (a path absent on both sides is equal); on a difference prints the paths and `generated files or go.mod are stale: commit the output of 'make gen' and 'go mod tidy'` and exits 1; removes the temporary directory. The regenerated output stays in the tree |
    | `check` | `drift`, `lint`, `test`, then `scripts/sync-skills.sh --check`, in this order, stopping at the first failure |
    | `vuln` | `$(TOOLS)/govulncheck ./...` |
    | `image` | `docker build --build-arg VERSION=$(VERSION) -t $(IMAGE) .` |
    | `image-smoke` | dep `image`; `scripts/image-smoke.sh $(IMAGE) $(APP_NAME) $(VERSION)` |
    | `migrate-up`, `migrate-down`, `migrate-status` | deps `build .env.local`; `./$(BIN) db migrate up`, `down`, `status` |
    | `dev-up` | `docker compose up -d --wait postgres` |
    | `dev-down` | `docker compose down` (volumes are kept) |
    | `keycloak-up` | `docker compose up -d --wait --force-recreate keycloak` |
    | `smoke` | dep `keycloak-up`; `go test -race -count=1 -tags smoke -run '^TestSmoke' ./internal/test/` |
    | `token` | `curl -fsS -d grant_type=password -d client_id=app-test -d username=$(KC_USER) -d password=password $(KEYCLOAK_ISSUER_URL)/protocol/openid-connect/token`, piped through `sed -E 's/.*"access_token":"([^"]+)".*/\1/'` |
    | `labels` | `scripts/setup-labels.sh` |
    | `sync-skills` | `scripts/sync-skills.sh` |
    | `clean` | `rm -rf bin profile.cov` |

    Removed: `gen-oapi`, `db-up`, `db-down`, `BIN_DIR`, `COVER_PROFILE`, the
    `OAPI_CODEGEN_VERSION` and `GOOSE_VERSION` pins, `goimports@latest`, the hand-kept help text
    with the repository name.

#### 4. `.golangci.yml` (Modify)

*   **Type / Scope**: Lint config.
*   **Proposed Changes**: `run.build-tags: [smoke]`. Exclusion and formatter paths:
    `internal/db/gen` (already present, stays) and `internal/gen`; `'.*\.gen\.go$'` is removed.
    `errcheck.exclude-functions`: `(*database/sql.DB).Close` stays (the goose path opens a
    `*sql.DB`); `(*github.com/jackc/pgx/v5.Conn).Close` is added. Rules removed: the
    `cmd/db_migrate\.go` SA1019 rule, the `pkg/dotenv/` rule, the
    `pkg/tests/|internal/server/test/` rule. Rules added: gosec off for
    `internal/server/config/dotenv\.go`; gosec `G108` off for `internal/server/management\.go`
    (pprof import); gocritic, gosec, noctx off for `internal/test/|internal/identity/testprovider/`;
    revive `var-naming` with text `method \w+ should be` off for `internal/rpc/` (a handler
    method has the name of its RPC). Linters added: `sloglint` (`key-naming-case: snake`,
    `no-mixed-args: true`), `forbidigo` (pattern `^(fmt\.Print(|f|ln)|print|println)$`),
    `errorlint`. `unparam` stays. `version: "2"` stays.

#### 5. `buf.yaml` (Create)

*   **Type / Scope**: buf module config.
*   **Proposed Changes**: `version: v2`; one module with `path: proto`; dep
    `buf.build/bufbuild/protovalidate`; lint `STANDARD`; breaking `FILE`. The dep stays after
    the strip; buf lints and generates with it unused.

#### 6. `buf.gen.yaml` (Create)

*   **Type / Scope**: buf codegen config.
*   **Proposed Changes**: `version: v2`; `clean: true`; managed mode enabled with one override,
    `go_package_prefix` = `<module>/internal/gen` (the one line init rewrites), and
    `go_package` management disabled for module `buf.build/bufbuild/protovalidate`; two plugins,
    `local: ["go", "tool", "protoc-gen-go"]` and `local: ["go", "tool", "protoc-gen-connect-go"]`,
    each with `out: internal/gen` and `opt: paths=source_relative`.

#### 7. `buf.lock` (Create)

*   **Type / Scope**: buf lock file.
*   **Proposed Changes**: Written by `buf dep update`; committed so generation is reproducible.

#### 8. `sqlc.yaml` (Modify)

*   **Type / Scope**: sqlc config.
*   **Proposed Changes**: `version: "2"`; engine `postgresql`; `schema: sql/migrations`;
    `queries: sql/queries`; Go package `gen`, `out: internal/db/gen`, `sql_package: pgx/v5`,
    `emit_empty_slices: true`; overrides `uuid` → `github.com/google/uuid.UUID` and
    `timestamptz` → `time.Time`. `emit_interface`, `emit_json_tags` and `emit_db_tags` are
    removed.

#### 9. `proto/api/v1/whoami.proto` (Create)

*   **Type / Scope**: Proto, permanent.
*   **Proposed Changes**: `syntax = "proto3"`; `package api.v1`; no imports. Service
    `WhoAmIService` with `rpc WhoAmI(WhoAmIRequest) returns (WhoAmIResponse)` and
    `option idempotency_level = NO_SIDE_EFFECTS`. `WhoAmIRequest` is empty. `WhoAmIResponse`:
    `string subject = 1`, `string username = 2`, `string email = 3`, `string name = 4`,
    `repeated string roles = 5`.

#### 10. `proto/api/v1/notes.proto` (Create)

*   **Type / Scope**: Proto, demo.
*   **Proposed Changes**: Line 1 `// DEMO: notes`; `package api.v1`; imports
    `buf/validate/validate.proto` and `google/protobuf/timestamp.proto`. A superset of this file
    (with an update RPC and paging fields) passed `buf format`, `buf lint` and `buf generate`
    on 2026-11-20. Contract:

    ```proto
    service NotesService {
      rpc CreateNote(CreateNoteRequest) returns (CreateNoteResponse);
      rpc GetNote(GetNoteRequest) returns (GetNoteResponse) {
        option idempotency_level = NO_SIDE_EFFECTS;
      }
      rpc ListNotes(ListNotesRequest) returns (ListNotesResponse) {
        option idempotency_level = NO_SIDE_EFFECTS;
      }
      rpc DeleteNote(DeleteNoteRequest) returns (DeleteNoteResponse);
    }

    message Note {
      string id = 1;
      string title = 2;
      string body = 3;
      google.protobuf.Timestamp create_time = 4;
    }

    message CreateNoteRequest {
      string title = 1 [(buf.validate.field).string = {min_len: 1, max_len: 200}];
      string body = 2;
    }
    message CreateNoteResponse { Note note = 1; }

    message GetNoteRequest { string id = 1 [(buf.validate.field).string.uuid = true]; }
    message GetNoteResponse { Note note = 1; }

    message ListNotesRequest {}
    message ListNotesResponse { repeated Note notes = 1; }  // the caller's newest 100

    message DeleteNoteRequest { string id = 1 [(buf.validate.field).string.uuid = true]; }
    message DeleteNoteResponse {}
    ```

#### 11. `internal/gen/api/v1/whoami.pb.go` (Create)

*   **Type / Scope**: Generated by `buf generate`, committed.
*   **Proposed Changes**: Output of `protoc-gen-go`, package `apiv1`; never edited.

#### 12. `internal/gen/api/v1/apiv1connect/whoami.connect.go` (Create)

*   **Type / Scope**: Generated by `buf generate`, committed.
*   **Proposed Changes**: Output of `protoc-gen-connect-go`, package `apiv1connect`; never
    edited.

#### 13. `internal/gen/api/v1/notes.pb.go` (Create)

*   **Type / Scope**: Generated, demo.
*   **Proposed Changes**: Output of `protoc-gen-go`; removed by `clean: true` on the first
    `make gen` after the strip.

#### 14. `internal/gen/api/v1/apiv1connect/notes.connect.go` (Create)

*   **Type / Scope**: Generated, demo.
*   **Proposed Changes**: Output of `protoc-gen-connect-go`; removed by `clean: true` on the
    first `make gen` after the strip.

#### 15. `sql/embed.go` (Create)

*   **Type / Scope**: Go, package `sql`.
*   **Proposed Changes**: `//go:embed all:migrations` on `var Migrations embed.FS`. Imported
    elsewhere as `migrations "<module>/sql"`.

#### 16. `sql/migrations/.gitkeep` (Create)

*   **Type / Scope**: Placeholder.
*   **Proposed Changes**: Empty file. Keeps the embedded directory present after the strip.

#### 17. `sql/migrations/00001_notes.sql` (Create)

*   **Type / Scope**: goose migration, demo. Replaces `migrations/0001_notes.sql`.
*   **Proposed Changes**:

    ```sql
    -- DEMO: notes
    -- +goose Up
    CREATE TABLE notes (
        id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
        owner      text NOT NULL,
        title      text NOT NULL,
        body       text NOT NULL,
        created_at timestamptz NOT NULL DEFAULT now()
    );

    CREATE INDEX notes_owner_created_at_idx ON notes (owner, created_at DESC);

    -- +goose Down
    DROP TABLE notes;
    ```

    `owner` holds `Principal.Subject`. Against today's table: `owner_id` becomes `owner`,
    `updated_at` and the default of `body` are dropped, the index gains `created_at DESC`.

#### 18. `sql/queries/notes.sql` (Modify)

*   **Type / Scope**: sqlc queries, demo.
*   **Proposed Changes**: Named parameters; every statement carries the owner.

    ```sql
    -- DEMO: notes

    -- name: CreateNote :one
    INSERT INTO notes (owner, title, body)
    VALUES (@owner, @title, @body)
    RETURNING *;

    -- name: GetNote :one
    SELECT * FROM notes
    WHERE id = @id AND owner = @owner;

    -- name: ListNotes :many
    SELECT * FROM notes
    WHERE owner = @owner
    ORDER BY created_at DESC, id DESC
    LIMIT 100;

    -- name: DeleteNote :execrows
    DELETE FROM notes
    WHERE id = @id AND owner = @owner;
    ```

#### 19. `internal/db/db.go` (Create)

*   **Type / Scope**: Go, package `db`.
*   **Proposed Changes**:

    ```go
    func Open(ctx context.Context, url string) (*pgxpool.Pool, error)
    func Migrate(ctx context.Context, url, command string, out io.Writer) error // "up" | "down" | "status"
    func CheckSchema(ctx context.Context, pool *pgxpool.Pool) error
    ```

    `Open`: `pgxpool.New`, then `Ping`; closes the pool when the ping fails. `Migrate` calls
    the unexported `migrate(ctx, url, command, fsys, out)` with the embedded `migrations`
    subdirectory: `sql.Open("pgx", url)` (blank import of `github.com/jackc/pgx/v5/stdlib`),
    `goose.NewProvider(goose.DialectPostgres, conn, fsys, goose.WithSessionLocker(locker))`
    with `locker` from `lock.NewPostgresSessionLocker()`. `up` applies all pending and prints
    one line per migration; `down` rolls back one (nothing applied is not an error); `status`
    prints `applied  <file>` or `pending  <file>` per file; any other command is an error. On
    `goose.ErrNoMigrations` it prints `no migrations`, pings and returns the ping result.
    `CheckSchema` calls the unexported `checkSchema(ctx, pool, fsys)`: highest embedded version
    0 → nil without a query; otherwise `SELECT coalesce(max(version_id), 0) FROM
    goose_db_version` (SQLSTATE 42P01 counts as 0); lower than the embedded version → error
    `database schema is at version <have>, this binary needs <need>: run "db migrate up"`. No
    import of `internal/db/gen`. The package comment states that a transaction is
    `pgx.BeginFunc(ctx, pool, fn)` in the domain package.

#### 20. `internal/db/db_test.go` (Create)

*   **Type / Scope**: Test, `package db_test`.
*   **Proposed Changes**: `TestOpen` (a query on `test.Database`; an unreachable URL errors).
    `TestMigrate` (fixtures on `test.EmptyDatabase`: status pending, up, status applied, up
    again prints nothing, down, down, down with nothing applied succeeds; an unknown command
    errors). `TestMigrate_NoMigrations` (the `.gitkeep`-only fixture and an empty `fstest.MapFS`:
    all three commands print `no migrations`; an unreachable database errors).
    `TestMigrate_Embedded` (the real embedded tree: up, one `down` per file, up; passes with
    zero files). `TestCheckSchema` (zero migrations, never migrated, behind, level, ahead).

#### 21. `internal/db/export_test.go` (Create)

*   **Type / Scope**: Test seam.
*   **Proposed Changes**: Exports `migrate` and `checkSchema` as `MigrateFS` and `CheckSchemaFS`
    to `package db_test`, which avoids the import cycle with `internal/test`.

#### 22. `internal/db/testdata/migrations/00001_first.sql` (Create)

*   **Type / Scope**: Test fixture.
*   **Proposed Changes**: Up creates table `first`; Down drops it.

#### 23. `internal/db/testdata/migrations/00002_second.sql` (Create)

*   **Type / Scope**: Test fixture.
*   **Proposed Changes**: Up creates table `second`; Down drops it. Makes order and single-step
    `down` testable without the demo.

#### 24. `internal/db/testdata/empty/.gitkeep` (Create)

*   **Type / Scope**: Test fixture.
*   **Proposed Changes**: Empty file; the shape of `sql/migrations` in a stripped tree, for
    `TestMigrate_NoMigrations`.

#### 25. `internal/db/gen/db.go` (Modify)

*   **Type / Scope**: Generated by sqlc, demo.
*   **Proposed Changes**: Regenerated for `pgx/v5` with sqlc v1.31.1; never edited.

#### 26. `internal/db/gen/models.go` (Modify)

*   **Type / Scope**: Generated by sqlc, demo.
*   **Proposed Changes**: Regenerated: `Note` with `ID uuid.UUID`, `Owner`, `Title`, `Body`,
    `CreatedAt time.Time`; no struct tags.

#### 27. `internal/db/gen/notes.sql.go` (Modify)

*   **Type / Scope**: Generated by sqlc, demo.
*   **Proposed Changes**: Regenerated from entry 18: `CreateNote`, `GetNote`, `ListNotes`,
    `DeleteNote` (returns the affected row count).

#### 28. `internal/db/gen/querier.go` (Delete)

*   **Type / Scope**: Generated by sqlc.
*   **Proposed Changes**: Deleted; `emit_interface` is off and nothing consumes a `Querier`.

#### 29. `internal/apperr/apperr.go` (Create)

*   **Type / Scope**: Go, error type. No Connect import.
*   **Proposed Changes**:

    ```go
    type Kind int
    const ( Internal Kind = iota; InvalidArgument; NotFound; AlreadyExists; PermissionDenied; FailedPrecondition )
    type Error struct { Kind Kind; Reason string; Message string }
    func (e *Error) Error() string // Reason + ": " + Message
    func New(kind Kind, reason, message string) *Error
    ```

#### 30. `internal/reqlog/reqlog.go` (Create)

*   **Type / Scope**: Go, request-log carrier and slog handler.
*   **Proposed Changes**:

    ```go
    func With(ctx context.Context, id string) context.Context
    func SetProcedure(ctx context.Context, procedure string)
    func SetCode(ctx context.Context, code string)
    func Procedure(ctx context.Context) string // "" when unset
    func Code(ctx context.Context) string      // "" when unset
    func NewHandler(h slog.Handler) slog.Handler
    ```

    `With` stores a pointer to a mutex-guarded `{id, procedure, code}`. The setters do nothing
    when the context has no carrier. `NewHandler` appends `request_id` and, when set,
    `procedure` to each record; `WithAttrs` and `WithGroup` re-wrap.

#### 31. `internal/reqlog/reqlog_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestHandler_AddsFields`, `TestSetters_WithoutFields`,
    `TestHandler_WithAttrsKeepsWrapper`.

#### 32. `internal/identity/identity.go` (Create)

*   **Type / Scope**: Go, package `identity`. Imports `context`, `errors`, `slices` only.
*   **Proposed Changes**:

    ```go
    var ErrUnauthenticated = errors.New("identity: unauthenticated")
    var ErrUnavailable     = errors.New("identity: provider unavailable")

    type Principal struct {
        Subject  string   // "sub"; the owner id rows are scoped by
        Username string   // "preferred_username"
        Email    string   // "email"
        Name     string   // "name"
        Roles    []string // "realm_access.roles", unfiltered
    }
    func (p *Principal) HasRole(role string) bool // false for a nil receiver
    func WithPrincipal(ctx context.Context, p *Principal) context.Context
    func FromContext(ctx context.Context) *Principal // nil when absent

    type DevVerifier struct{ Roles []string }
    func (d DevVerifier) Verify(context.Context, string) (*Principal, error)
    ```

    `DevVerifier.Verify` returns a new principal with `Subject` and `Username` `local-dev` and a
    copy of `Roles`; it never returns an error.

#### 33. `internal/identity/identity_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestHasRole` (nil-safe), `TestFromContext`, `TestDevVerifier` (fixed
    principal, empty token accepted, roles passed through).

#### 34. `internal/identity/keycloak/keycloak.go` (Create)

*   **Type / Scope**: Go, package `keycloak`. The only importer of go-oidc.
*   **Proposed Changes**:

    ```go
    var ErrIssuerMismatch = errors.New("keycloak: provider reports another issuer")

    func New(issuer, audience string) *Verifier
    func (v *Verifier) Verify(ctx context.Context, rawToken string) (*identity.Principal, error)
    func (v *Verifier) Ready(context.Context) error
    ```

    `New` builds `oidc.NewVerifier(issuer, keys, &oidc.Config{ClientID: audience,
    SupportedSigningAlgs: <the fixed list of section 7>})`, where `keys` is the package's own
    `oidc.KeySet` and its `http.Client` has `Timeout: 5 * time.Second` and the `replay`
    transport (5 s) over `http.DefaultTransport`. `Verify`: empty token → `ErrUnauthenticated`
    before go-oidc; then go-oidc; then the token-type and `sub` checks; errors wrap
    `identity.ErrUnavailable` when the key set reported a fetch failure through the context
    out-parameter, `identity.ErrUnauthenticated` otherwise. The key set treats an error from
    `RemoteKeySet.VerifySignature` as a fetch failure exactly when `errors.Unwrap(err) != nil`.
    `Ready`: nil at once when latched; otherwise run discovery, then call the remote key set's
    `VerifySignature` with the constant primer `eyJhbGciOiJSUzI1NiIsImtpZCI6Ii0ifQ.e30.AA` (header
    `{"alg":"RS256","kid":"-"}`, payload `{}`); when discovery succeeded and the primer's error
    is not a fetch failure, set the latch and return nil; otherwise return the error. A go-oidc
    `*oidc.IssuerMismatchError` is wrapped with `ErrIssuerMismatch`. The outbound requests use
    `context.Background()` with the client; the parameter exists for the consumer interface.

#### 35. `internal/identity/keycloak/keycloak_test.go` (Create)

*   **Type / Scope**: Test against `testprovider`.
*   **Proposed Changes**: `TestVerify`, a table. Accepted (5): a Keycloak-shaped access token;
    `aud` as a list that contains `api`; header `typ: at+jwt`; header `typ: application/at+jwt`
    with no `typ` claim; no `realm_access`. Rejected (18), each `ErrUnauthenticated` and not
    `ErrUnavailable`: expired; no `exp`; `nbf` 10 minutes ahead; wrong audience; no audience;
    wrong issuer; no `sub`; empty `sub`; claim `typ: ID`; claim `typ: ID` with an `at+jwt`
    header; claim `typ: Refresh`; no type at all; unknown `kid`; `alg: HS256`; `alg: none`; a
    tampered signature; a string that is not a JWT; the empty string. `TestVerify_Principal`,
    `TestVerify_ProviderDownAtStart`, `TestVerify_ProviderUnreachable`,
    `TestVerify_ProviderDownLater`, `TestVerify_WrongIssuer` (also asserts `ErrIssuerMismatch`
    from `Ready`), `TestVerify_ForgedTokenCost`. `TestReady`: an error while the provider is
    down; nil after it returns; then, with the provider down again and no token verified
    before, a valid token verifies and `Ready` stays nil without a request.

#### 36. `internal/identity/keycloak/replay.go` (Create)

*   **Type / Scope**: Go, unexported `http.RoundTripper`.
*   **Proposed Changes**: Answers a GET for a URL it fetched under its TTL ago from memory,
    failures included; reads at most 1 MiB of a body; one mutex around lookup and fetch.

#### 37. `internal/identity/keycloak/replay_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestReplay_RepeatsFailureThenRecovers`: one discovery attempt per
    window during an outage, then recovery.

#### 38. `internal/identity/testprovider/testprovider.go` (Create)

*   **Type / Scope**: Go, non-test package imported by tests only.
*   **Proposed Changes**:

    ```go
    const Audience = "api"
    const KeyID    = "test"

    func New() *Provider                                          // one RSA-2048 key per test process
    func (p *Provider) URL() string                               // the issuer
    func (p *Provider) Close()
    func (p *Provider) Claims(subject string, roles ...string) map[string]any
    func (p *Provider) Token(claims map[string]any) string        // header {"alg":"RS256","typ":"JWT","kid":"test"}
    func (p *Provider) Sign(header, claims map[string]any) string // any header
    func (p *Provider) SetDown(down bool)                         // every endpoint answers 503
    func (p *Provider) Requests() int64
    ```

    Serves `GET /.well-known/openid-configuration` and `GET /protocol/openid-connect/certs`.
    `Claims` returns a valid Keycloak-shaped payload: `iss`, `sub`, `aud "api"`, `azp
    "app-test"`, `typ "Bearer"`, `iat`, `exp` (+5 min), `preferred_username` (= subject),
    `email` (`<subject>@example.test`), `name` (`Test <subject>`), `realm_access.roles`. No
    test file of its own; the verifier tests exercise it.

#### 39. `internal/rpc/interceptor/errors.go` (Create)

*   **Type / Scope**: Connect interceptor.
*   **Proposed Changes**: `func Errors() connect.ServerInterceptor`: the mapping table and the
    kind table of section 6; the kind lookup uses the two-value map form so an unknown kind
    maps to `internal`; the `ErrorInfo` detail with `Domain: "api"`; panic recovery.

#### 40. `internal/rpc/interceptor/requestlog.go` (Create)

*   **Type / Scope**: Connect interceptor.
*   **Proposed Changes**: `func RequestLog() connect.ServerInterceptor`: `reqlog.SetProcedure`
    with `spec.Procedure` before the call, `reqlog.SetCode` after it; a nil error is `ok`.

#### 41. `internal/rpc/interceptor/interceptor_test.go` (Create)

*   **Type / Scope**: Test; calls a `connect.ServerFunc` directly, no server.
*   **Proposed Changes**: `TestErrors_Mapping` (every row of the mapping table and of the kind
    table, for unary, server-stream, client-stream and bidi stream types), `TestRequestLog`.

#### 42. `internal/rpc/whoami/whoami.go` (Create)

*   **Type / Scope**: Connect handler, permanent.
*   **Proposed Changes**: `type Handler struct{}` with `WhoAmI(ctx, *apiv1.WhoAmIRequest)
    (*apiv1.WhoAmIResponse, error)`: copies the five principal fields from
    `identity.FromContext`; a nil principal returns
    `connect.NewError(connect.CodeUnauthenticated, "no principal")`. No database access.

#### 43. `internal/rpc/whoami/whoami_test.go` (Create)

*   **Type / Scope**: Integration test, permanent, `package whoami_test`.
*   **Proposed Changes**: `TestWhoAmI`: one `test.NewServer`; `GET /readyz` answers 200; a
    generated client with a minted token gets the claims back (with roles, without roles); no
    token is `unauthenticated`.

#### 44. `internal/domain/notes/doc.go` (Create)

*   **Type / Scope**: Go, demo documentation.
*   **Proposed Changes**: Line 1 `// DEMO: notes`, line 2 blank, then the package comment of
    package `notes`: what the slice shows; the list of its files (entries 10, 17, 18, 45 to 48,
    52 and the hub line); the marker forms of section 12; the manual removal steps (delete the
    marked files and the hub line, `make gen`, `go mod tidy`, then recreate the database or add
    a migration that drops `notes`).

#### 45. `internal/domain/notes/notes.go` (Create)

*   **Type / Scope**: Go, demo. Line 1 `// DEMO: notes`.
*   **Proposed Changes**:

    ```go
    type Note = gen.Note

    var ErrNotFound = apperr.New(apperr.NotFound, "NOTE_NOT_FOUND", "note not found")

    func NewStore(pool *pgxpool.Pool) *Store
    func (s *Store) Create(ctx context.Context, owner, title, body string) (Note, error)
    func (s *Store) Get(ctx context.Context, owner string, id uuid.UUID) (Note, error)
    func (s *Store) List(ctx context.Context, owner string) ([]Note, error)
    func (s *Store) Delete(ctx context.Context, owner string, id uuid.UUID) error
    ```

    Each method is one statement through `gen.New(pool)`. `Get`: `pgx.ErrNoRows` →
    `ErrNotFound`. `Delete`: zero rows affected → `ErrNotFound`. Any other error is returned
    unchanged.

#### 46. `internal/domain/notes/notes_test.go` (Create)

*   **Type / Scope**: Test, demo, `package notes_test`. Line 1 `// DEMO: notes`.
*   **Proposed Changes**: `TestStore_List` on `test.Database`: newest first; 100 of 101 rows;
    another owner's rows are absent.

#### 47. `internal/rpc/notes/notes.go` (Create)

*   **Type / Scope**: Connect handler, demo. Line 1 `// DEMO: notes`.
*   **Proposed Changes**: `type Handler struct` over `*notes.Store`; `func New(store
    *notes.Store) *Handler`; methods exactly those of `apiv1connect.NotesServiceHandler`, no
    `Unimplemented` embed. Each method: principal from `identity.FromContext` (nil →
    `connect.NewError(connect.CodeUnauthenticated, "no principal")`); for `GetNote` and
    `DeleteNote`, `uuid.Parse` of the id (failure → `apperr.New(apperr.InvalidArgument,
    "NOTE_ID_INVALID", "id is not a UUID")`); one store call with `Subject` as owner, error
    returned unchanged; mapping through one unexported `toProto` (`CreatedAt` →
    `create_time`).

#### 48. `internal/rpc/notes/notes_test.go` (Create)

*   **Type / Scope**: Integration test, demo, `package notes_test`. Line 1 `// DEMO: notes`.
*   **Proposed Changes**: `TestNotes_CRUD` (create, get, list, delete, then get →
    `NOTE_NOT_FOUND`; the stored owner equals the token subject), `TestNotes_OwnerScoping` (get
    and delete by a second user → `not_found` with `NOTE_NOT_FOUND`; that user's list is
    empty), `TestNotes_Validation` (empty title, 201-character title, non-UUID id →
    `invalid_argument` and the violated field), `TestNotes_Unauthenticated`. Reasons are
    asserted as string literals.

#### 49. `internal/server/server.go` (Modify)

*   **Type / Scope**: Composition root. Replaces the chi `Server` struct, `NewWithConfig`,
    `InitDB`, `Initialize` with its seven toggles, the pprof guard and `cors.AllowAll()`.
*   **Proposed Changes**:

    ```go
    type Deps struct {
        DB          *pgxpool.Pool         // required
        Verifier    middleware.Verifier   // required
        CORSOrigins []string              // empty: no cross-origin access
        Metrics     prometheus.Registerer // nil: no histogram
    }
    func Handler(d Deps) (http.Handler, error)
    func New(addr string, h http.Handler) *http.Server
    func Serve(ctx context.Context, grace time.Duration, servers map[*http.Server]net.Listener) error
    ```

    `Handler` is `build(d, routes)`; the unexported `build(d Deps, hub func(*connect.Server,
    *routegroup.Bundle, Deps)) (http.Handler, error)` performs the seven steps of section 3.
    Constants: `maxRequestBytes = 4 << 20`, `readHeaderTimeout = 10 * time.Second`,
    `idleTimeout = 120 * time.Second`. `New` sets `Protocols` (`SetHTTP1(true)`,
    `SetUnencryptedHTTP2(true)`), the two timeouts and
    `ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn)`; `ReadTimeout` and
    `WriteTimeout` stay 0. `Serve` serves each server on its listener until one fails or `ctx`
    ends, then calls `Shutdown` on all of them concurrently with a fresh context of `grace`,
    calls `Close` on a server whose `Shutdown` failed, and returns the joined errors.

#### 50. `internal/server/routes.go` (Create)

*   **Type / Scope**: Hub.
*   **Proposed Changes**: `func routes(rpc *connect.Server, plain *routegroup.Bundle, d Deps)`
    with two lines: `mountWhoAmI(rpc, plain, d)` and `mountNotes(rpc, plain, d) // DEMO:line`.
    Its comment does not name the demo.

#### 51. `internal/server/routes_whoami.go` (Create)

*   **Type / Scope**: Mount file, permanent.
*   **Proposed Changes**: `func mountWhoAmI(rpc *connect.Server, _ *routegroup.Bundle, _ Deps)`
    registers `whoami.Handler{}` with `apiv1connect.RegisterWhoAmIServiceHandler`.

#### 52. `internal/server/routes_notes.go` (Create)

*   **Type / Scope**: Mount file, demo. Line 1 `// DEMO: notes`.
*   **Proposed Changes**: `func mountNotes(rpc *connect.Server, _ *routegroup.Bundle, d Deps)`
    registers `rpcnotes.New(notes.NewStore(d.DB))` with
    `apiv1connect.RegisterNotesServiceHandler`.

#### 53. `internal/server/health.go` (Create)

*   **Type / Scope**: Probes.
*   **Proposed Changes**: Two unexported consumer-side interfaces, `pinger` (`Ping(context.Context)
    error`) and `readier` (`Ready(context.Context) error`). `healthz`: 200, empty body.
    `readyz(db pinger, verifier any)`: ping with a 2 s timeout → on error 503, body `database
    unavailable`, WARN `database not ready`; then, when `verifier` implements `readier`, its
    `Ready` → on error 503, body `identity provider unavailable`, WARN `identity provider not
    ready`; otherwise 200, empty body. The real error goes to the log only.

#### 54. `internal/server/management.go` (Create)

*   **Type / Scope**: Management handler.
*   **Proposed Changes**: `func Management(g prometheus.Gatherer) http.Handler`: its own
    `http.ServeMux` with `GET /metrics` (`promhttp.HandlerFor`) and the `net/http/pprof`
    handlers under `/debug/pprof/`. No middleware.

#### 55. `internal/server/export_test.go` (Create)

*   **Type / Scope**: Test seam.
*   **Proposed Changes**: Exposes `Build` (= `build`), `Routes` (= `routes`) and `Readyz`
    (= `readyz`) to `package server_test`. `Readyz` is there because `Deps.DB` is a concrete
    pool, so a failing `pinger` cannot be passed through `Deps`.

#### 56. `internal/server/testroutes_test.go` (Create)

*   **Type / Scope**: Test helper, `package server_test`.
*   **Proposed Changes**: `newHandler(t, change ...func(*server.Deps)) (http.Handler,
    *test.Server)` takes its `Deps` and its `*test.Server` from `test.NewDeps`, applies
    `change`, and calls `server.Build` once with a hub that calls `server.Routes` and then
    registers the test procedures. `newServer(t, change...)` serves that handler the way
    `test.NewServer` does (`httptest.NewUnstartedServer`, `Protocols` from `server.New("",
    nil)`), sets `URL` and returns the `*test.Server`, so `Token`, `Client`, `Provider` and
    `Metrics` work as in any harness test. The file imports `internal/test` and not
    `internal/identity/testprovider`. The test procedures are registered by hand with
    `rpc.Register(connect.Method{...})` under `/test.v1.TestService/`: `Fail` (unary; modes
    `apperr`, `connect`, `plain`, `panic`, `wait`), `Tick` (server stream), `Sum` (client
    stream); the plain routes are `GET /test/plain`, `GET /test/panic`, `POST /test/echo`. No
    proto file, no mention of the demo.

#### 57. `internal/server/routing_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestRouting` (probes without a token; 404 and 405 without auth; an
    RPC with and without a token; Connect GET on a `NO_SIDE_EFFECTS` method; an unknown method
    is 401 without a token and 501 with one; wrong content type 415; invalid JSON 400; a plain
    route with and without a token; a panicking plain route 500; `X-Request-Id` and the
    security headers on every response), `TestRequestID`, `TestCORS`, `TestRoutegroupOrdering`
    (Mount once and last), `TestBodyLimit`.

#### 58. `internal/server/streaming_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestStreaming_Flushes` (HTTP/1.1 and h2c; a server stream longer
    than 1 s completes, which also proves no write timeout is set),
    `TestStreaming_WrapperWithoutFlush`.

#### 59. `internal/server/serve_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestNew` (protocols, the two timeouts, zero read and write
    timeouts), `TestServe` (three protocols on one port: Connect, gRPC, gRPC-Web; metrics and
    pprof only on the management listener; a request in flight at shutdown finishes),
    `TestServe_GraceExceeded` (an open stream is cut at the grace and `Serve` returns an
    error), `TestServe_StopsAllWhenOneFails`.

#### 60. `internal/server/logs_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestAccessLog`: one line per request with the fields of section 4,
    no line for probes, auth failures carry the code. `TestMetrics`: an RPC is observed with
    its procedure and code; an auth rejection with an empty procedure and `unauthenticated`; a
    provider outage with `unavailable`; an unknown method and a plain route produce no series.

#### 61. `internal/server/errors_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestErrors` over four client configurations (Connect, Connect JSON,
    gRPC, gRPC-Web), unary and streaming; a panic becomes `internal`; no internal text reaches
    a response.

#### 62. `internal/server/health_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestHealthz`. `TestReadyz` calls `server.Readyz` (entry 55)
    directly: a fake `pinger` returning a driver-looking error yields `database unavailable`; a verifier whose `Ready` fails yields
    `identity provider unavailable`; a verifier without `Ready` yields 200; neither error text
    reaches the body.

#### 63. `internal/server/auth_test.go` (Create)

*   **Type / Scope**: Test, full stack.
*   **Proposed Changes**: `TestAuth_GeneratedClients`: nine token cases (no token; not a JWT;
    expired; wrong audience; wrong issuer; no `sub`; ID token; unknown `kid`; provider down)
    on four client configurations, unary and streaming. `TestAuth_Wire` (every row of the
    auth-on-the-wire table of section 7, both `WWW-Authenticate` forms),
    `TestAuth_AnswersBeforeBody`, `TestAuth_Principal`, `TestAuth_DevPrincipal`
    (`identity.DevVerifier` through the stack, no token).

#### 64. `internal/server/middleware/middleware.go` (Create)

*   **Type / Scope**: HTTP middleware.
*   **Proposed Changes**:

    ```go
    func RequestID(next http.Handler) http.Handler
    func AccessLog(reg prometheus.Registerer, skip ...string) func(http.Handler) http.Handler
    func Recover(next http.Handler) http.Handler
    func SecurityHeaders(next http.Handler) http.Handler
    func CORS(origins []string) func(http.Handler) http.Handler
    func BodyLimit(n int64) func(http.Handler) http.Handler
    ```

    Behaviour as in section 4. Literals: the request id pattern `^[A-Za-z0-9._-]{1,64}$`; the
    histogram is one `prometheus.HistogramVec` named `rpc_server_duration_seconds` with labels
    `procedure` and `code` and `prometheus.DefBuckets`, registered with `MustRegister` when
    `reg` is not nil; preflight answers carry `Access-Control-Max-Age: 7200`; `Recover` answers
    `Internal Server Error`.

#### 65. `internal/server/middleware/middleware_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestRequestID`, `TestAccessLog_SkipsProbes`,
    `TestAccessLog_WriterKeepsFlush`, `TestAccessLog_Histogram` (one observation when the
    carrier has a code, none without, labels from the carrier), `TestRecover`,
    `TestSecurityHeaders`, `TestCORS`, `TestBodyLimit`.

#### 66. `internal/server/middleware/auth.go` (Create)

*   **Type / Scope**: HTTP middleware.
*   **Proposed Changes**:

    ```go
    type Verifier interface {
        Verify(ctx context.Context, rawToken string) (*identity.Principal, error)
    }
    func Auth(v Verifier) func(http.Handler) http.Handler
    ```

    Behaviour as in section 7. Literals: client messages `invalid or missing bearer token` and
    `identity provider unavailable`; log records DEBUG `token rejected` and ERROR `identity
    provider unavailable`, each with `error`; header values `Bearer` and
    `Bearer error="invalid_token"`.

#### 67. `internal/server/middleware/auth_test.go` (Create)

*   **Type / Scope**: Test with a stub verifier.
*   **Proposed Changes**: `TestAuth`, a header table: valid; scheme in any case; no header,
    another scheme, empty token → 401 and `WWW-Authenticate: Bearer`; rejected token → 401 and
    `Bearer error="invalid_token"`; outage → 503 and no header; nil principal with nil error →
    401. The verifier is called with `""` when there is no token; the carrier has the code.

#### 68. `internal/server/config/config.go` (Create)

*   **Type / Scope**: Config.
*   **Proposed Changes**:

    ```go
    type Config struct {
        Environment        string
        AuthProvider       string
        AuthDevRoles       []string
        KeycloakIssuerURL  string
        KeycloakAudience   string
        DatabaseURL        string
        ListenAddr         string
        ManagementAddr     string
        CORSAllowedOrigins []string
        LogLevel           slog.Level
    }
    func Load() (Config, error)
    func (c Config) Validate() error
    ```

    The ten variables, defaults and syntax rules of the table in section 8; one unexported
    list of the variable names, used by the reader and by the env-file test; the unexported
    `parse(lookup func(string) (string, bool)) (Config, error)`. Messages: `AUTH_PROVIDER "x"
    is not supported: use "keycloak" or "none"` (syntax); `AUTH_PROVIDER is required: set
    "keycloak", or "none" for local development`; `AUTH_PROVIDER=none is refused when
    APP_ENVIRONMENT is "<value>": allowed only in development, dev, local, test`;
    `KEYCLOAK_ISSUER_URL is required when AUTH_PROVIDER=keycloak`; `KEYCLOAK_AUDIENCE is
    required when AUTH_PROVIDER=keycloak`; `DATABASE_URL is required`; `MANAGEMENT_ADDR must
    differ from LISTEN_ADDR`.

#### 69. `internal/server/config/dotenv.go` (Create)

*   **Type / Scope**: Config.
*   **Proposed Changes**: One unexported function that parses `.env.local` in the working
    directory into a map, with the format of section 8. A missing file yields an empty map; a
    malformed line is an error that names the line number.

#### 70. `internal/server/config/config_test.go` (Create)

*   **Type / Scope**: Test, `package config`.
*   **Proposed Changes**: `TestParse` (defaults, every syntax rule, every violation reported
    in one error). `TestValidate`: unset provider; `none` with unset, `production`, `prod`,
    `Production`, `staging` refused and with `development`, `dev`, `local`, `test`, ` Dev `
    accepted; `keycloak` without issuer or audience; unset `DATABASE_URL`; equal addresses.
    `TestLoad_EnvironmentWinsOverDotenv`, `TestLoad_NoFile`, `TestDotenv`.
    `TestEnvFilesListEveryVariable`: `.env.example` (set or commented) and
    `docs/agents/env-reference.md` name exactly the reader's variables.

#### 71. `internal/test/postgres.go` (Create)

*   **Type / Scope**: Test harness.
*   **Proposed Changes**:

    ```go
    func Database(t *testing.T) string      // URL of a fresh database cloned from the migrated template
    func EmptyDatabase(t *testing.T) string // URL of a fresh database with no migration applied
    ```

    One container per process behind `sync.Once`: `postgres:17-alpine` (one constant) through
    `testcontainers.GenericContainer` with `-c fsync=off -c full_page_writes=off`; a start
    failure is cached. Then `CREATE DATABASE app_template` and `db.Migrate(ctx, url, "up",
    io.Discard)`. `Database`: `CREATE DATABASE "t<seq>_<test name>" TEMPLATE app_template`, the
    name lower-cased, non-alphanumerics replaced by `_`, cut to 63 bytes; cleanup runs `DROP
    DATABASE ... WITH (FORCE)`. `EmptyDatabase` is the same without `TEMPLATE`.

#### 72. `internal/test/postgres_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestDatabase_Parallel` (8 parallel subtests, each sees only its own
    row), `TestDatabase_LongName` (a name over 63 bytes).

#### 73. `internal/test/server.go` (Create)

*   **Type / Scope**: Test harness.
*   **Proposed Changes**:

    ```go
    type Server struct {
        URL      string
        DB       *pgxpool.Pool
        Provider *testprovider.Provider
        Metrics  *prometheus.Registry
    }
    func NewServer(t *testing.T, change ...func(*server.Deps)) *Server
    func NewHandler(t *testing.T, change ...func(*server.Deps)) (http.Handler, *Server)
    func NewDeps(t *testing.T) (server.Deps, *Server)
    func (s *Server) Token(subject string, roles ...string) string
    func (s *Server) Client(token string, opts ...connecthttp.Option) *connect.Client
    func HTTPClient(token string) *http.Client
    func Reason(err error) string                     // ErrorInfo reason of a Connect error, or ""
    func Violations(t *testing.T, err error) []string // field names of the protovalidate violations
    ```

    `NewDeps` opens a pool on `Database(t)`, starts a `testprovider`, and returns
    `server.Deps{DB, Verifier: keycloak.New(provider.URL(), testprovider.Audience), Metrics}`
    with the `*Server` that holds the same pool, provider and registry. `NewHandler` is
    `NewDeps`, then `change`, then `server.Handler`. `NewDeps` is exported for the one caller
    that needs another hub, the `internal/server` suite (entry 56): a handler is built once per
    `Deps`, because the histogram is registered on `Deps.Metrics` with `MustRegister`.
    `NewServer` serves the handler with `httptest.NewUnstartedServer`, taking `Protocols` from
    `server.New("", nil)`. `Token` is `Provider.Token(Provider.Claims(subject, roles...))`. `HTTPClient` speaks
    h2c and adds the header in a `RoundTripper`; an empty token sends none.

#### 74. `internal/test/server_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestReason`, `TestViolations` on constructed `connect.Error` values.

#### 75. `internal/test/smoke_test.go` (Create)

*   **Type / Scope**: Test, `//go:build smoke`.
*   **Proposed Changes**: `TestSmoke_WhoAmI`. Reads `KEYCLOAK_ISSUER_URL`, falling back to
    `http://localhost:8181/realms/app`; builds `keycloak.New(issuer, "api")` and requires
    `Ready` to succeed, failing with `Keycloak is not reachable: run make keycloak-up`; starts
    `test.NewServer` with that verifier. Tokens come from the realm's token endpoint: password
    grant on `app-test` with `scope=openid`, client credentials for `app-service` with
    `dev-only-secret`. Six cases through the generated client: alice (username `alice`, name
    `Alice Admin`, roles contain `admin` and `user`); bob (roles contain `user`, not `admin`);
    the service account (username `service-account-app-service`, empty email and name); alice's
    ID token, alice's refresh token and no token are `unauthenticated`.

#### 76. `main.go` (Modify)

*   **Type / Scope**: Entry point.
*   **Proposed Changes**: The blank `github.com/lib/pq` import is removed. The file imports
    `<module>/cmd` only and calls `cmd.Execute()`.

#### 77. `cmd/root.go` (Modify)

*   **Type / Scope**: CLI.
*   **Proposed Changes**: `var version = "dev"`, set by `-X <module>/cmd.version=<v>`. Root
    command with `Use: filepath.Base(os.Args[0])`, `SilenceUsage`, `SilenceErrors`,
    `Version: version`. `Execute()`: installs the JSON logger at INFO;
    `signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)` with
    `context.AfterFunc(ctx, stop)`, so a second signal kills the process; `ExecuteContext`; on
    error logs ERROR `command failed` with `error` and exits 1. `loadConfig()` calls
    `config.Load()` and re-installs the logger at `cfg.LogLevel`. Removed: `fmt.Println`, the
    "RESTful JSON service" text, the `config.ModuleName` reference.

#### 78. `cmd/run.go` (Modify)

*   **Type / Scope**: CLI.
*   **Proposed Changes**: The `run` command performs the ten steps of section 9. Constants:
    `dbOpenTimeout = 10 * time.Second`, `shutdownGrace = 25 * time.Second`. The registry is
    `prometheus.NewRegistry()` with `collectors.NewGoCollector()` and
    `collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})`; it is passed as
    `Deps.Metrics` and to `server.Management`. Listeners are bound with
    `net.ListenConfig.Listen`. The issuer-mismatch error is printed as `KEYCLOAK_ISSUER_URL:
    <go-oidc's message with both URLs>`. Replaces the signal channel written by hand.

#### 79. `cmd/db.go` (Modify)

*   **Type / Scope**: CLI.
*   **Proposed Changes**: `db` with one subcommand `migrate`, which has exactly `up`, `down`,
    `status`, built from one table. Each calls `loadConfig()`, fails with `DATABASE_URL is
    required` when it is unset, then `db.Migrate(cmd.Context(), cfg.DatabaseURL, <name>,
    cmd.OutOrStdout())`. No `os.Exit` in a `Run` function. The `Short` typo ("erform") and the
    `Long` text "drop or migrate a module" are corrected.

#### 80. `cmd/probe.go` (Create)

*   **Type / Scope**: CLI.
*   **Proposed Changes**: `probe`: `loadConfig()`; `GET http://<host>:<port>/readyz` with a
    2 s client timeout, host and port from `LISTEN_ADDR`, an empty or unspecified host (`0.0.0.0`,
    `::`) replaced by `127.0.0.1`; returns nil on 200 and an error otherwise.

#### 81. `cmd/cmd_test.go` (Create)

*   **Type / Scope**: Test.
*   **Proposed Changes**: `TestProbe` (200 → nil; 503, timeout and refused connection →
    error; host rewriting), `TestRun_ConfigErrors` (with no environment the error names
    `AUTH_PROVIDER is required` and `DATABASE_URL is required`, and no connection is
    attempted), `TestDBMigrate_NeedsOnlyDatabaseURL`.

#### 82. `.env.example` (Modify)

*   **Type / Scope**: Dev env file.
*   **Proposed Changes**: Rewritten; every variable of section 8 once. Header comment: this is
    the dev environment, `make run` copies it to `.env.local`, and the process environment
    wins over the file. Set: `APP_ENVIRONMENT=development`, `AUTH_PROVIDER=none`,
    `AUTH_DEV_ROLES=admin`, `DATABASE_URL=postgres://app:app@localhost:5434/app?sslmode=disable`,
    `KEYCLOAK_ISSUER_URL=http://localhost:8181/realms/app`, `KEYCLOAK_AUDIENCE=api`,
    `LISTEN_ADDR=127.0.0.1:8080`, `MANAGEMENT_ADDR=127.0.0.1:9090`. Commented with their
    defaults: `#CORS_ALLOWED_ORIGINS=`, `#LOG_LEVEL=INFO`.

#### 83. `docs/agents/env-reference.md` (Modify)

*   **Type / Scope**: Reference doc. It is in stage 1 because `TestEnvFilesListEveryVariable`
    reads it.
*   **Proposed Changes**: Rewritten as the table of section 8 with the same five columns
    (Variable, Type, Built-in default, Required, Syntax and meaning) and the five `Validate`
    rules below it.

#### 84. `docker-compose.yaml` (Modify)

*   **Type / Scope**: Dev compose file.
*   **Proposed Changes**: Rewritten. Header comment: dev only, throwaway credentials. `name:
    app` (the line init rewrites). Service `postgres`: `postgres:17-alpine`;
    `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` all `app`; port `127.0.0.1:5434:5432`;
    volume `postgres-data:/var/lib/postgresql/data`; health check `pg_isready -h 127.0.0.1 -U
    app -d app` every 2 s, 30 retries. Service `keycloak`:
    `quay.io/keycloak/keycloak:26.8.2`; command `start-dev --import-realm --http-port=8181`;
    `KC_BOOTSTRAP_ADMIN_USERNAME` and `KC_BOOTSTRAP_ADMIN_PASSWORD` `admin`, `KC_HOSTNAME`
    `http://localhost:8181`, `KC_HEALTH_ENABLED` `"true"`; port `127.0.0.1:8181:8181`; volume
    `./keycloak/realms:/opt/keycloak/data/import:ro`; health check a `/dev/tcp` request for
    `/health/ready` on port 9000 inside the container, every 5 s, 36 retries. One named volume
    `postgres-data`. Removed: `container_name`, `restart`, the fixed 5432 and 8080 ports, the
    `./docker-data` mounts, the custom network, Keycloak's database settings and `depends_on`,
    `POSTGRES_MULTIPLE_DATABASES`, the `init-db.sh` mount, the Postgres tuning flags.

#### 85. `keycloak/realms/app.json` (Create)

*   **Type / Scope**: Keycloak realm import, dev only.
*   **Proposed Changes**: Realm `app`, enabled, `revokeRefreshToken: true`,
    `refreshTokenMaxReuse: 0`. Realm roles `admin` and `user`. Clients, each with one protocol
    mapper `oidc-audience-mapper` (`included.custom.audience: api`, `access.token.claim: true`,
    `id.token.claim: false`): `app-web` (public; standard flow only; PKCE `S256`; redirect URIs
    and post-logout URIs `http://localhost:3000/*`; web origin `http://localhost:3000`);
    `app-service` (confidential, secret `dev-only-secret`; service accounts only); `app-test`
    (public; direct access grants only). Users `alice` (`alice@example.test`, first name
    `Alice`, last name `Admin`, realm roles `admin` and `user`) and `bob` (`bob@example.test`,
    `Bob`, `User`, realm role `user`), both enabled, email verified, password `password`, not
    temporary. No `clientScopes`, `defaultClientScopes`, `components` or `id` fields.

#### 86. `Dockerfile` (Modify)

*   **Type / Scope**: Container image; the file is empty today.
*   **Proposed Changes**: Build stage `FROM --platform=$BUILDPLATFORM golang:1.27`: copy
    `go.mod` and `go.sum`, `go mod download` with a module cache mount, copy the context, then
    with `ARG TARGETOS TARGETARCH VERSION=dev` build `CGO_ENABLED=0 GOOS=$TARGETOS
    GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w -X $(go list -m)/cmd.version=$VERSION"
    -o /out/app .`. Final stage `FROM gcr.io/distroless/static-debian12:nonroot`: `COPY` the
    binary to `/app`, `USER 65532:65532`, `EXPOSE 8080`,
    `HEALTHCHECK --interval=5s --timeout=3s --retries=3 CMD ["/app", "probe"]`,
    `ENTRYPOINT ["/app"]`, `CMD ["run"]`. No `# syntax=` line.

#### 87. `.dockerignore` (Create)

*   **Type / Scope**: Build context allow-list.
*   **Proposed Changes**: `*`, then `!go.mod`, `!go.sum`, `!main.go`, `!cmd/`, `!internal/`,
    `!sql/`, then `**/.env*` and `**/*_test.go`.

#### 88. `scripts/image-smoke.sh` (Create)

*   **Type / Scope**: Bash, called by `make image-smoke`.
*   **Proposed Changes**: Arguments `IMAGE NAME VERSION`. Prerequisites: Docker with the
    compose plugin; the image is built (the Make target depends on `image`); the script starts
    Postgres itself with `docker compose up -d --wait postgres` and leaves it running, as
    `make dev-up` does (`scripts/template-smoke.sh` removes it in its own teardown). Checks,
    in order: `docker image inspect` reports user `65532:65532`; `docker run --rm IMAGE
    --version` contains `VERSION`. Case 1: `docker run --rm IMAGE` with no environment exits
    non-zero and its output contains `AUTH_PROVIDER is required`. Case 2, with `--network
    NAME_default -e APP_ENVIRONMENT=development -e AUTH_PROVIDER=none -e
    DATABASE_URL=postgres://app:app@postgres:5432/app?sslmode=disable`: `docker run --rm ...
    IMAGE db migrate up` exits 0; `docker run -d --name NAME-smoke ... IMAGE` reaches health
    status `healthy` within 60 s, polled with `docker inspect -f '{{.State.Health.Status}}'`. A
    trap prints the container's logs and removes it. No token is obtained.

#### 89. `scripts/sync-skills.sh` (Modify)

*   **Type / Scope**: Bash. It is in stage 1 because `make check` calls it.
*   **Proposed Changes**: With no argument it mirrors as today. With `--check` it runs `diff
    -qr .claude/skills agents/skills`; on a difference it prints `agents/skills is stale: run
    make sync-skills` and exits 1. Any other argument is a usage error with exit 2.

Deletions of stage 1. Each file below is removed in full.

#### 90. `cmd/db_migrate.go` (Delete)

*   **Type / Scope**: CLI.
*   **Proposed Changes**: Deleted; folded into `cmd/db.go`. It used `sql.Open("postgres")`,
    `goose.Run` without a context and on-disk migrations, offered `reset`, and its `Long` text
    advertised `create`.

#### 91. `cmd/db_seed.go` (Delete)

*   **Type / Scope**: CLI.
*   **Proposed Changes**: Deleted. It only pings and prints success.

#### 92. `internal/api/handlers/health.go` (Delete)

*   **Type / Scope**: REST handler.
*   **Proposed Changes**: Deleted; `internal/server/health.go` replaces it. It returned the raw
    database error and the build string.

#### 93. `internal/api/handlers/notes.go` (Delete)

*   **Type / Scope**: REST handler, demo.
*   **Proposed Changes**: Deleted; `internal/rpc/notes` and `internal/domain/notes` replace it.

#### 94. `internal/api/handlers/notes_test.go` (Delete)

*   **Type / Scope**: Test on a fake `Querier`.
*   **Proposed Changes**: Deleted; the Postgres-backed demo tests replace it.

#### 95. `internal/api/handlers/users.go` (Delete)

*   **Type / Scope**: REST handler.
*   **Proposed Changes**: Deleted; `WhoAmIService` replaces `GET /v1/users/me`. The users list
    was the admin client's only caller and has no successor.

#### 96. `internal/api/openapi_server.gen.go` (Delete)

*   **Type / Scope**: Generated chi server.
*   **Proposed Changes**: Deleted.

#### 97. `internal/api/openapi_types.gen.go` (Delete)

*   **Type / Scope**: Generated OpenAPI types.
*   **Proposed Changes**: Deleted; `internal/gen/api/v1` replaces it.

#### 98. `internal/api/router/routes.go` (Delete)

*   **Type / Scope**: chi hub.
*   **Proposed Changes**: Deleted; `internal/server/routes.go` replaces it.

#### 99. `internal/api/router/routes_health.go` (Delete)

*   **Type / Scope**: chi routes.
*   **Proposed Changes**: Deleted; probes are registered in `internal/server/server.go`.

#### 100. `internal/api/router/routes_notes.go` (Delete)

*   **Type / Scope**: chi routes, demo.
*   **Proposed Changes**: Deleted; `internal/server/routes_notes.go` replaces it.

#### 101. `internal/api/router/routes_users.go` (Delete)

*   **Type / Scope**: chi routes.
*   **Proposed Changes**: Deleted; `internal/server/routes_whoami.go` replaces it.

#### 102. `internal/server/auth/auth.go` (Delete)

*   **Type / Scope**: chi-era auth middleware and guards.
*   **Proposed Changes**: Deleted; `internal/server/middleware/auth.go` replaces `Authenticate`.
    `RequireRealmRoles`, `RequireClientRoles`, `RequireOrganization`, `RequireAnyOrganization`,
    `RequireScopes`, `ModeOptional` and the header and scheme options have no caller and no
    successor.

#### 103. `internal/server/auth/auth_test.go` (Delete)

*   **Type / Scope**: Test.
*   **Proposed Changes**: Deleted; `internal/server/middleware/auth_test.go` and
    `internal/server/auth_test.go` replace it.

#### 104. `internal/server/auth/context.go` (Delete)

*   **Type / Scope**: Context helpers.
*   **Proposed Changes**: Deleted; `identity.WithPrincipal` and `identity.FromContext` replace
    `PrincipalFromContext`; `TokenFromContext` has no successor.

#### 105. `internal/server/auth/dev.go` (Delete)

*   **Type / Scope**: Dev principal.
*   **Proposed Changes**: Deleted; `identity.DevVerifier` replaces it.

#### 106. `internal/server/auth/dev_test.go` (Delete)

*   **Type / Scope**: Test.
*   **Proposed Changes**: Deleted; `TestDevVerifier` replaces it.

#### 107. `internal/server/auth/errors.go` (Delete)

*   **Type / Scope**: Auth errors.
*   **Proposed Changes**: Deleted; the two `identity` sentinels replace them.

#### 108. `internal/server/config/auth_config.go` (Delete)

*   **Type / Scope**: Auth config and startup validation.
*   **Proposed Changes**: Deleted; the dev allow-list moves into `config.go` unchanged.

#### 109. `internal/server/config/auth_config_test.go` (Delete)

*   **Type / Scope**: Test.
*   **Proposed Changes**: Deleted; its cases are carried into `TestValidate`.

#### 110. `internal/server/config/build_args.go` (Delete)

*   **Type / Scope**: Build placeholders.
*   **Proposed Changes**: Deleted; nothing fills them. `cmd.version` replaces them.

#### 111. `internal/server/config/db_config.go` (Delete)

*   **Type / Scope**: `PG*` DSN builder.
*   **Proposed Changes**: Deleted; `DATABASE_URL` replaces it.

#### 112. `internal/server/config/server_config.go` (Delete)

*   **Type / Scope**: Nested config structs.
*   **Proposed Changes**: Deleted; `config.go` replaces it.

#### 113. `internal/server/config/env/env.go` (Delete)

*   **Type / Scope**: `GetEnv*` helpers.
*   **Proposed Changes**: Deleted. They fall back silently on malformed values, and
    `GetProjectRootDir` resolves next to the binary, so `./bin/app` looks for `.env.local` and
    migrations under `bin/`. The new reader has no project root.

#### 114. `internal/server/errs/error.go` (Delete)

*   **Type / Scope**: REST problem-JSON error shape.
*   **Proposed Changes**: Deleted. It copies `err.Error()` into the response. `apperr` and the
    `Errors` interceptor replace it.

#### 115. `internal/server/errs/error_test.go` (Delete)

*   **Type / Scope**: Test.
*   **Proposed Changes**: Deleted with the package.

#### 116. `internal/server/middleware/cache_control.go` (Delete)

*   **Type / Scope**: HTTP middleware.
*   **Proposed Changes**: Deleted. Blanket `no-store` defeats Connect GET caching.

#### 117. `internal/server/middleware/logger.go` (Delete)

*   **Type / Scope**: HTTP middleware.
*   **Proposed Changes**: Deleted; `AccessLog` replaces it. It needs chi's response wrapper,
    logs the URL with its query string, buffers bodies and reports nanoseconds as
    `duration_ms`.

#### 118. `internal/server/middleware/no_cache.go` (Delete)

*   **Type / Scope**: HTTP middleware, no caller.
*   **Proposed Changes**: Deleted.

#### 119. `internal/server/middleware/noop.go` (Delete)

*   **Type / Scope**: HTTP middleware, no caller.
*   **Proposed Changes**: Deleted.

#### 120. `internal/server/middleware/secure.go` (Delete)

*   **Type / Scope**: HTTP middleware.
*   **Proposed Changes**: Deleted; the nine knobs become constants in `SecurityHeaders`.

#### 121. `internal/server/test/test_db.go` (Delete)

*   **Type / Scope**: Test harness with no caller.
*   **Proposed Changes**: Deleted; `internal/test/postgres.go` replaces it. It cannot run as
    written (unregistered driver, wrong host, disk path, one unpinned container per call).

#### 122. `internal/server/test/test_server.go` (Delete)

*   **Type / Scope**: Test harness with no caller.
*   **Proposed Changes**: Deleted; `internal/test/server.go` replaces it. It has no listener and
    no token.

#### 123. `keycloak/realms/myrealm.json` (Delete)

*   **Type / Scope**: 87 KB realm export.
*   **Proposed Changes**: Deleted; `keycloak/realms/app.json` replaces it. It has no
    `oidc-audience-mapper` that adds `api` and no `admin` realm role, and it ships private keys,
    a client secret and password hashes.

#### 124. `migrations/.gitkeep` (Delete)

*   **Type / Scope**: Placeholder.
*   **Proposed Changes**: Deleted; the directory moves to `sql/migrations` so it can be
    embedded.

#### 125. `migrations/0001_notes.sql` (Delete)

*   **Type / Scope**: Migration, demo.
*   **Proposed Changes**: Deleted; `sql/migrations/00001_notes.sql` replaces it.

#### 126. `oapi/openapi.yaml` (Delete)

*   **Type / Scope**: OpenAPI spec.
*   **Proposed Changes**: Deleted; `proto/api/v1/` replaces it.

#### 127. `pkg/db/db.go` (Delete)

*   **Type / Scope**: `database/sql` transaction wrapper, no importer.
*   **Proposed Changes**: Deleted; `pgx.BeginFunc` is the transaction helper.

#### 128. `pkg/db/db_test.go` (Delete)

*   **Type / Scope**: Test.
*   **Proposed Changes**: Deleted with the package.

#### 129. `pkg/db/ts_vector.go` (Delete)

*   **Type / Scope**: Full-text helper, no importer.
*   **Proposed Changes**: Deleted.

#### 130. `pkg/dotenv/dotenv.go` (Delete)

*   **Type / Scope**: Dotenv library.
*   **Proposed Changes**: Deleted; a reduced parser is `internal/server/config/dotenv.go`.

#### 131. `pkg/dotenv/dotenv_test.go` (Delete)

*   **Type / Scope**: Test.
*   **Proposed Changes**: Deleted; `TestDotenv` replaces it.

#### 132. `pkg/dotenv/testdata/.env.features` (Delete)

*   **Type / Scope**: Test fixture.
*   **Proposed Changes**: Deleted with the package.

#### 133. `pkg/dotenv/testdata/.env.local.malformed` (Delete)

*   **Type / Scope**: Test fixture.
*   **Proposed Changes**: Deleted with the package.

#### 134. `pkg/dotenv/testdata/.env.local.sample` (Delete)

*   **Type / Scope**: Test fixture.
*   **Proposed Changes**: Deleted with the package.

#### 135. `pkg/dotenv/testdata/.env1.local` (Delete)

*   **Type / Scope**: Test fixture.
*   **Proposed Changes**: Deleted with the package.

#### 136. `pkg/dotenv/testdata/.env2.local` (Delete)

*   **Type / Scope**: Test fixture.
*   **Proposed Changes**: Deleted with the package.

#### 137. `pkg/keycloak/admin.go` (Delete)

*   **Type / Scope**: Keycloak Admin REST client, 191 lines.
*   **Proposed Changes**: Deleted. A resource server needs no realm-wide read credential.

#### 138. `pkg/keycloak/errors.go` (Delete)

*   **Type / Scope**: Verifier errors.
*   **Proposed Changes**: Deleted; the two `identity` sentinels replace them.

#### 139. `pkg/keycloak/jwks.go` (Delete)

*   **Type / Scope**: The template's own JWKS cache.
*   **Proposed Changes**: Deleted; go-oidc's key set with the wrapper of entry 34 replaces it.

#### 140. `pkg/keycloak/jwt.go` (Delete)

*   **Type / Scope**: The template's own JWT parsing and claims.
*   **Proposed Changes**: Deleted. The principal fields `EmailVerified`, `GivenName`,
    `FamilyName`, `ClientRoles`, `Organizations`, `Scopes`, `Claims` and `Token` go with it.

#### 141. `pkg/keycloak/keycloak.go` (Delete)

*   **Type / Scope**: The template's own verifier.
*   **Proposed Changes**: Deleted; `internal/identity/keycloak` replaces it.

#### 142. `pkg/keycloak/keycloak_test.go` (Delete)

*   **Type / Scope**: Test.
*   **Proposed Changes**: Deleted with the package.

#### 143. `pkg/tests/test.go` (Delete)

*   **Type / Scope**: `RunningInTest()` helper.
*   **Proposed Changes**: Deleted. Its one caller was the `.env.local` skip in the old config;
    it counted any process with `CI` set as a test.

#### 144. `pkg/tests/test_test.go` (Delete)

*   **Type / Scope**: Test.
*   **Proposed Changes**: Deleted with the package. `pkg/` no longer exists.

#### 145. `scripts/init-db.sh` (Delete)

*   **Type / Scope**: Postgres init script.
*   **Proposed Changes**: Deleted. Its only job was creating Keycloak's database.

#### 146. `sql/queries/README.md` (Delete)

*   **Type / Scope**: Doc.
*   **Proposed Changes**: Deleted. It points at a skill that does not exist; `go-feature`
    documents the directory.

**Stage 2 — docs and skills.** Gate: `scripts/sync-skills.sh --check` passes and the two
acceptance greps of entry 173 print nothing. `make check` still passes.

#### 147. `README.md` (Modify)

*   **Type / Scope**: Template README. Init replaces it with a stub.
*   **Proposed Changes**: Rewritten with these sections, in order:
    1.  Title and two sentences: a Go-only GitHub template for Connect-RPC backends; what `make
        init` does.
    2.  Quickstart. Prerequisites: Go 1.26 or newer (the `toolchain` line fetches 1.27.x),
        Docker, GNU Make, git, curl, a C compiler, `gh` for labels. Commands: `make tools`,
        `scripts/init.sh --module <path> --name <name> --yes` (or `make init`), `make dev-up`,
        `make migrate-up`, `make run`, the two `curl` calls of CUJ 2. A second block shows the
        template run without init, with one sentence: run `make dev-down` before init if you
        ran `make migrate-up` on the uninitialised template.
    3.  What you get: the table of section 20.
    4.  Calling the API without generated code: every RPC accepts
        `POST /<package>.<Service>/<Method>` with a JSON body; side-effect-free methods also
        accept GET; errors are JSON with `code` and `message`. The last commit of the chi +
        OpenAPI version is the tag `rest-chi-openapi`; it is not maintained.
    5.  Auth: `none` versus `keycloak`; a dev token with `make -s token`.
    6.  Building a feature: one sentence and a link to `AGENTS.md` §8. No layout.
    7.  The AI workflow, as today, with the skill counts corrected.
    8.  After "Use this template": what does not come along (labels, secrets, settings, branch
        protection); what is live at once (CI on the first push, Dependabot); two projects on
        one machine need different host ports; `gonew` is not supported.
    9.  Maintaining this template: `git push origin rest-chi-openapi`,
        `gh repo edit --template`, the three `scripts/template-smoke.sh` commands of section
        18 with their precondition (`make dev-down` first, so port 5434 is free; one leg at a
        time), bumping the Makefile pins, the `toolchain` line and `golang:` together; GitHub
        disables scheduled workflows after 60 days without repository activity, re-enable them.
    10. Licence: the template is MIT; the author asks nothing of projects generated from it;
        init deletes `LICENSE`, and a generated project adds its own.

#### 148. `AGENTS.md` (Modify)

*   **Type / Scope**: Agent instructions, single source of truth.
*   **Proposed Changes**: Rewritten. Target headings, listed here once:

    | Heading | Content |
    |---|---|
    | `# AGENTS.md` | no repository name |
    | `## 1. Hard constraints` | generated files are `internal/gen/**` (from `proto/`), `internal/db/gen/**` (from `sql/`) and `agents/skills/**`; migrations run through `make migrate-up`; "validate before claiming done" is `make check` |
    | `## 2. Tech stack` | the new stack line; one sentence: Connect v2 and routegroup differ from older material, read `docs/agents/stack-notes.md` before writing transport code |
    | `## 3. Layout` | the tree of section 1 as a table, without its demo annotations (no `notes`, no `DEMO`), with hub and mount file named; the only place the tree is written down |
    | `## 4. Code generation` | `proto/**` → `make gen-proto`; `sql/queries`, `sql/migrations` → `make sqlc`; `.claude/skills/**` → `make sync-skills`; all → `make gen` |
    | `## 5. Commands` | the targets of entry 3 |
    | `## 6. Configuration` | environment only; `.env.local` created by Make from `.env.example`; `AUTH_PROVIDER` has no default, `APP_ENVIRONMENT` defaults to `production`, `none` only on the allow-list; link to `docs/agents/env-reference.md`; a handler reads the caller with `identity.FromContext` |
    | `## 7. Go conventions` | handlers return `apperr` errors and hold no rule or query; domain packages import no transport package; the middleware-versus-interceptor rule of section 2; pgx and `pgx.BeginFunc`; integration tests through `internal/test` |
    | `## 8. The worked example` | the slice as link → path (contract, migration, queries, domain, handler, mount file, test); the example in the tree is the `whoami` slice, which has no persistence half; `go-feature` carries snippets for that half |
    | `## 9. Development workflow` | unchanged |
    | `## 10. Skill index` | the descriptions of `go-feature` and `go-middleware` updated |
    | `## 11. Review engine` | unchanged |
    | `## 12. Editing skills` | adds that `make check` fails when the mirror is stale |

#### 149. `CLAUDE.md` (Modify)

*   **Type / Scope**: Claude-specific instructions.
*   **Proposed Changes**: The title becomes `# Claude Code`. Nothing else.

#### 150. `CONTEXT.md` (Modify)

*   **Type / Scope**: Domain glossary. Init does not touch it.
*   **Proposed Changes**: Rewritten once. Domain terms: the empty stub table that
    `setup-project` detects. Infrastructure terms restated for the new stack: principal
    (`identity.FromContext`), auth provider, vertical slice (points at `AGENTS.md` §8), tracer
    bullet, spec / ticket / decision, plan, seam (`server.Deps` and consumer-side interfaces),
    generated file (`internal/gen/`, `internal/db/gen/`). The term "Note", the module map and
    the sentence about `make init` resetting the file are removed; the map is replaced by a
    link to `AGENTS.md` §3.

#### 151. `docs/production.md` (Modify)

*   **Type / Scope**: Operator doc. Ships to derived projects.
*   **Proposed Changes**: Rewritten with these sections: (1) the ships table of section 20
    with a "where" column next to "proven by"; its not-included list says "a REST variant" and
    names no tag; its validation row names no demo test and reads "validation failure: only
    once a message carries a rule"; (2) required configuration and the fail-closed rules, with a sample
    environment that sets `AUTH_PROVIDER=keycloak`, both Keycloak variables and a
    `DATABASE_URL` with `pool_max_conns`; (3) migrations: embedded; `db migrate up` as one job
    before the rollout; the session lock, and that the job must reach Postgres directly, not
    through a pooler in transaction mode; (4) image: build arguments, uid 65532, health check,
    Kubernetes probe paths; (5) listeners: the management listener has no auth and must not be
    published; (6) timeouts and limits: why there is no read or write timeout, that no handler
    is bounded by the server, the recipe of section 5, 4 MiB per message, a stream open at
    shutdown is cut at 25 s with exit code 1, the Kubernetes `preStop` `sleep`; (7) the
    database pool: the default size, `pool_max_conns`, and that readiness shares the pool; (8)
    CORS; (9) logs and metrics: the access-line fields, the histogram, alerting on
    `code="unavailable"`; (10) identity provider: the issuer rule, the readiness latch, outage
    behaviour; (11) one paragraph per "documented, not shipped" item; (12) not included. Links
    `docs/auth.md`.

#### 152. `docs/auth.md` (Create)

*   **Type / Scope**: Doc.
*   **Proposed Changes**: Seven sections: (1) how a request is authenticated, with the role
    check snippet and the meaning of 401, 503 and 403; `LOG_LEVEL=debug` shows why a token was
    rejected; (2) switching from `none` to `keycloak`, locally (`make keycloak-up`, one line in
    `.env.local`, `make -s token`, `make smoke`) and deployed; use `localhost`, not `127.0.0.1`,
    and no trailing slash; (3) production Keycloak checklist: do not import the dev realm; the
    audience mapper on every client that calls the API; no direct access grants; a short
    access-token lifespan; key rotation, and restart every API replica after deleting a signing
    key; (4) the three limits of section 7; (5) browser clients with bearer tokens; (6) service
    to service with client credentials; LDAP configured in Keycloak only; (7) not included.

#### 153. `docs/agents/request-path.md` (Create)

*   **Type / Scope**: Agent reference.
*   **Proposed Changes**: The two layers and the rule that splits them; root and interceptor
    order; the two groups; the diagram of section 2; the auth-on-the-wire table of section 7;
    what never reaches an interceptor; the `curl` forms for a unary RPC as POST and as GET; how
    to get a dev token.

#### 154. `docs/agents/stack-notes.md` (Create)

*   **Type / Scope**: Agent reference.
*   **Proposed Changes**: Library facts that differ from older versions, written as what to do:
    Connect v2 generated code, server, errors and client; routegroup rules (`Use` before routes,
    no method-qualified catch-all next to a Connect service); pgx (`pgx.ErrNoRows`, no implicit
    integer-to-text encoding); goose as a library with a `*sql.DB` from `pgx/v5/stdlib`; the
    response-writer `Flush` rule; `buf generate` deletes everything under `internal/gen`.

#### 155. `docs/middleware.md` (Delete)

*   **Type / Scope**: Doc.
*   **Proposed Changes**: Deleted; it describes the chi stack and its toggles.
    `docs/agents/request-path.md` replaces it.

#### 156. `docs/agents/middleware-api.md` (Delete)

*   **Type / Scope**: Doc.
*   **Proposed Changes**: Deleted; API reference for removed code.

#### 157. `docs/agents/README.md` (Delete)

*   **Type / Scope**: Doc.
*   **Proposed Changes**: Deleted. It is titled with the template's name, states Go 1.25, and
    tells agents to mount routes in `server.Initialize()`, which `AGENTS.md` assigns to the
    router package. `AGENTS.md` is the single source.

#### 158. `docs/glossary.md` (Delete)

*   **Type / Scope**: Empty stub.
*   **Proposed Changes**: Deleted. `CONTEXT.md` is the glossary.

#### 159. `docs/superpowers/plans/2026-07-20-skills-starter-pack.md` (Delete)

*   **Type / Scope**: Plan for the skills pack.
*   **Proposed Changes**: Deleted. It is copied into every derived project today.

#### 160. `docs/superpowers/specs/2026-07-20-skills-starter-pack-design.md` (Delete)

*   **Type / Scope**: Spec for the skills pack.
*   **Proposed Changes**: Deleted, same reason.

Skills rewritten in full or created (entries 161 to 172). Every other skill file with a stale
statement is covered by entry 173.

#### 161. `.claude/skills/go-feature/SKILL.md` (Modify)

*   **Type / Scope**: Skill, rewritten in full.
*   **Proposed Changes**: Description: "Builds business logic in this repo as a vertical slice
    from contract to storage. Use when adding or changing an RPC, a proto message, a migration,
    a sqlc query, a store or a handler, or when a schema change has to reach the API." Steps:
    1 contract (`PROTO.md`), 2 schema (`MIGRATIONS.md`), 3 queries (`SQLC.md`), 4 domain
    (`DOMAIN.md`), 5 handler and mount (`RPC.md`), 6 tests (`TESTS.md`). Worked example: the
    `whoami` files (`proto/api/v1/whoami.proto`, `internal/rpc/whoami/`,
    `internal/server/routes_whoami.go`) plus the snippets in the reference files, which use a
    made-up `tasks` domain. Finish: `make check`.

#### 162. `.claude/skills/go-feature/PROTO.md` (Create)

*   **Type / Scope**: Skill reference. Replaces `OPENAPI.md`.
*   **Proposed Changes**: One file per domain under `proto/api/v1/`, package `api.v1`, naming
    that passes `buf lint`, a protovalidate rule on every input field, `NO_SIDE_EFFECTS` for
    reads, `create_time` / `update_time`, the command `buf breaking --against
    '.git#branch=main'`. One note on paging, the only one in the pack (section 12): a list that
    needs pages conventionally takes `page_size` and `page_token` and returns
    `next_page_token`; the token format is the project's choice and nothing in the tree is
    paged. Done when `make gen-proto` puts the method on the generated handler interface.

#### 163. `.claude/skills/go-feature/DOMAIN.md` (Create)

*   **Type / Scope**: Skill reference.
*   **Proposed Changes**: A `Store` over the pool using the generated queries, with a snippet;
    owner scoping in the query; `pgx.BeginFunc` for transactions; `pgx.ErrNoRows` to an
    `apperr` not-found sentinel; reasons as `<NOUN>_<CONDITION>`; no transport import.

#### 164. `.claude/skills/go-feature/RPC.md` (Create)

*   **Type / Scope**: Skill reference. Replaces `HANDLERS.md`.
*   **Proposed Changes**: The handler maps request to store to response and holds no rule;
    principal from the context; the role check line; the mount-file rules of section 3 and the
    hub line; a service without auth is a second server on the public group; streaming: none
    ships, a stream is authenticated once, has no server-side bound and is cut at the shutdown
    grace, with a pointer to `docs/production.md`.

#### 165. `.claude/skills/go-feature/SQLC.md` (Modify)

*   **Type / Scope**: Skill reference, rewritten in full.
*   **Proposed Changes**: Named parameters; the owner predicate in every statement;
    `:execrows` for deletes; `pgx.ErrNoRows`; `pgtype` for nullable columns and the two
    overrides; no generated interface; how the store consumes the generated code; `make sqlc`
    removes the output directory first and skips when there is no query file.

#### 166. `.claude/skills/go-feature/MIGRATIONS.md` (Modify)

*   **Type / Scope**: Skill reference, rewritten in full.
*   **Proposed Changes**: `sql/migrations/NNNNN_snake_name.sql`, five digits, both goose
    sections, written by hand; `make migrate-up`, `make migrate-down`, `make migrate-status`;
    migrations are embedded, so a new file needs a rebuild; `run` refuses to start on an older
    schema.

#### 167. `.claude/skills/go-feature/TESTS.md` (Modify)

*   **Type / Scope**: Skill reference, rewritten in full.
*   **Proposed Changes**: Tests are integration tests through `internal/test` in an external
    package: `test.NewServer`, a generated client from `s.Client(s.Token(...))`, assertions
    with `connect.CodeOf`, `test.Reason` and `test.Violations`. Required cases: each error
    reason, another owner's row, a rejected input, no token. Table-driven, `t.Parallel()`.

#### 168. `.claude/skills/go-feature/HANDLERS.md` (Delete)

*   **Type / Scope**: Skill reference.
*   **Proposed Changes**: Deleted; `RPC.md` replaces it.

#### 169. `.claude/skills/go-feature/OPENAPI.md` (Delete)

*   **Type / Scope**: Skill reference.
*   **Proposed Changes**: Deleted; `PROTO.md` replaces it.

#### 170. `.claude/skills/go-middleware/SKILL.md` (Modify)

*   **Type / Scope**: Skill, rewritten in full.
*   **Proposed Changes**: Description: "Adds or changes an HTTP middleware or a Connect
    interceptor in this repo. Use when a concern has to run on every request or every RPC
    before the handler - rate limiting, security or cache headers, request-scoped logging or
    context values, error mapping - or when a ticket asks for a new middleware or interceptor."
    Step 0 chooses the layer by the rule of section 2. Middleware: a function in
    `internal/server/middleware/`, registered in the root or group `Use` list in
    `internal/server/server.go` before any route on that group; a response wrapper keeps
    `Flush` through `http.NewResponseController` and has `Unwrap`; a rejection is written
    through the Connect error writer and sets the code on the request-log carrier.
    Interceptor: `func(next connect.ServerFunc) connect.ServerFunc` in
    `internal/rpc/interceptor/`, its position in the chain. No enable flag. Test on the four
    client configurations through the harness.

#### 171. `.claude/skills/tdd/PATTERNS.md` (Modify)

*   **Type / Scope**: Skill reference, rewritten in full.
*   **Proposed Changes**: The test shapes for this stack: a store test on `test.Database`; a
    handler test through `test.NewServer` and a generated client; a middleware or interceptor
    test through the server suite's hand-registered procedures; a pure function test. No fake
    store, no `-short` gate.

#### 172. `.claude/skills/setup-project/SKILL.md` (Modify)

*   **Type / Scope**: Skill.
*   **Proposed Changes**: Step 2 becomes: `test -e scripts/template-smoke.sh`; done when the
    file is absent; present means `make init` has not run. The description of init says what it
    does now: renames the module and the project, strips the demo unless told to keep it,
    deletes `LICENSE` and the template-only files, replaces `README.md`; it does not touch
    `CONTEXT.md`. The term sources for `CONTEXT.md` are the domain packages, the proto files
    and the migrations; the module-map sentence is removed.

#### 173. Skills and one doc edited by the mapping table (Modify)

*   **Type / Scope**: 30 skill files and `docs/agents/go-style.md`.
*   **Proposed Changes**: In each file below, every statement that names an old term is
    rewritten with the new term of this table. Nothing else in these files changes, except the
    file-specific note in the list.

    | Old term | New term |
    |---|---|
    | chi, `chi.URLParam`, verb helpers | stdlib mux patterns through routegroup; `r.PathValue`; fields of the RPC request |
    | `oapi/openapi.yaml`, the OpenAPI spec, `make gen-oapi`, `internal/api/*.gen.go` | `proto/api/v1/<domain>.proto`, `make gen-proto`, `internal/gen/api/v1/` |
    | endpoint, `/v1/<resource>`, an HTTP status returned by a handler | RPC, `/api.v1.<Service>/<Method>`, a Connect code with an `ErrorInfo` reason |
    | `internal/api/handlers/<domain>.go` | `internal/rpc/<domain>/` (handler) and `internal/domain/<domain>/` (store and rules) |
    | `internal/api/router/routes_<domain>.go`, the route hub | `internal/server/routes_<domain>.go` (mount file), `internal/server/routes.go` (hub) |
    | `errs.Write(w, status, err)`, `RenderJSON`, `internal/server/errs` | return `apperr.New(kind, reason, message)`; `internal/apperr`; the `Errors` interceptor maps it |
    | `auth.PrincipalFromContext`, `auth.TokenFromContext`, `internal/server/auth`, the `Require*` guards | `identity.FromContext`, `internal/identity`, `HasRole` in the handler; there is no token accessor |
    | `pkg/keycloak`, `pkg/db`, `pkg/dotenv`, `pkg/tests`, "`pkg/` is a promise" | `internal/identity/keycloak`; there is no top-level `pkg` directory |
    | lib/pq, `database/sql`, `*sql.DB`, `sql.Tx`, `sql.ErrNoRows`, `sql.Null*` | pgx/v5, `*pgxpool.Pool`, `pgx.Tx`, `pgx.ErrNoRows`, `pgtype.*` |
    | `gen.Querier`, a fake querier, `httptest` with a fake store or against the router, `httptest.NewRecorder` in a handler test, `internal/server/test` | no generated interface; an integration test through `internal/test` on Postgres, with a generated client against `test.NewServer` |
    | `server.NewWithConfig`, `Initialize`, `InitDB` | `server.Handler(Deps)`; `cmd/run.go` fills `Deps` |
    | `env.GetEnv*` | one field per variable in `internal/server/config/config.go` |
    | `testing.Short()`, `go test -short`, a no-Docker subset | removed; Docker is a prerequisite of the tests |
    | `migrations/NNNN_name.sql` | `sql/migrations/NNNNN_name.sql` |
    | `owner_id`, `ListNotesByOwner`, positional `$1` | `owner`, named parameters such as `@owner` |
    | `make db-up`, `make db-down`, `db seed`, `db migrate reset` | `make dev-up`, `make dev-down`; the other two are removed |
    | `make test` and `make lint`, in any role | `make check` where the text means the gate or a command that must pass; `go test -race ./<package>/...` for a focused run; "the test step of `make check`" or "the lint step of `make check`" where it means that one tool |
    | `go-pkgz/rest`, `go-chi/cors`, `cors.AllowAll()`, the `SERVER_*` variables | `internal/server/middleware`; `CORS_ALLOWED_ORIGINS`; no enable flags |
    | `DEMO`, the notes slice as the worked example, any demo path | the `whoami` files; an example that needs persistence uses a made-up `tasks` domain |
    | the template's name or module path | no name; `<module>` or `$(basename "$PWD")` |
    | `docs/middleware.md`, `docs/agents/middleware-api.md`, `docs/agents/README.md`, `docs/glossary.md` | `docs/agents/request-path.md`, `AGENTS.md`, `CONTEXT.md` |
    | `HANDLERS.md`, `OPENAPI.md` | `RPC.md`, `PROTO.md` |

    The words `handlers`, `routes_`, `cors` and `migrate-up` survive with their new meaning and
    are not banned. Files:

    *   `.claude/skills/code-review/SKILL.md`
    *   `.claude/skills/code-review/reviewers/go-documentation.md`: "a new or changed RPC";
        proto comments are the API documentation.
    *   `.claude/skills/code-review/reviewers/go-implementation.md`: "every link of the slice"
        lists the new links.
    *   `.claude/skills/code-review/reviewers/go-quality.md`: not-found mapping in the store;
        every query scoped by owner; every service on the authed server unless public on
        purpose; no transport import in domain packages.
    *   `.claude/skills/code-review/reviewers/go-simplification.md`: handler to store is the
        whole stack.
    *   `.claude/skills/code-review/reviewers/go-smells.md`: a handler without rules; one mount
        file per domain; limits live in validation rules.
    *   `.claude/skills/code-review/reviewers/go-testing.md`: no `-short` claim; cases by error
        reason.
    *   `.claude/skills/codebase-design/SKILL.md`: seams are `server.Deps` and consumer-side
        interfaces.
    *   `.claude/skills/codebase-design/DEEPENING.md`
    *   `.claude/skills/codebase-design/DESIGN-IT-TWICE.md`
    *   `.claude/skills/diagnosing-bugs/SKILL.md`: the ladder is a focused `go test`, a harness
        test, a request against `make run` (form in `docs/agents/request-path.md`); the logging
        variable is `LOG_LEVEL`.
    *   `.claude/skills/domain-modeling/SKILL.md`
    *   `.claude/skills/domain-modeling/CONTEXT-FORMAT.md`: a neutral example row; the
        instruction to mark demo terms is removed.
    *   `.claude/skills/domain-modeling/ADR-FORMAT.md`
    *   `.claude/skills/grilling/SKILL.md`
    *   `.claude/skills/handoff/SKILL.md`: the file name uses `$(basename "$PWD")`.
    *   `.claude/skills/implement/SKILL.md`
    *   `.claude/skills/research/SKILL.md`: the example question names a current library; the
        comment line that shows a module-cache path is dropped; the toolchain examples name no
        goose CLI.
    *   `.claude/skills/resolving-merge-conflicts/SKILL.md`: generated code is regenerated with
        `make gen`, never merged by hand.
    *   `.claude/skills/tdd/SKILL.md`: tests go through the harness; no fake-store pattern and
        no `-short` gate.
    *   `.claude/skills/to-plan/SKILL.md`: the example plan takes each path from `AGENTS.md`
        §8; its validation command is `make check`.
    *   `.claude/skills/to-plan/TASK-SHAPES.md`
    *   `.claude/skills/to-spec/SKILL.md`: the testing sentence names the harness and a
        generated client.
    *   `.claude/skills/to-tickets/SKILL.md`: the example ticket is an RPC with a reason per
        failure.
    *   `.claude/skills/to-tickets/WIDE-REFACTOR.md`
    *   `.claude/skills/triage/SKILL.md`: the closing criterion and the close comment say
        `make check`.
    *   `.claude/skills/writing-for-agents/SKILL.md`
    *   `.claude/skills/writing-for-agents/LANGUAGE.md`: the demo row is removed; rows for hub,
        mount file and harness are added.
    *   `.claude/skills/writing-for-agents/SKILL-MECHANICS.md`: the quoted `go-feature`
        description equals the one in entry 161; the angle-bracket example is a proto path,
        not a `/v1` path.
    *   `.claude/skills/writing-style/SKILL.md`: the quoted commit example names no removed
        file.
    *   `docs/agents/go-style.md`: handler and store wording; the transaction helper is
        `pgx.BeginFunc`; tests go through the harness, with no fake querier and no `-short`
        guard.

    Acceptance greps, run from the repository root. Both must print nothing and exit 1:

    ```text
    git grep -nIE \
      -e '(^|[^A-Za-z])chi([^A-Za-z]|$)' \
      -e 'oapi|[Oo]pen[Aa][Pp][Ii]' \
      -e 'lib/pq|database/sql|go-pkgz/rest|go-chi' \
      -e 'errs\.Write|RenderJSON|URLParam|Querier|PrincipalFromContext|TokenFromContext' \
      -e 'NewWithConfig|InitDB|GetEnv|testing\.Short|test -short' \
      -e 'sql\.(Tx|ErrNoRows|Null)' \
      -e 'internal/api|internal/server/(auth|errs|test)([^_A-Za-z]|$)|(^|[^A-Za-z.])pkg/' \
      -e '(^|[^/A-Za-z])migrations/|(^|[^A-Za-z0-9_])/v1/' \
      -e 'db-up|db-down|db seed|owner_id|SERVER_[A-Z]' \
      -e 'DEMO|go-vibe-starter|Go Vibe Starter|driif' \
      -e 'NotesService|TestNotes|notes\.(proto|sql)|routes_notes|internal/(domain|rpc)/notes' \
      -- .claude/skills .claude/agents docs AGENTS.md CLAUDE.md CONTEXT.md ':!docs/designs'

    git grep -nIE -e 'make (test|lint)' -- .claude/skills .claude/agents
    ```

    Three patterns are anchored so that new paths pass. `internal/server/(auth|errs|test)` must
    end there, so the test files `internal/server/auth_test.go` and
    `internal/server/testroutes_test.go` may be named. `/v1/` must not follow a path
    character, so `proto/api/v1/` and `internal/gen/api/v1/` pass. `pkg/` must not follow a
    letter or a dot; a module-cache path (`.../go/pkg/mod/...`) still fails, which is why the
    `research` skill drops its path comment. The second grep enforces the single done criterion: no skill
    names `make test` or `make lint` in any role (see the mapping row). Mapped but in no grep,
    because the word stays correct elsewhere, and therefore checked by reading the files of
    the list: `httptest` (the harness is built on it), `*sql.DB`
    (`docs/agents/stack-notes.md` names it for goose), `Initialize`, `endpoint` and the HTTP
    status numbers. `README.md` is outside the first grep in the template tree because it
    names the old stack and the tag; `docs/designs` is outside it because this document does.

#### 174. `agents/skills/**` (Modify, generated)

*   **Type / Scope**: Mirror of `.claude/skills`, 44 files today.
*   **Proposed Changes**: Regenerated by `make sync-skills`, never edited by hand. After stage 2
    it has the three new `go-feature` files and lacks the two deleted ones.

**Stage 3 — template mechanics.** Gate: the `stripped` and `keep-demo` commands of section 18
pass locally, run one after the other and after `make dev-down` in the template tree (the
stage 1 checks by hand leave the `app` Postgres on port 5434). The stage ends with the
definition of done (Goals).

#### 175. `scripts/init.sh` (Modify)

*   **Type / Scope**: Init script, kept in derived projects. Rewritten.
*   **Proposed Changes**: `#!/usr/bin/env bash`, `set -euo pipefail`, bash 3.2 (no `mapfile`,
    no associative arrays, no `${v,,}`); no `sed -i`, `grep -P`, `xargs -r`, `readlink -f`;
    searches go through `git grep` and `git ls-files -z`; no python.

    ```text
    usage: scripts/init.sh [--module PATH] [--name NAME] [--keep-demo] [-y|--yes] [-h|--help]

      --module PATH   Go module path. Required on the first run. Default on later runs: the current path.
      --name NAME     project name: binary bin/NAME, compose project, image name.
                      Default: last element of the module path, lower-cased
                      (the element before it when the last is a major version such as v2).
      --keep-demo     keep the demo slice. First run only; ignored later.
      -y, --yes       never prompt, take flags and defaults.
    exit codes: 0 done, 1 a step or the residue check failed, 2 usage or validation error
    ```

    Steps as in section 14. Literals:
    *   Modes: stdin a terminal and no `--yes` → prompts for module path, name, "Keep the demo
        slice? [y/N]", then prints the plan and asks "Proceed? [y/N]". No terminal and no
        `--yes` → exit 2 with `init: not a terminal; pass --module, --name and --yes`. `--yes`
        without `--module` on a first run → exit 2.
    *   Validation: module path against
        `^[a-z0-9][a-z0-9.-]*\.[a-z][a-z0-9-]*(/[A-Za-z0-9_~-][A-Za-z0-9._~-]*)+$`, then
        `go mod edit -module` on a temporary copy of `go.mod`; name against
        `^[a-z0-9]+([_-][a-z0-9]+)*$`, at most 50 characters.
    *   Preflight (exit 1): a git work tree whose root has `go.mod`, `Makefile`,
        `docker-compose.yaml`; `git`, `go`, `make` on `PATH`; `make tools-check`.
    *   State: old module path from `go.mod`; old name from the `APP_NAME` line; first run =
        `scripts/template-smoke.sh` exists; demo present = some file outside `internal/gen`
        and `internal/db/gen` has a first line that matches `^(//|--) DEMO: `.
    *   First run: `rm -f .github/workflows/template-smoke.yml SECURITY.md LICENSE`;
        `rm -rf docs/designs`; delete `docs/plans/**/*.md` and `docs/adr/*.md` (the `.gitkeep`
        files stay); write the `README.md` stub: title `# <name>`; quickstart (`make tools`,
        `make dev-up`, `make migrate-up`, `make run`, the two `curl` calls of CUJ 2, `make
        check`); links to `AGENTS.md`, `CONTEXT.md`, `docs/production.md`, `docs/auth.md`; "run
        `/setup-project` in your agent". Strip unless `--keep-demo`: delete every file whose
        first line matches the marker; delete every line that ends with `// DEMO:line`;
        `find internal proto -type d -empty -delete`. With `--keep-demo` print: `demo kept:
        init will not remove it later. Removal is manual, and its migration is version 1 of
        every database it was applied to.` Last: `rm -f scripts/template-smoke.sh`.
    *   Module path: candidates from `git grep -I --untracked -l -F -e "$OLD" -- .
        ':!internal/gen' ':!internal/db/gen'`; each rewritten through a temporary file,
        replacing the old path where it is followed by a character outside `[A-Za-z0-9_~-]`
        or by the end of the line; `go mod edit -module` last.
    *   Name: `Makefile` line `^APP_NAME := .*$`; `docker-compose.yaml` line `^name: .*$`.
    *   Residue check: `git grep -nIF --untracked` over all files except `scripts/init.sh`,
        for: the old module path, when it changed; and, when no demo-marked file is left, the
        string `DEMO:` and the path of every file the strip deleted in this run. The bare
        owner name of the template is checked by entry 176 step 4, not here.
    *   Output on success (the `LICENSE` line on a first run only):

        ```text
        Initialised <name> (<module>).
        Removed LICENSE: add your own.
        Next:
          1. git add -A && git commit -m "Initialise project"
          2. make dev-up        Postgres on 127.0.0.1:5434
          3. make migrate-up
          4. make run           127.0.0.1:8080, AUTH_PROVIDER=none from .env.local
          5. make check         the gate CI runs
          6. make labels        once the repository exists on GitHub (needs gh)
        Then run /setup-project in your agent.
        ```

#### 176. `scripts/template-smoke.sh` (Create)

*   **Type / Scope**: Bash, template-only, deleted by init. Same portability rules as entry
    175.
*   **Proposed Changes**: `scripts/template-smoke.sh --module M --name N [--keep-demo]
    [--no-docker]`, run from the template tree; it tests the index, so stage what should be
    tested. Without `--no-docker` it needs host port 5434 free: run `make dev-down` in the
    template tree first, and run the legs one after the other. `N` equal to the template's own
    project name (`app`) is a usage error with exit 2, so the teardown of step 7 can never
    remove the template tree's dev volume. Steps of section 18 with these literals:
    1.  `DEST=$(mktemp -d)`; `git checkout-index --all --prefix="$DEST/"`; in `DEST`:
        `git init -q -b main`, add, commit. Every commit the script makes is `git -c
        user.name=template-smoke -c user.email=template-smoke@example.invalid -c
        commit.gpgsign=false commit -q -m <message>`, so it depends on no git configuration of
        the machine.
    2.  `make tools` in `DEST`.
    3.  `"$BASH" scripts/init.sh --module M --name N --yes [--keep-demo] </dev/null`; commit.
    4.  In `DEST`, each of these prints nothing: `git grep -nIiF -e
        'github.com/driif/go-vibe-starter' -e 'driif'`; without `--keep-demo`, `git grep -nIF
        -e 'DEMO:' -e 'internal/domain/notes' -e 'internal/rpc/notes' -e 'NotesService' -e
        'notes.proto' -e 'notes.sql' -e 'routes_notes' -e 'TestNotes' -- .
        ':!scripts/init.sh'`; the two acceptance greps of entry 173, the first with
        `README.md` added to its paths. These
        paths do not exist: `.github/workflows/template-smoke.yml`,
        `scripts/template-smoke.sh`, `SECURITY.md`, `LICENSE`, `docs/designs`; without
        `--keep-demo` also `proto/api/v1/notes.proto`, `internal/domain`,
        `internal/rpc/notes`, `internal/db/gen`.
    5.  `make check`, then `git status --porcelain` is empty. With `--no-docker`: `make gen`,
        `go mod tidy`, `go build ./...`, `go vet ./...`, `make lint`, then the same clean-tree
        test.
    6.  Init again with the same arguments; `git status --porcelain` is empty.
    7.  `make image-smoke`, unless `--no-docker`. Just before it the script sets a flag; an
        `EXIT` trap runs `docker compose down -v` in `DEST` when the flag is set, on success
        and on failure.

    Each step prints its number and name before it runs; the script exits 1 at the first
    failing step. The trap removes `DEST` after the teardown when every step passed, and keeps
    it otherwise.

#### 177. `.github/workflows/template-smoke.yml` (Create)

*   **Type / Scope**: CI workflow, template-only, deleted by init.
*   **Proposed Changes**: Job `init` with `if: github.repository == 'driif/go-vibe-starter'`
    and a matrix of the three legs of section 18 (`stripped` and `keep-demo` on
    `ubuntu-latest`, `macos` on `macos-latest`). Steps: checkout, setup-go with
    `go-version-file: go.mod`, the leg's command. Triggers, permissions, concurrency and action
    pins as in entry 178.

#### 178. `.github/workflows/ci.yml` (Modify)

*   **Type / Scope**: CI workflow, ships to derived projects.
*   **Proposed Changes**: Rewritten. Triggers: `pull_request`, `push` to `main`, `schedule`
    with cron `17 5 * * 1`, `workflow_dispatch`. `permissions: contents: read`.
    `concurrency: {group: ${{ github.workflow }}-${{ github.ref }}, cancel-in-progress: true}`.
    Actions pinned by commit SHA with the tag in a comment, as recorded on 2026-11-20:
    `actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1` with
    `persist-credentials: false`, and
    `actions/setup-go@d8cbb8045d7f43706b9177cceb36c1f91a6644d5 # v7.1.0` with
    `go-version-file: go.mod`. Job `check` on `ubuntu-latest`: checkout, setup-go, `make
    tools`, `make check`, `make vuln`. Job `image` on `ubuntu-latest`: checkout, `make
    image-smoke`. No tool pin and no Go version in the file. Removed: the `env` block of pins,
    the `golangci-lint-action` step, the `codegen-drift` job.

#### 179. `.github/workflows/keycloak.yml` (Create)

*   **Type / Scope**: CI workflow, ships to derived projects.
*   **Proposed Changes**: Job `smoke` on `ubuntu-latest`: checkout, setup-go, `make smoke`; on
    failure `docker compose logs keycloak`. `pull_request` and `push` to `main` are filtered to
    the paths `keycloak/**`, `docker-compose.yaml`, `internal/identity/**`,
    `internal/server/middleware/auth*.go`, `internal/test/**` and
    `.github/workflows/keycloak.yml`; the weekly schedule and `workflow_dispatch` are not
    filtered. Permissions, concurrency and action pins as in entry 178.

#### 180. `.github/dependabot.yml` (Modify)

*   **Type / Scope**: Dependabot config.
*   **Proposed Changes**: Four ecosystems on `/`, each weekly with one group that matches
    `*`: `gomod` (group `go-dependencies`, limit 5, as today), `github-actions` (group
    `github-actions`, as today), `docker` (new group `docker-images`), `docker-compose` (new
    ecosystem, group `compose-images`).

#### 181. `.gitattributes` (Create)

*   **Type / Scope**: Git attributes.
*   **Proposed Changes**: `linguist-generated=true` for `internal/gen/**`, `internal/db/gen/**`
    and `agents/skills/**`, so review diffs collapse them.

#### 182. `SECURITY.md` (Create)

*   **Type / Scope**: Security policy, template-only, deleted by init.
*   **Proposed Changes**: About ten lines: report through GitHub private vulnerability
    reporting; the dev compose credentials and the dev realm are public on purpose.

#### 183. `.gitignore` (Modify)

*   **Type / Scope**: Git ignore rules.
*   **Proposed Changes**: Removes `.pkg`, `tmp`, `assets/mnt/**`, `!assets/mnt/.gitkeep`,
    `dumps` and `*.out`. `docker-data/` stays ignored, because existing clones still hold that
    directory from the old compose file. `bin`, `.env.local`, `.env`, `profile.cov` and the
    `.ralphex` lines stay.

#### 184. `.github/ISSUE_TEMPLATE/ticket.yml` (Modify)

*   **Type / Scope**: Issue form.
*   **Proposed Changes**: The default value of the validation field is `make check` in place
    of `make test` and `make lint`.

#### 185. `.github/PULL_REQUEST_TEMPLATE.md` (Modify)

*   **Type / Scope**: Pull request template.
*   **Proposed Changes**: The example comment under "How verified" names `make check`.

#### 186. `.ralphex/config` (Modify)

*   **Type / Scope**: Config of ralphex, the tool that runs plan files unattended. Comment only.
*   **Proposed Changes**: The comment names `make check` as the validation command in place of
    `make test` and `make lint`.

**Unchanged.** Tracked files the design keeps as they are; they have no entry:

*   `LICENSE` (deleted by init in derived projects)
*   `scripts/setup-labels.sh`
*   `.ralphex/.gitignore`
*   `.github/ISSUE_TEMPLATE/bug.yml`, `.github/ISSUE_TEMPLATE/config.yml`,
    `.github/ISSUE_TEMPLATE/decision.yml`, `.github/ISSUE_TEMPLATE/spec.yml`
*   `docs/adr/.gitkeep`, `docs/plans/.gitkeep`, `docs/plans/completed/.gitkeep`,
    `docs/plans/maps/.gitkeep`
*   `sql/queries/.gitkeep`
*   `.claude/agents/go-documentation.md`, `.claude/agents/go-implementation.md`,
    `.claude/agents/go-quality.md`, `.claude/agents/go-simplification.md`,
    `.claude/agents/go-smells.md`, `.claude/agents/go-testing.md`
*   `.claude/skills/grill-me/SKILL.md`, `.claude/skills/grill-with-docs/SKILL.md`,
    `.claude/skills/improve-architecture/SKILL.md`, `.claude/skills/vibe/SKILL.md`,
    `.claude/skills/wayfinder/SKILL.md`

### Verification Plan

*   **Compilation/Build**: `make tools && make build`; `go build ./... && go vet ./...`;
    `make image VERSION=test`.
*   **Unit Tests**: The template has no separate unit tier. `make test` (`go test -race
    ./...`) runs the integration tests and needs Docker for the Postgres container. Focused
    runs: `go test -race ./internal/server/... ./internal/rpc/... ./internal/reqlog/... ./cmd/...`,
    `go test -race ./internal/identity/...`,
    `go test -race ./internal/db/... ./internal/test/... ./internal/domain/...`.
*   **Static Analysis & Linting**: `make lint` (`golangci-lint run`, `buf lint`,
    `buf format -d --exit-code`); `make vuln`; `scripts/sync-skills.sh --check`; the two
    acceptance greps of entry 173.
*   **Test Coverage Requirement**: No percentage. The named tests in the entries are the
    requirement; `make cover` prints the summary. Every row of the auth-on-the-wire table, of
    the `Errors` mapping table and of the kind table, and every `Validate` rule of section 8,
    has a test case.
*   **E2E/Integration**:
    *   The gate: `make tools && make check` (drift, `make lint`, `make test`, mirror check).
    *   The three template-smoke legs, with the commands of section 18.
    *   Image: `make image-smoke`.
    *   Keycloak: `make smoke` (starts Keycloak 26.8.2 with the shipped realm and runs
        `TestSmoke_WhoAmI`); `make -s token | cut -c1-20` prints the start of a JWT.
    *   Quickstart by hand before merge: the commands of CUJ 2.
    *   Fail-closed by hand, from a directory with no `.env.local`: `env -i bin/app run` exits
        1 naming `AUTH_PROVIDER is required` and `DATABASE_URL is required`;
        `env -i AUTH_PROVIDER=none DATABASE_URL=postgres://x bin/app run` exits 1 naming the
        allow-list; `go list -deps . | grep -c internal/identity/testprovider` prints 0.
    *   Negative checks, each run once when the scripts are written: `scripts/init.sh --yes`
        without `--module` on a first run, `--module foo`, `--name My_App`, no terminal
        without `--yes` → exit 2 and a clean tree; one tool stamp removed → `make init` and
        `make gen` stop with `tools missing or outdated: run 'make tools'`; the string `DEMO:`
        left in a doc → init exits 1 at the residue step with `file:line`; a skill edited
        without syncing → `scripts/sync-skills.sh --check` exits 1; a query file changed
        without `make gen` → `make check` fails at the drift step; init killed during
        regeneration and rerun → completes.
    *   Acceptance: the definition of done in Goals. Then `git push origin rest-chi-openapi`
        and `gh repo edit --template`.

--------------------------------------------------------------------------------

<!-- Provenance: Indicates initial structural drafting via EGM. -->

*Drafted with EGM*
