# Go backend template

A Go-only GitHub template for Connect-RPC backends with PostgreSQL and optional Keycloak.
`make init` renames the module/project, removes the reference slice unless kept, regenerates code,
and leaves a working WhoAmI service and integration-test harness.

## Quickstart

Prerequisites: Go 1.26 or newer (the toolchain directive selects 1.27.x), Docker with Compose,
GNU Make, git, curl and a C compiler for sqlc. GitHub CLI is needed only for labels and repository work.
Choose **Use this template** on GitHub, clone your repository, then:

```sh
make tools
scripts/init.sh --module github.com/your-org/your-project --name your-project --yes
make dev-up
make migrate-up
make run
```

In another terminal:

```sh
curl -fsS http://localhost:8080/readyz
curl -fsS -H 'Content-Type: application/json' -d '{}' http://localhost:8080/api.v1.WhoAmIService/WhoAmI
make check
```

`make init` without flags prompts interactively. Pass `--keep-demo` on the first run to retain the
persistence example. `make run` creates `.env.local` from `.env.example`; it explicitly enables local
auth with the admin role and binds the API and management listeners to loopback. Postgres listens on 127.0.0.1:5434. The API listens on port 8080. Frontend work starts separately
in `frontend/`, with your choice of stack.

To try the template before creating a project:

```sh
make tools
make dev-up
make migrate-up
make run
```

Run `make dev-down` before initializing if you tried it first: initialization changes the compose
project name, while the old stack would still hold port 5434.

## What you get

| Included | Evidence |
|---|---|
| Connect, gRPC, gRPC-Web and JSON POST/safe GET on HTTP/1.1 and h2c | protocol integration tests |
| stdlib ServeMux with routegroup and thin RPC handlers | production stack used by test harness |
| pgx pool, sqlc, embedded goose migrations and startup schema check | real Postgres migration/store tests |
| Fail-closed Keycloak access-token verification and explicit local auth | verifier, auth and real-Keycloak smoke tests |
| Input validation, safe error mapping and panic recovery | interceptor and RPC tests |
| Request ids, JSON access logs, CORS allow-list and message limits | HTTP integration tests |
| Readiness/liveness and graceful shutdown | health and lifecycle tests |
| Metrics and pprof on an opt-in private listener | management tests |
| Static non-root image with version and health probe | `make image-smoke` |
| Drift/lint/race-test/mirror checks, advisory scan and dependency updates | CI workflows |

Deployment still owns TLS, secrets, backups, reverse-proxy limits, rate limiting, metrics storage,
tracing, release publishing and manifests. See [production guidance](docs/production.md) for the
boundaries and streaming timeout recipe. No frontend, browser e2e suite, htmx template, tenancy,
Keycloak admin client or LDAP bind is included. Directory federation belongs in Keycloak.

## Call without generated clients

Every unary RPC accepts `POST /<package>.<Service>/<Method>` with a JSON body and
`Content-Type: application/json`. Safe methods also support Connect GET. Errors contain stable
`code` and `message` fields, with structured details for domain reasons and validation violations.
Generated clients provide type safety and Connect/gRPC/gRPC-Web protocol selection.
The previous chi/OpenAPI implementation is preserved at `rest-chi-openapi`; it is not maintained.

## Authentication

`AUTH_PROVIDER` is required. `none` is accepted only in development/dev/local/test and supplies a
`local-dev` principal with no default roles. `keycloak` requires issuer and audience. To try it:

```sh
make keycloak-up
# Set AUTH_PROVIDER=keycloak in .env.local, then restart make run.
make -s token
make smoke
```

The dev realm has alice and bob, an audience mapper, a PKCE browser client and a service client.
Its password grant is for local tests only. [Auth guidance](docs/auth.md) covers production clients,
LDAP federation, browser flows, key rotation and verifier limits.

## Build a feature

Read [AGENTS.md §8](AGENTS.md#8-the-worked-example) and use the `go-feature` skill for the vertical
slice. `make help` lists commands; `make check` is the gate before a change is considered complete.

## Working with agents

The canonical skills live under `.claude/skills/`, mirrored by `make sync-skills` to `agents/skills/`.
Read [AGENTS.md](AGENTS.md) first. There are 24 skills: `/vibe` is the map, `/setup-project` prepares
the project vocabulary and issue labels, and the normal feature flow is:

```text
/grill-me → /to-spec → /to-tickets → /implement
                               ↘ /to-plan + ralphex
```

GitHub issues hold intent; disposable files under `docs/plans/` hold execution. The Go feature,
middleware, testing, research and code-review skills provide the implementation disciplines.

## After Use this template

GitHub copies the files, not labels, secrets, settings or branch protection. CI runs on your first
push; Dependabot configuration also comes along. Run `/setup-project` to prepare labels and domain
language, add your own licence, then set repository protections for your team. `gonew` is not a
supported initialization path. Two local projects need distinct host ports in compose and matching
environment values. Derived projects have no automatic update mechanism from this template.

## Maintain the template

The GitHub template setting and historic tag are maintained with:

```sh
git push origin rest-chi-openapi
gh repo edit --template
```

Stage the intended file set before smoke-testing it. Stop the template dev stack first so port 5434
is free; run the Linux legs one at a time:

```sh
make dev-down
scripts/template-smoke.sh --module example.com/acme/smoke-svc --name smoke-svc
scripts/template-smoke.sh --module git.example.org/platform/teams/billing-service --name billing-service --keep-demo
/bin/bash scripts/template-smoke.sh --module example.com/acme/smoke-svc --name smoke-svc --no-docker
```

Bump the four Makefile tool pins, Go toolchain directive, Docker Go base image and test Postgres
image together as needed. Dependabot updates Go modules, actions and image declarations. GitHub may
disable scheduled workflows after 60 days without repository activity; re-enable them when resuming.

## Licence

The template is MIT. The author asks nothing of projects generated from it. Initialization removes
`LICENSE` so a new project can choose its own licence and attribution.
