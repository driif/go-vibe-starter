---
name: grilling
description: The interview primitive — stress-test a plan, design or decision until nothing is silently assumed. Use when the user says "grill me" or "grill this", when a request is too fuzzy to implement, when an issue carries the `needs-grilling` label, or when another skill needs an interview before it can proceed.
---

Interview the user relentlessly until you reach a shared understanding. Map the topic as a **design
tree**: every decision branches into the decisions that hang off it.

Work the tree in **rounds**. The **frontier** is every decision whose prerequisites are already
settled — the questions you can ask *now* without guessing at answers you have not heard yet. Ask
the whole frontier in one numbered round, each question carrying your recommended answer, then wait
for the user before the next round.

## Question format

```
❓ **Q1** - **<question title>**: <body, possibly several paragraphs, possibly lettered choices>

➡️ <your recommended answer, plus the one-line reason for it>
```

Every question gets a recommendation. Do the deciding work, put the choice to the user, let them
overrule you.

## Rounds

Each round of answers reshapes the tree: settled decisions push the frontier outward and unblock the
questions that depended on them. Recompute the frontier and ask the next round. A question whose
answer depends on another question still open in this round belongs to a *later* round.

## Facts are yours, decisions are theirs

Finding facts is your job, never the user's. Read before you ask: `AGENTS.md` for conventions and
commands, `CONTEXT.md` for the domain language, `docs/adr/` for decisions already made,
`oapi/openapi.yaml` and `migrations/` for the current contract and schema, `.env.example` for
configuration, `gh issue view <n>` for the issue under discussion, `rg` over `internal/` for how the
code behaves today.

Run expensive lookups without stalling the round. On Claude, dispatch a read-only subagent and keep
asking; on other agents run the lookup inline. Either way a running lookup is an unsettled
prerequisite, so only the questions downstream of it wait — ask the rest of the frontier now.

The *decisions* are the user's. Put each one to them and wait.

## Frontier seeds for backend work

Openers that reliably grow a tree here. Take the ones the topic touches and drop the rest.

- **Shape of the slice** — which links of the slice the change walks (migration → sqlc query →
  OpenAPI → handler → route → test), and whether it is thin enough to be one **tracer bullet**.
- **Ownership** — which **principal** may read or change the thing: owner-scoped by subject like the
  `DEMO` notes slice, role-gated, or unauthenticated.
- **Storage** — columns, keys, indexes, what is nullable, what is unique, what cascades on delete.
- **API contract** — path and verbs, request and response schemas, which inputs are rejected and
  with which status code.
- **Seams** — what has to swap later, and what the boundary looks like (an `AUTH_PROVIDER`-style
  provider switch, an interface defined at the consumer).
- **Failure and volume** — expected size, behaviour when the database is unreachable, what retries.

Seeds open branches; the real tree grows out of the answers.

## Done

The session ends when the frontier is empty: every branch visited, nothing silently assumed.
Summarise the shared understanding in a few lines and wait for the user's confirmation before acting
on any of it. The usual next step is `/to-spec`, which turns this conversation into a spec issue
without re-interviewing.
