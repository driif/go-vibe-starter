# Design it twice

The long form of the discipline in [`SKILL.md`](SKILL.md): explore several interfaces for the same
behaviour before committing to one. Reach for this when the surface is load-bearing — a package
several handlers will call, a seam that will outlive the ticket — rather than for every function.

## 1. Frame the problem space

Write the frame for the user before exploring anything:

- The behaviour the module must provide, and the constraints any interface has to satisfy.
- Its dependencies and their category (see [`DEEPENING.md`](DEEPENING.md)) — that decides which
  designs are even testable.
- A rough call-site sketch in Go to make the constraints concrete. A sketch, not a proposal.

Show it, then start exploring immediately. The user reads while the work runs.

## 2. Explore three interfaces

Each exploration produces a **radically different** exported surface, not three spellings of one.
Give each a different constraint:

- **Minimal** — 1–3 entry points, maximum leverage per entry point.
- **Common caller** — make the most frequent call trivial, even if the rare one gets verbose.
- **Seam elsewhere** — move the seam: push it to the consumer, up to the router, down to `gen`.

On Claude, dispatch these as parallel subagents — one brief each, and briefs carry the vocabulary
from [`SKILL.md`](SKILL.md) plus the domain terms from `CONTEXT.md` so the three designs name things
the same way. Elsewhere, work them sequentially in one session, writing each design out fully before
starting the next so the second does not collapse into a variation of the first.

Every brief states the files involved, the current coupling, the dependency category, and what
should end up hidden behind the seam.

Each design comes back as:

1. The exported surface — signatures, plus invariants, error modes, context effects.
2. A call site in Go showing how a handler in `internal/api/handlers` would use it.
3. What the implementation hides.
4. Its adapters, and how a test reaches it.
5. Where its leverage is thin.

## 3. Compare and recommend

Present the designs one at a time so each is absorbed on its own, then compare them in prose on
**depth**, **locality** and **seam placement** — the three axes, named with those words.

Close with your own pick and the reason, or a hybrid if one design's seam belongs under another's
surface. Be opinionated: the user wants a read, not a menu. If the pick becomes work, it goes to
`/to-spec` and `/to-tickets` like anything else.
