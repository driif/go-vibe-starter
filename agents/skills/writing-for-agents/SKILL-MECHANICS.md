# Skill mechanics

The skill-specific branch of [`writing-for-agents`](SKILL.md): what changes when the document is a
skill. Frontmatter, the invocation choice, splitting by invocation, router skills. Everything else
about writing it is the universal reference in `SKILL.md`.

## Frontmatter

YAML between `---` fences at the top of `.claude/skills/<name>/SKILL.md`.

| Field | Value |
|---|---|
| `name` | Matches the directory name exactly, kebab-case (`go-feature`, not `Go Feature`). |
| `description` | The skill's top-level context pointer. Model-invoked: trigger branches. User-invoked: one human-facing line. |
| `disable-model-invocation` | `true` on user-invoked skills only. Omit the key entirely otherwise. |

Nothing else. A skill that needs more configuration is a skill trying to be a program.

```yaml
---
name: go-feature
description: Building a feature that touches the database or the HTTP API in this repo. Use when adding an endpoint, a migration, a sqlc query, or a handler, or when a ticket says "add /v1/<thing>".
---
```

```yaml
---
name: to-tickets
disable-model-invocation: true
description: Break a spec issue into tracer-bullet ticket issues.
---
```

Sibling reference files sit in the same directory with no frontmatter and a plain `#` heading, as
this file does. Reach them by a relative link from `SKILL.md`.

## Invocation

Two choices, trading the two loads.

- **Model-invoked** keeps a `description` the agent always holds, so it can fire the skill on its own
  and other skills can reach it. Typing the name still works: model-invocation *includes* user reach,
  it never removes it. The price is permanent context load. A model-invoked skill whose content is
  all reference is also the pack's home for shared reference, since several skills can invoke one
  copy. Mechanics: omit `disable-model-invocation`, and write a model-facing description carrying the
  trigger branches, under the pointer rules in `SKILL.md`.
- **User-invoked** strips the description from the agent's reach: only the human typing the name
  invokes it, and no other skill can. Zero context load, paid for in cognitive load, because the
  human becomes the index that must remember it exists. Mechanics: set
  `disable-model-invocation: true` and make the `description` one human-facing line with the trigger
  list stripped.

Pick model-invocation when the agent must reach the skill on its own, or another skill must. In this
pack that is the disciplines: `go-feature`, `tdd`, `code-review`, `grilling`, `writing-style`. If the
skill only ever fires because a human typed it — the lifecycle commands, `/to-spec`, `/to-plan`,
`/implement` — make it user-invoked and pay no context load.

Shared reference that two user-invoked skills both need can live in neither, since with no
descriptions neither can fire the other. Push it to a plain file outside the skill system:
`docs/agents/*.md`, which any skill can point at.

## Splitting by invocation

The invocation cut of splitting (the sequence cut lives in `SKILL.md`): split off a model-invoked
skill when it has a distinct leading word that should trigger it on its own, a word you actually type
in your prompts, or when another skill must reach it. `grill-me` and `grill-with-docs` are thin
user-invoked wrappers over the model-invoked `grilling` for exactly this reason: the interview
primitive has one home, and two entry points reach it. You pay context load for each new
always-loaded description, so that independent reach has to be worth it.

## Router skills

When user-invoked skills multiply past what a human can remember, that piled-up cognitive load is
cured by a **router skill**: one user-invoked skill naming the others and when to reach for each, so
the human has one name to remember instead of fifteen. `vibe` is this pack's router. It can only
hint, never fire: user-invoked skills have no description, so nothing but the human reaches them.

A router earns its place by staying a map. Keep it to the flow, the on-ramps, and one line per skill,
and let each skill hold its own steps.

## After editing

Run `make sync-skills` so `agents/skills/` matches. The mirror is generated output; the canonical
copy under `.claude/skills/` is the one you edit.
