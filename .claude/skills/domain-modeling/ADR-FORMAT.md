# ADR format

One architecture decision per file in `docs/adr/`, named `NNNN-<slug>.md` — a four-digit
zero-padded number and a kebab-case slug of the title (`0003-auth-provider-seam.md`).

## Numbering

Read `docs/adr/`, take the highest existing number, add one, zero-pad to four digits. An empty or
missing directory starts at `0001`. Numbers are never reused, including for a decision later
reversed — the reversal gets its own number.

## Template

```md
# NNNN. <Short title of the decision>

- Status: accepted
- Date: <YYYY-MM-DD>

## Context
<What forced the decision: the constraint, the pressure, the alternatives on the table.>

## Decision
<What was decided, in the present tense and in the project's own vocabulary.>

## Consequences
<What this buys, what it costs, and what now has to be true. Name the follow-up work.>
```

`Status` is one of `accepted`, `superseded by ADR-NNNN`, or `deprecated`. Superseding edits the old
file's status line and nothing else — the old context stays readable — and the new ADR names what it
replaces in its own Context section.

Keep each section short. A three-line Context that names the real pressure beats a page of
background.

## What qualifies

Decisions in a repo shaped like this one that meet the three tests in
[`SKILL.md`](SKILL.md#offer-an-adr-only-when-all-three-hold):

- **Seams that carry lock-in** — the `AUTH_PROVIDER` switch between `none` and `keycloak`, the
  database, anything that would take weeks to swap.
- **Deliberate deviation from the obvious path** — sqlc and hand-written SQL instead of an ORM,
  OpenAPI-first codegen instead of hand-rolled routes. These stop the next engineer from "fixing"
  something that was chosen.
- **Boundary and ownership rules** — which package owns a concept, which data a **principal** may
  reach, what one domain refuses to know about another.
- **Process substrate** — GitHub Issues as the planning substrate, `docs/plans/` as execution only.
- **Constraints invisible in the code** — a compliance rule, a latency budget a partner contract
  imposes, a version pinned because of an upstream bug.
- **Rejected alternatives whose rejection is non-obvious** — when the reason is subtle, record it, or
  the same suggestion returns in six months.

Skip the routine: a library that could be swapped in an afternoon, a naming choice, anything the
code already makes obvious on reading.
