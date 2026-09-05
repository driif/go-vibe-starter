# Task shapes

How to cut `### Task N:` sections for work that is not a straight feature slice. The format rules in
[`SKILL.md`](SKILL.md) hold for every shape here; only the cutting changes.

## Feature slice (the default)

The slice runs migration → sqlc query → OpenAPI → handler → route → test. Cut it in two or three
tasks, not six: the data layer (migration + queries + `make sqlc` + query tests), then the HTTP
layer (OpenAPI + `make gen-oapi` + handler + route + handler tests), then verification. A **tracer
bullet** ticket should not need more, and each task leaves `make test` green on its own.

When a slice touches the router, the last checkbox of the HTTP task registers the domain in the hub
(`internal/api/router/routes.go`), so the endpoint is reachable the moment the task closes.

## Wide refactor

A mechanical change whose blast radius fans across the tree — renaming a column, retyping a shared
symbol, moving a package. No vertical slice can land green, so it runs **expand–contract**, which
`/to-tickets` defines and cuts into tickets. One task per phase, in the same order: expand, then one
migrate task per batch, then contract.

Each migrate task's test checkboxes are "update the tests in `<package>` to the new form" rather
than new tests. Size a batch at one package or one directory, and name that package in the task
title, so the loop cannot drift into another batch.

## Bug fix

Cut it **red-green**. Task 1 reproduces: add the failing test that pins the defect, and say in the
checkbox that it is expected to fail at this point. Task 2 fixes the code and turns that test green.
Task 3 covers the neighbours the same defect could reach. Splitting reproduction from fix keeps the
loop from "fixing" a bug it never demonstrated.

## Hardening pass

Config validation, error paths, timeouts, a lint or CI tightening. These have no natural slice, so
cut by **seam**: one task per boundary that changes behaviour (`AUTH_PROVIDER` handling, readiness
check, the middleware chain). Each task's tests assert the new behaviour at that seam and the
unchanged behaviour beside it, which is what stops a hardening pass from silently narrowing the API.

## Generated code

Any task that edits `oapi/openapi.yaml`, `sql/queries/` or `migrations/` owns the regeneration in
the same task — `make gen-oapi`, `make sqlc`, or `make gen` — and commits the generated output.
The verification task ends with `make gen` plus `git diff --exit-code` to catch **drift**. Generated
files are never hand-edited, so a checkbox never asks for an edit inside `*.gen.go` or
`internal/db/gen`.
