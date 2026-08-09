---
name: diagnosing-bugs
description: Working a bug down to its cause before changing anything. Use when something is broken, throwing, hanging, flaking or slow, when a test fails, when an issue carries the `bug` label, or when the user says "debug this" or "why is this failing".
---

Six phases in order: reproduce → minimize → hypothesize → instrument → fix → verify. The gate sits at
the end of phase 1 — no fix is proposed and no source file is edited until a command reproduces the
bug on demand.

## 1. Reproduce

Build one command that fails on demand. In descending order of preference:

1. A Go test at the package where the bug lives:
   `go test -race -count=1 -run TestNotesGetRejectsForeignOwner ./internal/api/handlers`.
2. An `httptest` request through the real chi route, so middleware and the `principal` are in play.
3. `curl` against `make run`, with the request body the reporter actually sent.
4. A `cmd/` invocation: `./bin/app db migrate up`, or whatever subcommand carries the failure.

`-count=1` defeats the test cache, so green means a real run. Reach for `test.E2e` in
`internal/server/test` when the bug needs a real Postgres.

The command qualifies when it:

- fails on **the symptom the reporter described**, not a nearby failure — the wrong repro leads to
  the wrong fix;
- returns the same verdict every run;
- finishes in seconds;
- runs unattended.

**Intermittent bugs:** the goal is a higher reproduction rate, not a clean one. `-count=100`,
`-race`, `-parallel 8`, a tightened timeout, an injected delay at the suspect window. A 50% failure
rate is debuggable; 1% is not — keep raising it.

**Completion criterion:** you can name one command, you have run it, and you have shown its failing
output. If nothing reproduces, stop and say so: list what you tried and ask for the one thing that
would close the gap — the exact request, the environment, the log window around the failure. Reading
code to build a theory without this command is the failure this skill exists to prevent.

## 2. Minimize

Shrink the failing case one cut at a time — request fields, middleware, config values, rows, callers,
concurrency — rerunning the command after each cut and keeping only cuts that leave it red.

Done when every remaining element is load-bearing: removing any one of them makes the failure
disappear. What is left is both the hypothesis space for phase 3 and the regression test in phase 5.

## 3. Hypothesize

Write **three to five ranked causes before testing any of them**. Generating one at a time anchors
you to the first plausible story.

Each must be falsifiable — state the prediction: *if the owner scoping is applied after the lookup,
then passing a foreign id will return the row rather than 404.* A cause you cannot phrase as a
prediction is a vibe; sharpen it or drop it.

Show the ranked list before probing. The user often re-ranks it in one line ("that config shipped
yesterday").

## 4. Instrument

One probe per prediction, one variable changed at a time.

- **Targeted `slog` at debug level** on the boundary that separates two hypotheses, values in
  attributes: `slog.Debug("[dbg-a4f2] note lookup", "owner", owner, "id", id)`. `cmd/run.go` installs
  the default handler at `slog.LevelInfo`, so a debug probe is dropped until you lower it — in a test,
  `slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))`.
- **A focused test** asserting the intermediate value, which often survives as the regression test.
- **The request logger**, for request and response shape without touching code:
  `SERVER_LOGGER_LOG_REQUEST_BODY=true`, `SERVER_LOGGER_LOG_RESPONSE_BODY=true`,
  `SERVER_LOGGER_LOG_REQUEST_HEADER=true`.
- **`go test -race`** for anything involving shared state; `/debug/pprof/goroutine?debug=2` on a
  running server for a hang or a leak.

Tag every temporary probe with one prefix (`[dbg-a4f2]`) so removing them is a single grep.

**Completion criterion:** you can name the line where the wrong value first appears and the input
that makes it wrong. A killed hypothesis sends you back to phase 3 with what you learned — not to a
wider change.

## 5. Fix

Write the regression test first, at the seam where the bug actually occurs — see `tdd` for the shape
and where it lives. Watch it fail against the current code. Then make the smallest change that
addresses the **cause** you confirmed, and watch it pass.

If no seam can express the bug (the failure needs a call chain no test can reach), that is itself the
finding: report it alongside the fix, because the architecture is what stopped the bug from being
pinned down.

## 6. Verify

- The regression test fails on the old code and passes on the new. Prove it: `git stash`, run, then
  `git stash pop`.
- `make test` and `make lint` both exit 0.
- The phase-1 command, run against the original un-minimized scenario, is green.
- `grep -rn "\[dbg-" --include='*.go' .` prints nothing.
- The confirmed cause is written into the issue comment or the change description in one sentence,
  so the next reader inherits it.

Then answer once, now that you know more than you did at the start: what would have caught this
earlier — a validation at the boundary, a checked error, a test seam that does not exist yet?
