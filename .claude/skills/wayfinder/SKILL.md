---
name: wayfinder
description: Use when planning work too large for one session — creates a local markdown decision map and resolves it one open question at a time until the path is clear.
---

# Wayfinder

Chart large, foggy work as a shared **decision map** in local markdown, then resolve it one open
question at a time until the route to the destination is clear. Wayfinder produces *decisions*,
not deliverables — work moves to execution (via `make-plan`) only once the path is clear.

## The map

Create one map per initiative at `docs/plans/maps/<human-readable-name>.md`. Use human-readable
names everywhere — never bare IDs. Sections:

- **Destination** — what success looks like (a spec, a decision, or a concrete change).
- **Notes** — domain context and guidance for future sessions.
- **Decisions so far** — closed questions, each with a one-line answer.
- **Open questions (frontier)** — in-scope fog. Each is a named question tagged with a type:
  - `research` — look it up (docs, code, tools).
  - `grilling` — decide with the user via the `grilling` skill.
  - `prototype` — build a rough spike to ground the discussion.
  - `task` — a manual prerequisite that unblocks a decision.
- **Out of scope** — work consciously ruled out.

## Charting (first pass)

Name the destination → map the frontier breadth-first (list the open questions that block it) →
write the map file → stop. Don't answer questions yet.

## Working (each subsequent pass)

Load the map → pick the next open question → resolve it (research / grill / prototype) → record a
one-line answer under **Decisions so far** → update the frontier (remove answered, add newly
revealed questions) → repeat until the frontier is empty, then hand off to `make-plan`.
