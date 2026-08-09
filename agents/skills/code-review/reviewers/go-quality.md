# go-quality

Find bugs and security holes in the Go changes on this branch. Read-only: report findings and leave
the working tree exactly as you found it.

Get your own context — `git diff <base>...HEAD` and `git log <base>..HEAD --oneline` — then read the
full file around each hunk, because a bug is usually visible only with the code the diff omits. Work
here is normally left uncommitted, so review `git status --short`, `git diff` and `git diff --cached`
as part of the change.

Architectural simplicity belongs to `go-simplification`; conventions and naming to `go-smells`. Stay
on defects.

## Correctness

1. **Errors.** Every returned error is checked at the call site — a `_ =` on anything but a
   deliberate writer is a finding. Comparison uses `errors.Is` / `errors.As`, never a string match.
   `sql.ErrNoRows` is mapped to a 404 rather than a 500. Wrapping adds context once, not at every
   layer, and no error is swallowed into a log line and then ignored. In a handler, every
   `errs.Write(w, status, err)` is followed by `return`; a missing `return` writes a second body.
2. **Nil and zero values.** The `ok` from `auth.PrincipalFromContext` respected before the
   `*keycloak.Principal` is dereferenced. A nil map written to, a nil pointer dereferenced on the
   error path, a `*sql.DB` used before `Initialize`, an empty slice indexed at `[0]`, a typed nil
   pointer stored in an interface and compared against `nil`.
3. **Boundaries.** Off-by-one, wrong comparison operator, an inverted condition, an early `return`
   that skips a required write, a `break` that only leaves the inner loop.
4. **Context.** `context.Context` is the first parameter and is threaded from `r.Context()` to the
   query. A long operation carries a timeout. `context.Background()` inside a request path drops
   client cancellation and is a finding.
5. **Resources.** Every `*sql.Rows`, response body and file is closed on all paths, `defer` included
   on the error branches. A `defer` inside a loop accumulates until the function returns.
6. **Concurrency.** Goroutines that outlive the request, a `WaitGroup` whose `Add` runs after the
   goroutine starts or that can never reach zero, a channel send with no receiver, package-level
   maps and slices mutated from handler goroutines, `httptest` servers and pool handles left
   running. Data races are what `go test -race ./...` finds — say so rather than guessing.
7. **HTTP.** The status code matches the outcome, the body is written once, and nothing writes after
   `WriteHeader`. Request bodies are size-bounded before decoding.

## Security

1. **Authorization.** Every route that touches user data sits behind the authenticator in its
   `internal/api/router/routes_*.go` — only `/health` and `/ready` sit outside the authenticated
   group — and every query is scoped to the `principal` subject, the `owner_id` pattern in
   `internal/api/handlers/notes.go`. A handler that trusts an id from the path or body without that
   scoping is `critical`, as is a config path that leaves `AUTH_PROVIDER=none` reachable under
   `APP_ENVIRONMENT=production`.
2. **Injection.** SQL reaches the database through sqlc-generated, parameterized queries. A
   hand-built query string, a `fmt.Sprintf` into SQL, or a user-controlled path joined without
   `filepath.Clean` is a finding.
3. **Input validation.** Lengths, formats and enums are checked at the top of the handler and match
   the constraints declared in `oapi/openapi.yaml`.
4. **Secrets.** No credential, token or key is hardcoded, logged, or carried in an error returned to
   the client. `slog` calls near auth are checked for a token or an `Authorization` header value.
5. **Disclosure.** `errs.Write` details describe what the caller did wrong; internal messages, driver
   errors and stack detail stay in the log.

## Report

Problems only — a clean pass is an empty report. Re-read each candidate at `file:line` before you
write it down and report only the **confirmed** ones; anything you cannot point at is a false
positive you keep to yourself.

One entry per finding:

- `SEVERITY: path/to/file.go:LINE — one-line claim` (critical / major / minor)
- **Impact** — what goes wrong at runtime, and for whom.
- **Fix** — the specific change.

Findings only. If a hunk is clean, say nothing about it.
