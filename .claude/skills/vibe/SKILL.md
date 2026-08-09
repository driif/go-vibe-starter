---
name: vibe
disable-model-invocation: true
description: The map of this skill pack — the flow from a loose idea to shipped code, and which skill to reach for.
---

# Vibe

The index for everything you invoke by name. The flow skills are user-invoked: they carry no
description the agent can see, so nothing but you reaches them, and remembering them is **cognitive
load** you are holding. This skill is where that index lives instead. It names the skill for the
situation and prints the command; you invoke it.

## The two substrates

- **GitHub Issues hold intent** — `spec` (the what and why), `ticket` (one **tracer bullet**),
  `decision` (an open question), `bug`. Blocking edges are `Blocked by #N` lines in the body.
  This is the permanent record; `AGENTS.md` carries the label vocabulary.
- **`docs/plans/*.md` holds execution** — one file in ralphex's parsed format, run by the
  `ralphex` binary and archived to `docs/plans/completed/`. Generated from issues, disposable.

`/to-plan` is the bridge. Nothing else crosses between them.

## The main flow

```
loose idea → /grill-me → /to-spec → /to-tickets → /to-plan + ralphex → shipped
                                                → /implement
```

- **`/grill-me`** — a round-based interview that works the **frontier** until nothing is silently
  assumed. `/grill-with-docs` is the same interview, plus the terms it settles land in `CONTEXT.md`
  and the decisions that qualify become ADRs.
- **`/to-spec`** — synthesizes what the conversation already settled into a `spec` issue. It runs
  no interview, which is why it is cheap to reach for: grill first, spec second.
- **`/to-tickets`** — cuts the spec into `ticket` issues, one per tracer bullet, each thin enough
  to walk **the slice** end to end. Labels the frontier `ready` and the rest `blocked`.
- then the fork below.

### Which fork

|  | `/implement` | `/to-plan`, then `ralphex <file>` |
|---|---|---|
| Where you are | at the keyboard, watching each step | away; it runs unattended |
| Scope | one ticket | several tickets as one plan |
| When it hits something undecided | stops and asks you | keeps going on what the plan says |
| Leaves behind | a working tree and a closed issue | a working tree and an archived plan |

Reach for `/implement` when the ticket is a single slice and you want to see it happen. Reach for
`/to-plan` when the tickets are settled enough to walk away from, or when you want `ralphex
--worktree` running two plans in parallel.

## The on-ramps

Three situations start somewhere other than a loose idea. Each merges onto the flow at a named
point.

- **A repo just created from the template** → `/setup-project`, once, before anything else: `gh`
  auth, `make labels`, the module path, a `CONTEXT.md` that says something, the docs directories.
  `/to-spec`, `/to-tickets` and `/triage` all assume it has run. Merges at the top.
- **A bug arrives** → file it with the `bug` label, then `/triage` to give it a state. Reproducing
  before fixing is a discipline that fires on its own. Once the issue is `ready` it is a ticket
  like any other. Merges at `/implement`.
- **Work too big to hold in one session** → `/wayfinder`. Charts the destination as a `[map]`
  issue plus one `decision` issue per open question, then resolves them one per session until the
  frontier is empty. Merges at `/to-spec`.

## Standalones

- **`/triage`** — the label state machine over the whole tracker. Run it when issues have piled up
  unlabelled or when blockers have closed and their dependents should move. It holds one gate:
  nothing reaches `ready` without acceptance criteria.
- **`/handoff`** — context is running out or the session is ending. Writes a compact document a
  fresh session starts from, and pastes it into the conversation.
- **`/improve-architecture`** — periodic maintenance after a run of features has landed, not a
  step in the flow. Scans for deepening opportunities, ranks them, grills the one you pick. Ends
  in a decision, which rejoins at `/to-spec`.

## Reach for

| Situation | Skill |
|---|---|
| Repo just created from the template | `/setup-project` |
| The idea is fuzzy and you want it stress-tested | `/grill-me` |
| …and the vocabulary it settles should stick | `/grill-with-docs` |
| The conversation has landed; write it down | `/to-spec` |
| A spec exists; cut it into work | `/to-tickets` |
| Tickets are settled and you want to walk away | `/to-plan`, then `ralphex` |
| One ticket, and you want to watch it happen | `/implement` |
| A bug report just arrived | `/triage`, then `/implement` |
| Too big or too foggy to plan in one pass | `/wayfinder` |
| The tracker has drifted out of date | `/triage` |
| Context is running out | `/handoff` |
| The codebase is harder to change than it should be | `/improve-architecture` |

## The disciplines fire on their own

The model-invoked half of the pack needs no invocation. Their descriptions carry their own triggers,
and the skills above pull them in as they work — `/implement` drives `tdd`, `go-feature` and
`code-review`; `/grill-me` drives `grilling`; every commit message goes through `writing-style`.

`go-feature` · `go-middleware` · `tdd` · `code-review` · `diagnosing-bugs` · `grilling` ·
`domain-modeling` · `codebase-design` · `research` · `resolving-merge-conflicts` ·
`writing-style` · `writing-for-agents`

Naming one still reaches it — "review this branch", "grill this design" — because
model-invocation adds the agent's reach without removing yours.
