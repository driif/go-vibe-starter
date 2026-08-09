# go-documentation

Find documentation the branch should have updated and documentation the branch made wrong.
Read-only: report the gap and draft the text, never edit the file.

Read the current `README.md`, `AGENTS.md`, `CONTEXT.md`, `.env.example`, `docs/agents/*.md` and
`docs/adr/*.md` before you judge anything — a gap is only a gap when the item is not already
documented somewhere in that set. Then get your own diff with `git diff <base>...HEAD`, plus
`git status --short`, `git diff` and `git diff --cached`, since work here is normally left
uncommitted.

## What each document owes the change

**`README.md`** — the human-facing surface. It must cover a new endpoint, a new CLI command or flag,
a new make target, a new environment variable, changed user-visible behaviour, a new dependency or
system requirement, and anything that breaks an existing setup. It owes nothing to an internal
refactor, a bug fix restoring documented behaviour, or a test-only change.

**`AGENTS.md`** — the agent-facing single source of truth. It must cover a new architectural pattern
the branch establishes, a new convention, a new Makefile target, a new library, and a change to the
directory layout. A new or edited skill under `.claude/skills/` owes it a skill-index row **and** a
matching `agents/skills/` hunk in the diff, which is what `make sync-skills` produces. Code that
follows an existing pattern owes it nothing.

**`.env.example` and `docs/agents/env-reference.md`** — every variable the config package reads
appears in both, with its default and a one-line comment. A new `env.GetEnv*` call with no entry in
either is `major`; the two files disagreeing with each other is `major`.

**`CONTEXT.md`** — the ubiquitous language and module map. A new domain term in the code, or a
package whose responsibility changed, belongs here.

**`docs/adr/`** — a decision the branch makes that a future reader would otherwise have to
reverse-engineer (a swapped library, a rejected approach, a `seam` deliberately placed). Report the
missing ADR; do not write it. A decision that contradicts an existing record means that ADR needs
superseding, and that is `major`.

**`oapi/openapi.yaml`** — the API contract, and the source the handler types generate from. A
behaviour visible over HTTP that the spec does not describe is a finding, as is a description that no
longer matches the handler.

**Godoc** — every exported identifier the branch adds carries a comment starting with its name.

**The plan**, when one is in play (`docs/plans/`) — report completed work whose checkboxes are still
unticked, and status that no longer matches the tree. Report it; leave the file alone.

## Staleness

Sweep for documentation the change quietly falsified: a renamed flag still documented by its old
name, a changed default, a removed endpoint still listed, a make target that no longer exists, a
comment describing behaviour the diff replaced.

## Boundaries

Comment *style* — lowercase in-code comments, history comments — belongs to `go-smells`. You cover
whether the documentation is true and complete.

## Report

Problems only — a clean pass is an empty report. Re-read each candidate at `file:line` before you
write it down and report only the **confirmed** ones; anything you cannot point at is a false
positive you keep to yourself.

One entry per gap:

- **Trigger** — `path/to/file.go:LINE`, the change that creates the need.
- **Missing or wrong** — what is absent or no longer true.
- **Where** — the file and section it belongs in.
- **Draft** — the text to add, written in the voice of that document. Plain, specific, no AI
  attribution and no mention of how the change was produced.
