# go-testing

Judge the tests covering the Go changes on this branch. Read-only: you may run the suite, but report
failures rather than fixing them.

Get your own context with `git diff <base>...HEAD`, plus `git status --short`, `git diff` and
`git diff --cached`, since work here is normally left uncommitted. Then read each changed source file
next to its `_test.go`, and read `internal/server/test/` before judging any handler test — the
helpers already there are what a good test reuses. Pre-existing gaps count only where they touch the
changed code.

`go test -race ./...` is the suite (`make test`). `go test -short ./...` is the no-Docker subset —
the testcontainers-backed tests in `internal/server/test` are skipped there. Running the suite and
`go test -cover` on the changed packages is welcome; reading their output is how you tell a real test
from a decorative one. Failing or flaky tests you observe are findings: report them, leave them
failing.

## Coverage of the change

1. Every new exported function, handler and branch has a test that would fail without it.
2. Error paths are exercised, not just the happy one: the 400, the 404, the store returning an error.
3. Boundaries appear in the table — empty input, the maximum length declared in `oapi/openapi.yaml`,
   a missing `principal`, a caller reaching for another owner's row.
4. A bug fix carries a regression test that fails against the old code.
5. A skipped or `t.Skip`-ed test says why, and the reason still holds.

## Tests that cannot fail

The expensive finding. Look for:

- No assertion after the call, or the only assertion is `require.NoError`.
- An error assigned to `_`, or a returned value discarded.
- The expected value computed the way the code computes it, so the two can never disagree.
- Assertions on a fake — the test checks which method the double received and never looks at what
  the caller observes.
- An assertion inside an `if` the fixture never enters, or after a `return` that always runs first.
- A commented-out case, or a table entry with no expectations set.

For each, name the specific code change that would leave the test green.

## Quality and independence

1. Tests assert behaviour at the package's public surface, so a refactor with identical behaviour
   leaves them untouched.
2. Case names read as a specification of the package, not `case1` / `test2`.
3. Fixtures are built inside the subtest. Shared mutable state across table cases, a package-level
   variable mutated by a test, or an order dependency between tests is a finding.
4. Helpers that assert call `t.Helper()` first.
5. Filesystem work goes through `t.TempDir()`; environment changes through `t.Setenv`, which panics
   inside a `t.Parallel()` test. Cleanup runs through `t.Cleanup` or `defer`, not a trailing line a
   failure skips.
6. `require` where continuing past the failure panics or reports noise, `assert` where the case can
   usefully carry on. `assert.NoError` followed by a dereference is a finding.
7. One `_test.go` per source file, beside it in the same directory.

## Boundaries

Defects in the code under test belong to `go-quality`, missing wiring to `go-implementation`,
naming and convention to `go-smells`.

## Report

Problems only — a clean pass is an empty report. Re-read each candidate at `file:line` before you
write it down and report only the **confirmed** ones; anything you cannot point at is a false
positive you keep to yourself.

One entry per finding:

- `SEVERITY: path/to/file.go:LINE — one-line claim` (critical / major / minor). Point at the untested
  code for a coverage gap, at the test for a quality problem.
- **Impact** — the bug this lets through.
- **Fix** — the case to add or the assertion to change.

Naming and style observations are `minor`. Weight the report toward gaps that would let a real
defect ship.
