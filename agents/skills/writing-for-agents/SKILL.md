---
name: writing-for-agents
description: Writing documents an agent consumes in this repo. Use when creating or editing a skill under `.claude/skills/`, when editing `AGENTS.md` or `CLAUDE.md`, or when writing a reference doc a skill points at (`docs/agents/*`, a sibling `.md` file).
---

Reference for any document an agent reads here: a skill, `AGENTS.md`, `CLAUDE.md`, a doc reached by a
pointer. The packaging differs, the writing does not. The same levers make each one predictable,
meaning the agent takes the same *process* every run, not that it produces the same output.

Writing a skill? Also read [`SKILL-MECHANICS.md`](SKILL-MECHANICS.md) for frontmatter, the
invocation choice, and router skills.

## Where these documents live

Canonical skills are `.claude/skills/<name>/SKILL.md` plus sibling reference `.md` files;
`make sync-skills` mirrors the tree to `agents/skills/` for Codex, Pi and opencode. Two consequences
for the writing:

- **Write steps any of those agents can run**: shell commands, `make` targets, file edits, `gh`
  calls. Where Claude-only tooling is genuinely faster (parallel read-only subagents from
  `.claude/agents/`), name it as the fast path and give the sequential fallback in the same step.
- **`AGENTS.md` is the single source of truth** for stack, conventions, config and commands. A skill
  that restates it is duplication: point at the section instead.

## Context pointers

A **context pointer** is a reference held in the agent's context that names out-of-context material
and encodes the condition for reaching it. A skill's description is one. So is the `AGENTS.md` line
naming `docs/agents/go-style.md`. The pointer's *wording*, not its target, decides when the agent
reaches the material and how reliably. A must-have target behind a weakly worded pointer is a
variance bug: sharpen the wording first, and inline the material only if sharpening fails.

A pointer does two jobs: state what the material is, and list the **branches** that should trigger
reaching it (a branch is a distinct case the document handles, so different runs take different paths
through it). Every word of an always-loaded pointer costs on every turn, so front-load the leading
word, write one trigger per branch (synonyms renaming a single branch are one branch written twice),
and cut identity the body already carries.

## The two loads

Every document and pointer you add spends one of two budgets.

- **Context load** — always-loaded material on the agent's window: an `AGENTS.md` line, a skill
  description, anything sitting in context every turn whether or not it fires.
- **Cognitive load** — the cost on the human: which documents exist and when to reach for each. The
  human is the index. Not a cost to minimise; it is the price of human agency. Spend it where human
  judgement matters, remove it where it does not.

Material reached only through a pointer escapes context load at the price of the pointer's own line;
material with no pointer at all rides entirely on cognitive load.

## Information hierarchy

A document mixes two content types: **steps** (ordered actions the agent performs) and **reference**
(definitions, rules, facts consulted on demand). `go-feature` is mostly steps, `code-review` mostly
reference, this skill all reference. The decision is where each piece sits on the **information
hierarchy**, a ladder ranked by how immediately it is needed.

1. **In-file step** — the primary tier: what the agent does, in order.
2. **In-file reference** — consulted on demand. Often a legitimately flat peer-set (every Go
   convention on one rung), which is a fine arrangement, not a smell.
3. **Disclosed reference** — pushed into a separate file behind a pointer, loaded only when the
   pointer fires. Spans a sibling file (`go-feature/SQLC.md`) through external reference any document
   can point at (`docs/agents/env-reference.md`).

Push too little down and the top bloats; push too much and you hide material the agent needs.

**Progressive disclosure** is the move down the ladder so the top stays legible. Branching is the
cleanest test: inline what every branch needs, disclose what only some branches reach. `go-feature`
keeps the ordered slice in `SKILL.md` and discloses `SQLC.md`, `OPENAPI.md` and the rest, because a
given feature touches only some of them in depth.

**Co-location** is the within-file companion: the ladder decides how far down a piece sits,
co-location decides what sits beside it once there. Keep a concept's definition, rules and caveats
under one heading so reading one part brings its neighbours.

**Sprawl** is the failure mode: a document too long even when every line is live and unique.
Attention thins across the excess. The cure is the ladder. Keep every `SKILL.md` under ~120 lines.

## Steps and completion criteria

Every step ends on a **completion criterion**, the condition telling the agent the work is done.

- **Clarity** — can the agent tell done from not-done? "Tests look fine" invites **premature
  completion**; "`make test` and `make lint` both exit 0" does not. The visible steps still ahead
  supply the pull to rush; the criterion's clarity is the resistance. Sharpen the bound first, it is
  local and cheap. Only if it stays fuzzy *and* you observe the rush, split the sequence so the later
  steps sit behind a real context boundary (a hand-off or a subagent dispatch; an inline call leaves
  them in context and clears nothing).
- **Demand** — how much it requires. "Every handler in `internal/api/handlers` accounted for" forces
  thorough work where "review the handlers" does not. Demand drives the digging done inside a step,
  latent in the wording rather than written as its own step, and it is not step-bound: "every
  finding verified at `file:line`" binds flat reference just as "every step done" binds a sequence.

The strongest criteria are both checkable and exhaustive.

## When to split

Splitting one document into two spends one of the two loads, so split only when the cut earns it.

- **By sequence** — split a run of steps where the later ones tempt the agent to rush the one in
  front of it. Keeping them out of view drives more work on the current task.
- **By invocation** — skill-specific: see [`SKILL-MECHANICS.md`](SKILL-MECHANICS.md).

## Language

Which words the document runs on is its own reference: **leading words**, the shared vocabulary
every skill in this pack reuses verbatim, prompting the **positive** rather than the prohibition,
and pruning **sediment** and **no-ops**. All of it in [`LANGUAGE.md`](LANGUAGE.md) — read it before
writing prose, not after.

One rule stays here because it is a guardrail: state the target behaviour, never the ban. A
prohibition earns its place only where no positive phrasing exists ("never hand-edit `*.gen.go`"),
and it travels with the positive target it protects.
