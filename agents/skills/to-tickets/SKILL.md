---
name: to-tickets
disable-model-invocation: true
description: Break a spec issue into tracer-bullet ticket issues with their blocking edges.
---

Cut a spec into **tracer bullets**: tickets thin enough that each one walks the whole slice
end to end and leaves the system working. Publish them as `ticket` issues carrying their blocking
edges.

Prerequisite: `/setup-project`. Input: a spec issue number, or the spec sitting in this conversation.

## 1. Read the spec

```sh
gh issue view <N> --comments
```

The comments carry the grilling that happened after the body was written, so read both. Then read the
code the spec lands in, and use `CONTEXT.md` vocabulary for every title and body — a ticket titled in
the domain's own words survives the weeks it may sit open.

## 2. Cut tracer bullets

**The slice is the unit.** One good ticket in this repo is usually one `go-feature` vertical slice:
migration → sqlc query → OpenAPI → handler → route → test. It ends with `make test` and `make lint`
green and a working endpoint. `notes` in the `DEMO` slice is the worked example of that shape.

- **Vertical, never horizontal.** "Notes CRUD, create only" is a ticket. "All the migrations" is a
  layer, and a layer leaves nothing demonstrable at the end of it.
- **One fresh context window each.** If a ticket cannot be held in one session, split it by
  behaviour: create before list, list before delete.
- **Prefactor first.** Work that makes the change easy — extracting a `deep module`, opening a
  `seam` — is its own ticket, and it blocks the tickets it serves.
- **Blocking edges are real gates.** A ticket blocks another when the second cannot compile, run or
  be tested until the first lands. Sharing a file is not a gate; the `hub` takes one
  `register<Domain>(s)` line per domain precisely so two slices never collide there.
- A mechanical change whose blast radius fans across the codebase resists vertical slicing. Sequence
  it expand–contract instead: [`WIDE-REFACTOR.md`](WIDE-REFACTOR.md).

## 3. Get approval on the breakdown

Present a numbered list before publishing anything. Per ticket: **title**, **blocked by** (ticket
numbers from this list, or "nothing"), and **what it delivers** in one line of observable behaviour.
Then ask three questions: is the granularity right, is every edge a genuine gate, and does anything
want merging or splitting. Iterate until the user accepts the list.

## 4. Publish in dependency order

Blockers first, so every `Blocked by #N` names a real issue number. Body sections match
`.github/ISSUE_TEMPLATE/ticket.yml`:

```sh
cat > /tmp/ticket-01.md <<'BODY'
## What this slice delivers

POST /v1/notes persists a note owned by the calling principal and returns 201 with the created note.

## Acceptance criteria

- [ ] `POST /v1/notes` with a valid body returns 201 and the created note
- [ ] `owner_id` is taken from the principal, never from the request body
- [ ] a malformed body returns 400 through `errs.Write`
- [ ] `notes_test.go` covers the success case and both failure cases

## Blocked by

Blocked by #12

## Validation commands

make test
make lint
BODY

gh issue create --title "[ticket] create a note" --label ticket --body-file /tmp/ticket-01.md
```

Write "Blocked by" as one `Blocked by #N` line per blocker so GitHub renders the cross-reference, or
`None — can start immediately`. Acceptance criteria are checkable statements about behaviour, not
implementation steps; keep file paths out of the body and let the agent find the code.

## 5. Label the frontier

The **frontier** is every ticket whose blockers are all closed — the work that can start now.

```sh
gh issue edit <N> --add-label ready      # nothing blocks it
gh issue edit <N> --add-label blocked    # a Blocked by line names an open issue
```

`/triage` walks these forward as blockers close.

## 6. Index the spec

Comment the breakdown onto the spec issue so it stays the map:

```sh
gh issue comment <spec> --body-file /tmp/ticket-index.md
```

One line per ticket: `#<N> — <title>` and its blockers. Leave the spec open and unedited otherwise.

## 7. Report

Print the ticket URLs in dependency order, mark the frontier, and name the next move: `/to-plan
<N>` for an unattended ralphex run, or `/implement <N>` to build one interactively.
