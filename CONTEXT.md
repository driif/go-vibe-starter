# Context

The domain vocabulary belongs here. The package map is in [AGENTS.md §3](AGENTS.md#3-layout).

## Ubiquitous language

| Term | Means |
|---|---|
| | |

## Infrastructure terms

| Term | Means |
|---|---|
| principal | authenticated caller read through `identity.FromContext` |
| auth provider | `keycloak` or explicitly permitted local `none` |
| vertical slice | contract through persistence and transport to tests; see [AGENTS.md §8](AGENTS.md#8-the-worked-example) |
| tracer bullet | one thin end-to-end slice |
| spec / ticket / decision | GitHub issues holding intent, one slice, or an open question |
| plan | disposable execution checklist in `docs/plans/` |
| seam | injected dependencies in `server.Deps` or a consumer-side interface |
| generated file | output in `internal/gen/` or `internal/db/gen/`; change its source |
