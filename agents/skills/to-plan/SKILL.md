---
name: to-plan
disable-model-invocation: true
description: Turn ticket issues into a ralphex execution plan file under docs/plans/.
---

Ticket issues carry the intent. `docs/plans/YYYYMMDD-<slug>.md` is what the `ralphex` binary
executes. This skill is the bridge between the two, and its output is **parsed** rather than read:
the headings, the `Task` keyword and the checkbox placement below are structural tokens. Reproduce
them exactly.

## 1. Check the executor

Run `which ralphex`. If it is missing, say so and give the install line —
`brew install umputun/apps/ralphex` on macOS, or
`go install github.com/umputun/ralphex/cmd/ralphex@latest` anywhere with Go. Write the plan either
way; the file stands on its own until the binary lands.

## 2. Read the tickets

```bash
gh issue list --label ticket --label ready
gh issue view <N> --json number,title,body,labels --jq '.title, .body'
```

Take from each ticket: what the slice delivers, the acceptance criteria, the `Blocked by #N` edges,
and the validation commands. Plan the **frontier** — tickets whose blockers are all closed. A ticket
still blocked by an open issue stays out of this plan and gets its own later.

One plan covers one coherent piece of work: a single **tracer bullet**, or a short chain of them
whose blocking edges run in a straight line.

## 3. Read the code the slice touches

Open the files the tickets name and the nearest existing example of the same shape — the `notes`
`DEMO` slice for a database-plus-API feature, `internal/api/handlers/health.go` for a bare endpoint.
Plans that name real paths, real packages and the existing pattern to copy execute cleanly; plans
that describe the work abstractly stall on the first task. `AGENTS.md` holds the conventions.

## 4. Cut the tasks

One task is one logical unit — one migration, one endpoint, one middleware — around five checkboxes,
each naming a file. Order tasks so each leaves the tree green, because ralphex validates after every
one. For shapes other than a straight feature slice (wide refactor, bug fix, hardening pass), read
[`TASK-SHAPES.md`](TASK-SHAPES.md).

Present the task list as prose and get approval before writing the file. Cheap to re-cut now,
expensive once the loop is running.

## 5. Write `docs/plans/YYYYMMDD-<slug>.md`

Date is today, slug is kebab-case from the work. Example, for tickets #41 and #42:

````markdown
# Projects API

## Overview
Adds a `projects` domain: create, list and delete a project scoped to the calling principal.
From #41 (create + list) and #42 (delete). Walks the same slice as the existing handlers, and
adds no new dependency.

## Context
- Issues: #41, #42
- Files involved: `migrations/`, `sql/queries/projects.sql`, `oapi/openapi.yaml`,
  `internal/api/handlers/projects.go`, `internal/api/router/routes_projects.go`
- Patterns to follow: the `notes` DEMO slice for shape, `errs.Write` for error responses, the
  `principal` from the request context for `owner_id`, helpers in `internal/server/test`

## Validation Commands
- Test: `make test`
- Lint: `make lint`

## Implementation Steps

### Task 1: Add the projects schema and queries
- [ ] add `migrations/0002_projects.sql` with goose Up/Down and an index on `owner_id`
- [ ] add `sql/queries/projects.sql` with `CreateProject :one`, `ListProjectsByOwner :many`,
      `DeleteProject :exec`, all parameterized
- [ ] run `make sqlc` and commit the regenerated `internal/db/gen` output
- [ ] write tests for `CreateProject` and `ListProjectsByOwner` using `internal/server/test` helpers
- [ ] write tests for `DeleteProject` including the not-found case
- [ ] run `make test` and `make lint` — must pass before Task 2

### Task 2: Expose the projects endpoints
- [ ] add `/v1/projects` (GET, POST) and `/v1/projects/{id}` (DELETE) with schemas to
      `oapi/openapi.yaml`, then run `make gen-oapi`
- [ ] add `internal/api/handlers/projects.go`, scoping every query to the principal subject
- [ ] add `internal/api/router/routes_projects.go` with `registerProjects(s)`, called from the hub
- [ ] write table-driven tests in `internal/api/handlers/projects_test.go` for the success cases
- [ ] write tests for the error cases: another principal's project, missing id, invalid body
- [ ] run `make test` and `make lint` — must pass before Task 3

### Task 3: Verify acceptance criteria
- [ ] verify every acceptance criterion in #41 and #42 holds
- [ ] run `make gen` and confirm `git diff --exit-code` is clean — no drift
- [ ] run `make test` and `make lint` — both exit 0

## Post-Completion
*Informational only — no checkboxes in this section.*

- Smoke-test the endpoints against a running `make db-up && make run`.
- Close #41 and #42 referencing the merged change.
````

### Format guardrails

- `- [ ]` checkboxes appear **only** inside `### Task N:` sections. A stray checkbox in Overview,
  Context or Post-Completion costs extra loop iterations.
- `Task` in the header is the token ralphex matches. It stays in English and keeps its number.
- Every task ends with its own test checkboxes as separate items, then
  `run make test and make lint — must pass before Task N+1`.
- When a task's behaviour only becomes observable in a later task, still list the test checkbox and
  name the dependency in it.
- Checkboxes are for work the agent can automate. Manual verification, deploys and issue bookkeeping
  go under `## Post-Completion`.
- ralphex moves the finished plan to `docs/plans/completed/` itself. Leave it in `docs/plans/`.

## 6. Print the next command

End by printing it literally, so the user can run it:

```
ralphex docs/plans/20260809-projects-api.md
```

Add `--worktree` to run it in an isolated git worktree — the way to run two plans in parallel — and
`--review` to skip task execution and run the review pipeline only over work already done.
