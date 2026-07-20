---
name: tdd
description: Use when implementing a feature or bugfix and tests are wanted first — red-green-refactor with this repo's Go test conventions (table-driven, testify require/assert, go test -race).
---

# TDD

Red-green-refactor for Go, using this repo's conventions. See `AGENTS.md` for the full set.

## Cycle

1. **Red.** Write the smallest failing test first. Run it and confirm it fails for the right
   reason.
2. **Green.** Write the minimal code to pass. Run the package (or `go test -race ./...`) until
   green.
3. **Refactor.** Clean up with tests green; keep behavior identical.

## Conventions

- Standard library `testing`; `stretchr/testify` — `require` for fatal assertions, `assert` for
  non-fatal.
- **Table-driven tests** for multiple cases; give each case a descriptive name.
- One test file per source file (`foo.go` → `foo_test.go`).
- HTTP handlers/middleware: use `net/http/httptest`.
- Servers: bind `127.0.0.1:0` for an OS-allocated port; always `defer srv.Close()`.
- Run with the race detector: `make test` (`go test -race ./...`).
