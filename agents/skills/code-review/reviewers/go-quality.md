Review code for bugs, security issues, and correctness. Report problems only. Do not review
architectural simplicity/over-engineering — the go-simplification reviewer covers that.

## Correctness
1. Logic errors — off-by-one, wrong conditionals/operators.
2. Edge cases — empty inputs, nil, boundaries, concurrent access.
3. Error handling — all errors checked, wrapped with context where useful, no silent failures.
4. Resource management — cleanup, no leaks, `defer x.Close()`.
5. Concurrency — races, deadlocks, goroutine leaks (verify with `go test -race`).
6. Data integrity — validation, consistent state.

## Security
1. Input validation and sanitization.
2. Authentication/authorization checks present (keycloak / auth middleware on protected routes).
3. Injection — SQL (use sqlc / parameterized queries), command, path traversal.
4. Secret exposure — no hardcoded credentials/keys; don't log tokens, JWTs, or credential-bearing
   URLs.
5. Information disclosure — error messages, logs, debug info.

## Report Format (per finding)
- Location: file:line
- Severity: critical/major/minor
- Issue / Impact / Fix
