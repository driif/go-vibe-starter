# go-smells

Judge the branch against this repo's own conventions and against the shapes that make Go code hard to
read. Read-only.

Read `AGENTS.md` and `docs/agents/go-style.md` first — they are the standard. Then get your own diff
(`git diff <base>...HEAD`, plus `git status --short`, `git diff` and `git diff --cached`, since work
here is normally left uncommitted) and read each changed file whole, beside the code already in that
package: consistency with the surrounding file is evidence, personal preference is not. Placement
smells are invisible inside a hunk. Cite the rule or the existing code behind every finding.

Skip anything `golangci-lint` already enforces (`make lint`); a linted violation is not worth a
review finding.

## Conventions

1. **Logging** through `log/slog`, structured, with key-value attributes.
2. **Errors** written from handlers with `errs.Write(w, status, err)`; wrapped with
   `fmt.Errorf("...: %w", err)` where the context aids debugging, and only once.
3. **Comments** lowercase in-code, godoc starting with the identifier's own name on every exported
   identifier, describing the current state and the *why*. A comment recording history ("now uses
   X", "previously Y", "changed to Z") belongs in `git log` and is a finding wherever it lands.
4. **Interfaces** declared at the consumer, one to three methods. An interface declared next to its
   implementation is a producer interface and a finding, even where it compiles fine.
5. **Control flow** returns early; input validation sits at the top of the function. Naked returns
   only in a function short enough to see whole. `v.(T)` without the `, ok` form, outside a type
   switch, is an unchecked assertion.
6. **Context** is `ctx context.Context` as the first parameter, never a struct field, never
   `context.TODO()` in shipped code, and the request's own context reaches the database.
7. **Constants and state**: named constants for timeouts, limits, roles and header names;
   package-level `var` that changes at runtime and `init()` that does work are findings;
   `regexp.MustCompile` sits at package level, not inside the matching function.
8. **Panics** stay out of construction — this repo took the panic out of `server.NewWithConfig`
   deliberately. Error strings are lowercase and unpunctuated.
9. **Layout** follows the repo's shape: handlers in `internal/api/handlers`, one `routes_*.go` per
   domain called from the `hub`, middleware in `internal/server/middleware`, config read only in
   `internal/server/config`, migrations in `migrations/`, queries in `sql/queries/`.
10. **Naming and imports** match the package's existing style — no stutter (`handlers.HandlerNote`),
    consistent receiver names, `any` over `interface{}`, error variable names, and the
    stdlib / third-party / local import grouping.
11. **Generated files** are never hand-edited. An edit inside `internal/api/*.gen.go` or
    `internal/db/gen/` is `critical`.
12. **No AI attribution** anywhere: not in a comment, a commit message, a PR body or an issue. A
    `//nolint` carries its reason on the same line.

## Smells

Each is a judgement call, and a documented repo convention overrides it. Name the smell, quote the
hunk, and give the move.

- **Mysterious name** — a function, variable or type whose name hides what it does. → Rename; if no
  honest name exists, the design is unclear.
- **Duplicated logic** — the same shape in more than one hunk. → Extract it and call it from both.
- **Long function** — several jobs under one name. → Split at the seams the comments already mark.
- **Deep nesting** — three or more levels of `if` inside a function. → Invert the conditions and
  return early.
- **Magic literal** — an unexplained number or string. → A named constant beside the rule it encodes,
  as `maxNoteTitleLen` mirrors `oapi/openapi.yaml`.
- **Feature envy** — a function reaching through another type's fields more than its own. → Move it
  onto the type it envies.
- **Primitive obsession** — a bare `string` or `int` standing in for a domain concept with rules. →
  Give the concept a small type.
- **Data clump** — the same three parameters travelling together everywhere. → One struct.
- **Mixed abstraction levels** — a function doing HTTP decoding and SQL parameter assembly in the
  same twenty lines. → Separate the levels.
- **Shotgun surgery** — one logical change spread across many files in this diff. → Say which module
  the scattered pieces want to live in.
- **Divergent change** — one file edited for several unrelated reasons. → Split it so each file
  changes for one reason.

## Boundaries

Bugs belong to `go-quality`, over-engineering to `go-simplification`, missing docs to
`go-documentation`, tests to `go-testing`. Spend your attention on what the linter cannot see:
placement, naming and intent.

## Report

Problems only — a clean pass is an empty report. Re-read each candidate at `file:line` before you
write it down and report only the **confirmed** ones; anything you cannot point at is a false
positive you keep to yourself.

One entry per finding:

- `SEVERITY: path/to/file.go:LINE — one-line claim` (critical / major / minor; most live at `minor`).
- **Convention** — the `AGENTS.md` rule, the `docs/agents/go-style.md` line, or the existing code
  that sets the pattern. For a smell, name the smell.
- **Fix** — the specific change.

Findings only.
