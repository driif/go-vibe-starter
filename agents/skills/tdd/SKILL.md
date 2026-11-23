---
name: tdd
description: Writes Go tests in this repo test-first, red-green-refactor. Use when implementing a ticket or fixing a bug, when a change needs a regression test, when asked for "red-green" or "write a test for X", or when judging whether an existing test is worth keeping.
---

Use red-green-refactor for meaningful new behavior. Read [PATTERNS.md](PATTERNS.md) for test shapes
and `AGENTS.md` for conventions. Tests assert caller-visible behavior and independent expected values.

1. Write the smallest failing test for one behavior. Run it and confirm the failure names the
   missing behavior; a compiler error or missing fixture is not that failure.
2. Write the least implementation that satisfies it. Iterate with a focused command such as
   `go test -race -count=1 -run TestWhoAmI ./internal/rpc/whoami`.
3. Refactor while that behavior stays pinned. Follow the tracer bullet through each required layer.

Handler tests use generated clients against `test.NewServer`, and store tests use `test.Database`.
The real HTTP stack, OIDC issuer and Postgres are the fixtures. Docker is required. Tests importing
`internal/test` use an external test package. Pure function tests may use a private-package seam.

Use descriptive table cases, fresh per-case fixtures, `t.Helper`, `t.TempDir` and `t.Setenv`.
Environment mutation cannot run in parallel. `require` stops dependent assertions; `assert` covers
independent fields. A foreign owner's row must be not-found; include error reasons and invalid input.

Reject tests that recompute the implementation's expected value, assert only which method was
called, have no meaningful assertion, or share mutable state across cases. Fakes belong at genuine
external seams, not between a handler and its local Postgres store. Do not add tests that merely
mirror reversible implementation details.

Completion criterion: meaningful new behaviors have been observed failing and then passing, and
`make check` exits 0.
