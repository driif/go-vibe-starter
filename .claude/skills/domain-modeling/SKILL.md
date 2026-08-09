---
name: domain-modeling
description: Build and sharpen this project's domain model. Use when pinning down domain terminology or a ubiquitous language, when a decision is worth recording as an ADR, when `CONTEXT.md` disagrees with the code, or when another skill needs the domain model maintained as it works.
---

The *active* discipline: challenge terms, invent edge-case scenarios, and write the result down the
moment it crystallises. Reading `CONTEXT.md` for vocabulary is a one-line habit any skill can do —
this skill is for changing the model.

Two artifacts, both in the working tree:

- **`CONTEXT.md`** (repo root) — the ubiquitous language plus a short module map. Shape:
  [`CONTEXT-FORMAT.md`](CONTEXT-FORMAT.md).
- **`docs/adr/NNNN-<slug>.md`** — one decision per file. Shape and numbering:
  [`ADR-FORMAT.md`](ADR-FORMAT.md).

Create each lazily: the first resolved term creates `CONTEXT.md`, the first qualifying decision
creates `docs/adr/`. `make init` resets both for a fresh project, so write them as a new team's
starting point rather than a history.

## During the session

**Challenge against the glossary.** When a term the user reaches for conflicts with the one in
`CONTEXT.md`, say so in the round it appears: "`CONTEXT.md` defines a Note as owner-scoped, but you
just described one shared across users — which is it?"

**Sharpen fuzzy language.** When a word is vague or carries two meanings, propose one canonical term
and name the losers so they can be avoided. "You are saying *account* — do you mean the **principal**
carried in the request context, or the row in `users`? Those are different things."

**Stress-test with concrete scenarios.** Invent specific cases that probe the boundary between two
concepts and force a precise answer. "A note whose owner is deleted: does the note disappear, or does
it outlive its owner as an orphan?"

**Cross-reference with code.** A term that exists claims a footprint: a table in `migrations/`, a
query in `sql/queries/`, a schema in `oapi/openapi.yaml`, a package under `internal/`. Check what is
actually there and surface any contradiction. "The API exposes `PATCH /v1/notes/{id}`, but you just
said a note is immutable once created — which is right?"

**Write it down inline.** The moment a term settles, edit `CONTEXT.md`; the moment a decision meets
all three tests below, write the ADR. Capture them as they land rather than batching to the end,
where they get lost.

**Keep the module map current.** When the session moves a responsibility between packages, or adds
one, update the map rows in `CONTEXT.md` to match `internal/` and `pkg/` as they now stand.

## Guardrails

`CONTEXT.md` holds terms and the module map only. Implementation detail belongs in the code,
conventions in `AGENTS.md`, decisions in `docs/adr/`. The doctrine words this skill pack itself runs
on (`the slice`, `tracer bullet`, `deep module`) live in `writing-for-agents` — `CONTEXT.md` carries
the *product's* language.

## Offer an ADR only when all three hold

1. **Hard to reverse** — changing your mind later costs real work.
2. **Surprising without context** — a future reader will look at the code and ask why.
3. **A real trade-off** — there were genuine alternatives and one was picked for stated reasons.

Miss any one and skip it: easily reversed decisions get reversed, unsurprising ones are never
questioned, and a decision with no alternative records nothing. `ADR-FORMAT.md` lists what qualifies
in a repo shaped like this one.

## Done

Every term the session argued about is either defined in `CONTEXT.md` or was deliberately rejected
there under `_Avoid_`; every decision passing all three tests has a file in `docs/adr/`; the module
map matches the tree. Tell the user which files you touched.
