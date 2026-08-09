---
name: triage
disable-model-invocation: true
description: Move GitHub issues through the label state machine — pick a state, do what it asks, then hand the issue on.
---

Every open issue sits in exactly one **state**, named by one label. Triage reads the issue, decides
which state it belongs in, and performs that state's exit action so it can move on.

Two label families, and they are independent. **Category** — `spec`, `ticket`, `decision`, `bug` —
comes from the issue form and never changes. **State** — `needs-info`, `needs-grilling`, `ready`,
`blocked`, `wontfix` — is what this skill moves. An issue carrying two state labels is a bug in the
board: say so and ask before touching it.

Prerequisite: `/setup-project`.

## The state machine

| State | Entry condition | Exit action |
|---|---|---|
| *(none)* | Freshly filed, no state label yet | Read it, then apply exactly one of the states below |
| `needs-info` | The claim cannot be checked: no repro, no version, no error text, or the ask is ambiguous about who wants what | Post the specific questions. When the reporter answers, re-triage into another state |
| `needs-grilling` | The ask is understood but underspecified — the outcome, the boundary or the acceptance criteria are missing | Run `/grill-me`, fold the answers into the issue body, then move to `ready` |
| `ready` | Outcome, acceptance criteria and validation commands are all present and nothing open gates it — an agent can start now | `/to-tickets` for a spec, `/to-plan` or `/implement` for a ticket |
| `blocked` | The body carries a `Blocked by #N` naming an issue that is still open | When every named issue closes, swap `blocked` for `ready` |
| `wontfix` | Deliberately not doing it, or it already exists in the code | Comment the reason, then close as not planned |

Transitions worth naming: `needs-info` → `needs-grilling` is common (the reporter answers, and the
answers turn out to be fuzzy). `ready` → `blocked` happens when a new dependency is discovered
mid-flight. Anything can go to `wontfix`. Nothing goes to `ready` without acceptance criteria in the
body — that is the one gate this skill holds.

## Show what needs attention

```sh
gh issue list --state open --search "-label:ready -label:needs-info -label:needs-grilling -label:blocked -label:wontfix"
gh issue list --state open --label needs-info
gh issue list --state open --label ticket --label ready
gh issue list --state open --label blocked
```

Present the four buckets — untriaged, waiting on the reporter, the ready queue, blocked — oldest
first, one line each: number, title, category, age. Let the user pick, or take the issue number they
named.

## Triage one issue

```sh
gh issue view <N> --comments
```

1. **Read everything**: body, comments, labels, and any prior triage notes, so a question already
   answered is not asked twice.
2. **Check the code before believing the issue.** Search for the behaviour by domain concept, using
   `CONTEXT.md` vocabulary rather than the reporter's wording. Already implemented is a `wontfix`.
   For a `bug`, reproduce it — a `confirmed` repro, quoted with the failing command and its output,
   is worth more than every other line of triage.
3. **Recommend a state** with the reasoning and what you found in the code. Wait for the user.
4. **Apply it**, with the exit action from the table.

## Applying a state

```sh
gh issue edit <N> --add-label ready --remove-label needs-grilling
gh issue comment <N> --body-file /tmp/triage-<N>.md
gh issue close <N> --reason "not planned" --comment "<why, and where it already lives>"
```

Swap the old state label out in the same `gh issue edit` call, so the issue never briefly carries
two.

`needs-info` comment shape — what is established goes in first, so grilling already done survives:

```markdown
## Triage notes

**Established so far**

- ...

**Still needed from @<reporter>**

- ...
```

Every question specific and answerable. "Please provide more info" gets no reply worth having.

## Unblocking sweep

For each open `blocked` issue, read its edges and check them:

```sh
gh issue view <N> --json body -q .body | grep -o 'Blocked by #[0-9]*'
gh issue view <M> --json state -q .state
```

Every named issue `CLOSED` → `gh issue edit <N> --add-label ready --remove-label blocked`. Report the
newly-unblocked frontier at the end of the sweep; that is usually the most useful output of a triage
session.

## What closes an issue

- **`ticket`** — the change is merged and `make test` and `make lint` both exit 0. Close with a
  comment naming the commit or PR:
  `gh issue close <N> --comment "Done in <sha>; make test and make lint green."`
- **`spec`** — every ticket it spawned is closed. Close with a comment listing them.
- **`decision`** — the answer is recorded in `docs/adr/`. Close with a comment linking the ADR.
- **`bug`** — a regression test covers it and passes. Close referencing the test.
- **`wontfix`** — closed as not planned, with a reason durable enough to answer the same request a
  year from now.
