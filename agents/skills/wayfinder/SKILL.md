---
name: wayfinder
disable-model-invocation: true
description: Chart work too big for one session as decision issues on GitHub, then resolve them one at a time until the route is clear.
---

A loose idea has arrived, too big for one session and wrapped in fog: the way from here to the
**destination** is not visible yet. Wayfinder finds that way. It charts the route as `decision`
issues on this repo's GitHub, then works them one at a time until nothing is left to decide.

## It plans, it does not build

Every issue it creates resolves a question; its answer is a decision, not a slice of build work. The
pull to start building is the signal the map has run out — that is the moment to hand off to
`/to-spec` and `/to-tickets`, not to keep going.

## Refer by name

Issues have titles, so use them: *Pick the outbox delivery guarantee*, not `#42`. A wall of numbers
is illegible where names read at a glance. The number rides inside the name as a link.

## The substrate

GitHub issues, driven with `gh` and labelled `decision` — `/setup-project` gets `gh` authenticated
and the labels created. Two kinds: **the map**, one issue titled `[map] <destination>` holding the
state of the whole effort, and **a decision**, one question carrying `Part of #<map>`. The map is an
index, not a store: each answer lives in its own issue, and the map gists it and links. Its body:

````markdown
## Destination
<what reaching the end looks like — a spec to hand off, a decision to lock, a change made in place.
Two lines. Every session reads this before choosing anything.>

## Notes
<domain context, standing preferences, skills each session should run>

## Decisions so far
- [<decision issue title>](<url>) — <one-line answer>

## Not yet specified
<in-scope fog: questions you can see coming but cannot yet state sharply>

## Out of scope
<ruled beyond the destination; never graduates>
````

A decision issue follows `.github/ISSUE_TEMPLATE/decision.yml`, with the question filled at charting
time and the rest filled by whoever resolves it:

````markdown
## The question
<phrased so a named option answers it>

Type: grilling
Part of #12
Blocked by #14
````

## Types

One `Type:` line per decision issue.

- `research` — a fact the decision waits on, from outside this working directory. Resolved by
  the `research` skill, agent alone. The one type you may resolve several of per
  session.
- `grilling` — a conversation with the user, and the default. Runs the `grilling` and
  `domain-modeling` skills.
- `prototype` — a rough spike on a throwaway branch, when "how should it behave" is the real
  question. Link the branch from the issue; the spike is evidence, not the deliverable.
- `task` — manual work that unblocks a decision: provisioning an account, moving data so its shape
  can be seen. The agent drives it where it can, otherwise hands over a precise checklist. The
  answer records what was done and the facts later decisions depend on.

## Frontier and fog

`Blocked by #N` lines are the edges; an issue is unblocked once every issue it names is closed. The
**frontier** is the open decision issues, minus the map, whose blockers are all closed — what is
takeable now.

Beyond the frontier lies fog, and the map is deliberately incomplete: chart only what you can see.
The test for where a question goes is whether you can state it sharply **now**, not whether you can
answer it now. Sharp enough to state becomes an issue even when blocked; anything looser goes to
**Not yet specified** in coarse patches and graduates later. Work past the destination is not fog at
all — it goes to **Out of scope**, and never graduates.

## Chart the map

1. **Name the destination.** Run `grilling` and `domain-modeling` until two lines say what reaching
   the end looks like. The destination fixes the scope, so it is settled first and shapes every
   issue after it.
2. **Map the frontier.** Grill again, breadth-first — fan across the whole space rather than deep on
   one thread — surfacing the open questions and which are takeable today. No fog at all means the
   way is already clear: say so, skip the map, and point at `/to-spec`.
3. **Create the map issue**: Destination and Notes filled, Decisions so far empty, the fog written
   into Not yet specified.
4. **Create the decision issues** you can state sharply, then wire `Blocked by #N` in a second pass,
   since issues need numbers before they can reference each other.
5. **Run the research issues** now, in parallel — they are agent-alone, and their answers sharpen
   everything else.
6. Stop. Charting is one session's work and resolves nothing else.

## Work the map

1. Load the map issue — the low-resolution view, not every child.
2. Choose one issue: the user's if they named one, otherwise the first on the frontier. Assign it to
   yourself before any work, so a concurrent session skips it.
3. Resolve it, zooming as needed: pull the body of a related or closed issue on demand, run the
   skills the Notes name. In doubt, `grilling` and `domain-modeling`.
4. Record it: post Options considered, Recommendation and the decision as a comment, close the
   issue, append one line to the map's Decisions so far.
5. Move the fog: create the questions the answer just made sharp (create, then wire), clear the
   graduated patch from Not yet specified, and close anything now past the destination with one line
   under Out of scope.

**One `grilling` issue per session** — research issues excepted. Expect other sessions to be editing
the same issues, so re-read the map before you edit it.

The route is clear when the frontier and Not yet specified are both empty. Say so, close the map
issue with a comment linking the decisions, and hand off: `/to-spec` for the spec issue, then
`/to-tickets` for the tracer bullets.
