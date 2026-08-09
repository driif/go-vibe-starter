---
name: tdd
description: Writing Go tests in this repo, test-first. Use when implementing a ticket or fixing a bug, when a change needs a regression test, when asked for "red-green" or "write a test for X", or when judging whether an existing test is worth keeping.
---

`red-green` is the loop. This skill is what makes the tests it leaves behind worth keeping: what a
good test is, where tests live, the shape they take here, and the anti-patterns that produce tests
which pass forever and catch nothing.

Worked examples of each shape: [`PATTERNS.md`](PATTERNS.md). The conventions this repo already
documents live in `AGENTS.md` and `docs/agents/go-style.md`.

## The loop

1. **Red.** Write the smallest failing test for one behaviour. Run it and read the failure: it must
   fail on the assertion, naming the behaviour that is missing. A test that failed to compile, or
   failed on a missing fixture, has not been red yet.
2. **Green.** Write the least code that satisfies it. Run the one test while iterating:
   `go test -race -count=1 -run TestCreateNoteRejectsEmptyTitle ./internal/api/handlers`. `-count=1`
   defeats the test cache, so a green result is a real run.
3. **Refactor.** Behaviour is now pinned. Reshape the implementation with the tests green.

One `red-green` per behaviour, in the order the `tracer bullet` walks the slice. Each cycle teaches
the next one what the interface actually wants; a batch of tests written up front pins imagined
behaviour instead.

**Completion criterion:** `make test` and `make lint` both exit 0, and every test you added has been
observed red at least once.

## What a good test is

- **Asserts behaviour at the package's public surface.** The observable output — the HTTP status and
  body, the returned value, the error — not the route the code took to produce it. The implementation
  can be rewritten whole; the test should survive untouched.
- **Names the capability.** `TestCreateNoteRejectsEmptyTitle` and table cases like
  `"missing owner returns 404"` read as a specification of the package.
- **Draws expected values from an independent source** — the OpenAPI schema, the issue, a literal you
  worked out by hand. An expectation computed the way the code computes it agrees with the code by
  construction.
- **Fails for one reason,** and the failure message says which behaviour broke.

## Where tests live

- One `_test.go` beside each source file, in the same package: `internal/api/handlers/notes.go` →
  `internal/api/handlers/notes_test.go`. New behaviour goes in the existing file for that source
  file.
- **Handlers** go through `net/http/httptest` against the real chi route, so the middleware chain and
  the `principal` are part of what is under test.
- **Anything needing Postgres** uses `internal/server/test`: `test.E2e` boots a testcontainers
  Postgres, applies the migrations and registers the routes. It needs Docker, so gate it with
  `testing.Short()` and keep `go test -short ./...` green on a machine without a daemon.
- **Anything touching the filesystem** uses `t.TempDir()`, which cleans itself up.
- Every helper starts with `t.Helper()`, so a failure points at the caller's line.

## Shape

Table-driven with `t.Run` subtests and descriptive case names. `require` for anything the rest of the
case depends on (a non-nil response, a decode that must succeed) — it stops the case. `assert` for
independent field checks, so one wrong field still reports the other five.

Build each case's fixtures **inside** the loop body. A table whose cases share a pointer, a map, or a
`*sql.DB` row set passes or fails depending on order, and `-race` will eventually find it for you at
the worst moment.

## The seam to fake

Each handler type declares, at the consumer, the slice of the sqlc-generated querier it uses —
`noteStore` in `internal/api/handlers/notes.go` is the pattern. That interface is the `seam` for
handler tests: a small struct implementing its two or three methods drives the error paths Postgres
makes awkward to trigger. Take the seam the code already has rather than introducing a new interface
for the test's benefit.

Reach for the real database through `test.E2e` when the behaviour under test *is* the SQL — the
`owner_id` scoping, a unique constraint, a migration's default.

## Anti-patterns

Each has a tell. When you see the tell, the test is not paying for itself.

- **Asserting on the fake.** The test checks that `CreateNote` was called with certain arguments and
  never looks at the response. *Tell:* the assertions name the double, not the behaviour. Assert on
  what the caller can observe.
- **Restating the implementation.** The expected value is recomputed the way the handler computes it,
  or the test walks the same branches in the same order. *Tell:* changing the code and the test
  together always keeps it green. Assert against an independent expectation.
- **The test that cannot fail.** No assertion after the call, an error assigned to `_`, an assertion
  inside an `if` that the fixture never enters, or a subtest that returns early. *Tell:* it stayed
  green when you broke the code on purpose — which is why step 1 insists on seeing red.
- **Over-mocking.** A hand-rolled double for `*sql.DB`, the HTTP client, and the clock, to test one
  branch. *Tell:* the test file is longer than the code and asserts mostly on wiring. Use
  `gen.Querier`, `httptest`, and an injected time value.
- **Shared mutable state across cases.** Cases mutate a package-level fixture or a slice declared
  outside the loop. *Tell:* the suite passes with `-run` on one case and fails on the whole file, or
  fails under `-race`. Construct fresh state per case.
