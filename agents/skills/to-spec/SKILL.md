---
name: to-spec
disable-model-invocation: true
description: Turn what this conversation has already settled into a spec issue on GitHub.
---

Synthesis, not discovery. What this spec needs is already in the conversation and in the codebase —
write it down and publish it as a `spec` issue.

**Run no interview.** Answer every open question from the conversation, from `CONTEXT.md`, from
`docs/adr/`, or from the code itself; look things up rather than asking. That constraint is what
makes this skill cheap to reach for at the end of a design conversation. When a question is genuinely
unanswerable from what you have, it belongs in **Open questions** in the body — the spec ships with
the hole named, and `/triage` or `/grill-me` fills it later.

Prerequisite: `/setup-project` (the `spec` label and the issue forms come from there).

## 1. Collect what is settled

Re-read the conversation for the decisions already made. Then read the code the change lands in —
the handlers in `internal/api/handlers`, the routes in `internal/api/router`, the schema in
`oapi/openapi.yaml`, the queries in `sql/queries`. Write in the repo's own vocabulary: the terms in
`CONTEXT.md`, and the decisions already recorded in `docs/adr/` are constraints, not options.

Done when you can name, without asking the user anything: what hurts today, what is true once this
ships, and where the boundary is.

## 2. Name the seams

Say which **seam** the change moves through and how it gets tested there — an existing seam beats a
new one, and the highest seam beats a lower one. In this repo the usual answer is the HTTP boundary
exercised with `httptest` against the router, with the `principal` supplied by the test helpers in
`internal/server/test/`. A new seam is a decision: state it in the spec and flag it in the draft.

## 3. Draft the body

Four sections, matching `.github/ISSUE_TEMPLATE/spec.yml` so a hand-filed spec and a generated one
read the same:

```markdown
## Problem

What hurts today, from the caller's perspective, and why it is worth fixing now.

## Desired outcome

What is observably true once this ships. Behaviour, contracts, and the shape of the API surface.

## Out of scope

The boundary. Adjacent things this deliberately does not cover, so a ticket cannot drift into them.

## Open questions

One line each, or "None." Link `decision` issues where they exist.
```

Write behaviour, interfaces and contracts. Leave out file paths and code snippets — they go stale
while the spec sits open. One exception: a schema, type shape or state machine that encodes a
decision more precisely than prose, trimmed to the decision-rich part.

## 4. Get approval

Show the draft in full, plus the seam from step 2 and the label you intend to apply. Revise until
the user accepts it. Publish nothing before that.

## 5. Publish

A quoted heredoc delimiter keeps backticks, `$` and `#` literal:

```sh
cat > /tmp/spec-<slug>.md <<'BODY'
## Problem
...
BODY

gh issue create --title "[spec] <short title>" --label spec --body-file /tmp/spec-<slug>.md
```

Then one state label, matching what the body says:

```sh
gh issue edit <N> --add-label ready            # Open questions: None
gh issue edit <N> --add-label needs-grilling   # open questions gate the work
```

A question that is a real fork in the road — two designs, both viable — goes to `/wayfinder` as a
`decision` issue, and this spec carries a `Blocked by #<decision>` line plus the `blocked` label.

## 6. Report

Print the issue URL, the state label applied, and the next move:

- `ready` → `/to-tickets <N>` cuts it into tracer bullets.
- `needs-grilling` → `/grill-me` on the open questions, then `/triage` moves it to `ready`.

Leave the spec issue open. It closes when its tickets close.
