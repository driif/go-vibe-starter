# go-simplification

Find code on this branch that works but costs more complexity than the problem requires. Read-only.

Scope: over-engineering **this branch introduces or makes worse**. Pre-existing complexity the branch
does not touch is out of scope, and complexity the spec explicitly required is not a finding. Skip
generated output (`internal/api/*.gen.go`, `internal/db/gen/`) and vendored code.

Get your own context with `git diff <base>...HEAD`, plus `git status --short`, `git diff` and
`git diff --cached`, since work here is normally left uncommitted. Before reporting anything as
*unused*, *no callers* or *never triggers*, prove it: search the whole repo including `_test.go`
files, `oapi/openapi.yaml`, the `Makefile`, `scripts/` and the router files, and cite the search in
the finding. Config-driven and route-registered callers do not look like callers.

The `DEMO` notes slice exists to demonstrate **the slice**, so its existence is never a finding;
complexity inside it beyond that demonstration still is.

The target shape is a `deep module`: a small exported surface over an implementation that carries
real weight. Findings here are the inverse — surface added without weight behind it.

## Abstraction with nothing behind it

- A wrapper whose method calls one other method with the same signature.
- A constructor or factory for a single concrete type that no test substitutes.
- An interface declared in the package that implements it. Interfaces belong at the consumer —
  `noteStore` in `internal/api/handlers/notes.go` is the pattern.
- Handler → service → repository where a layer only forwards. In this repo a handler calling the
  sqlc-generated querier is the whole stack.
- Two types for the same data with converters between them, where the generated
  `internal/api` type would have served.

## Generalization ahead of need

- A generic mechanism serving one case: a registry with one entry, an event bus with one event.
- A new `internal/` package holding one function its only caller could hold.
- An options struct or functional-options set for two or three parameters.
- A new env var in `internal/server/config` that only ever takes one value. Every knob costs an
  `.env.example` line, a doc line and a branch to test forever.
- An extension point — hook, callback, plugin, middleware slot — with no second implementer.
- One struct with many optional fields standing in for several distinct shapes.

## Indirection

- A builder for a value a literal would express.
- A named type wrapping a stdlib primitive with no method and no invariant.
- Several middlewares in `internal/server/middleware` that do one job between them.
- A helper called once, whose body would read better inline.

## Fallbacks and future-proofing

- A default branch whose condition cannot be met.
- Old and new implementations side by side with nothing calling the old one.
- A feature flag for a decision that is already permanent.
- An error caught and swallowed into a fallback value, where failing loudly is the correct behaviour.

## Optimization ahead of measurement

- A cache over data read once at startup, or over a query that runs rarely.
- A worker pool, a custom data structure, or a hand-rolled index where a slice or map is fine.
- Anything justified by a performance claim with no measurement behind it.

## Boundaries

Bugs belong to `go-quality`, convention and placement to `go-smells`, missing pieces to
`go-implementation`. Stay on complexity this branch added.

## Report

Problems only — a clean pass is an empty report. Re-read each candidate at `file:line` before you
write it down and report only the **confirmed** ones; anything you cannot point at is a false
positive you keep to yourself.

One entry per finding:

- `path/to/file.go:LINE — one-line claim`
- **Pattern** — which of the above.
- **Problem** — the complexity it costs a future reader.
- **Simplification** — what the simpler code looks like.
- **Effort** — `trivial` / `small` / `medium` / `large`.

A `large` finding is a recommendation for a follow-up issue, not a change to make inside the review.
Say so in the entry.
