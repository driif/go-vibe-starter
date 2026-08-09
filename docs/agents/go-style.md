# Go house style

The conventions this repo is written in. [`AGENTS.md`](../../AGENTS.md) carries the short version;
this is the depth behind it. Where the two disagree, `AGENTS.md` wins and this file is stale — fix it.

## Packages and interfaces

- **Define an interface in the package that consumes it, not the one that implements it.** The
  consumer knows what it needs; the producer would only guess. The sqlc-generated `Querier` in
  `internal/db/gen` is the exception that proves it — it is generated for exactly this purpose and
  is the seam handlers and tests share.
- **Keep interfaces to 1–3 methods.** An interface with one implementation and no test double is
  not an abstraction, it is indirection. Delete it and use the concrete type.
- `internal/` is the hiding mechanism. Anything under it is invisible outside the module, so it can
  change freely. `pkg/` is a promise to the outside world — put something there only when you mean
  that promise.

## Functions and control flow

- **Return early.** Validate at the top, handle the error, return. Deep nesting is a smell that a
  guard clause is missing.
- **`context.Context` is the first parameter** on anything that blocks, does I/O, or can be
  cancelled. Never store one in a struct field.
- Name the receiver consistently across every method on a type, and keep it short.
- Compile regexes once at package level (`var xRe = regexp.MustCompile(...)`), never per call.
- Keep package-level state immutable. A mutable package-level variable is shared state between
  every test in the package.

## Errors

- Check immediately and return. No `_ = err` outside of the narrow cases the linter excludes.
- Wrap with `fmt.Errorf("doing the thing: %w", err)` **when the context helps debugging**. One layer
  of useful context beats four layers of `failed to: failed to: failed to:`.
- Error strings are lowercase and carry no trailing punctuation.
- Sentinel errors are `var ErrThing = errors.New("thing")`, compared with `errors.Is`.
- HTTP handlers write errors through `errs.Write(w, status, err)` — never `http.Error` directly, so
  the response shape stays uniform.

## Logging

`log/slog`, structured, always. Never `fmt.Println`, never a third-party logger.

```go
slog.Info("note created", "note_id", id, "owner_id", p.Subject)
slog.Error("db query failed", "error", err, "query", "ListNotesByOwner")
```

Keys are `snake_case` and stable — they are what you grep production by. Log the error under the
`error` key. Do not log secrets, bearer tokens, or full request bodies outside development.

## Comments

Comment what a competent Go reader could not work out from the code itself.

Write one when the **why** is invisible (a driver quirk, a required statement order, a deliberate
refusal to retry), when a function's contract is not obvious from its signature (which return value
means "absent" versus "broken", what a zero value means), or when a non-obvious invariant could be
broken silently.

Do not restate what the line does, and do not justify that a change was made — reviewers read the
diff, and that belongs in the commit message. **No history comments**: "now uses X", "previously Y",
"changed to Z" all describe a moment that has passed. Describe the current state only.

In-code comments are lowercase. Doc comments on exported identifiers start with the identifier name
and are a full sentence. One or two sentences is right for most things; a doc comment over about six
lines needs a specific reason, and if the explanation needs more room it belongs in `docs/` behind a
one-line pointer.

## Tests

- Table-driven, with descriptive case names that read as sentences in the failure output.
- `testify`: `require` when the test cannot meaningfully continue, `assert` when it can.
- **One `_test.go` per source file.** `notes.go` is tested by `notes_test.go`, not by
  `notes_handlers_extra_test.go`.
- Helpers call `t.Helper()` so failures point at the caller.
- `t.TempDir()` for anything touching the filesystem. Tests never write to a real config directory.
- `httptest` for handlers. Fake the `Querier` interface rather than standing up a database, unless
  the thing under test *is* the SQL.
- Anything that needs Docker (testcontainers) is guarded by `if testing.Short() { t.Skip(...) }`.
- No shared mutable state between table cases — each case gets its own fixture.

## Database

SQL over ORM, deliberately. Schema changes are goose migrations in `migrations/`; queries are
hand-written in `sql/queries/` and compiled to Go by sqlc.

- Queries are parameterized (`$1`, `$2`), never string-concatenated.
- Scope every query by the caller's principal where the data is owned. Doing it in SQL
  (`WHERE id = $1 AND owner_id = $2`) rather than in Go means a forgotten check cannot leak rows.
- Never hand-edit `internal/db/gen/**` or `internal/api/*.gen.go`. Change the source and run
  `make gen`.

## Dependencies

Add one only when it earns its place. The standard library, chi, and what is already in `go.mod`
cover most of what this repo needs. A new direct dependency is a decision worth an ADR when it
touches the request path, auth, or persistence.
