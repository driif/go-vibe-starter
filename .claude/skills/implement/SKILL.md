---
name: implement
disable-model-invocation: true
description: Build one ticket end to end with you in the loop, then close it.
---

The interactive counterpart to ralphex. `/to-plan` plus `ralphex docs/plans/<file>.md` runs a plan
unattended; this skill builds **one ticket** with the user present, stopping to ask whenever the
ticket does not decide something. Reach for it when the work is small enough not to need a plan
file, or when you want to watch it happen.

Scope is the ticket. Everything you notice beyond it becomes a new issue, not a bigger diff.

## 1. Read the ticket

```bash
gh issue view <N> --json number,title,body,labels --jq '.title, .body'
```

Take the acceptance criteria verbatim — they are the completion criterion for this whole run — plus
the `Blocked by #N` edges and the validation commands. If a blocker is still open, say so and stop.

If the ticket is too fuzzy to turn into failing tests, run `grilling` on it, fold the answers back
into the issue body with `gh issue edit`, and continue from the sharpened version.

## 2. Restate the plan and get a nod

In under ten lines: the acceptance criteria as you read them, the files you expect to touch, and the
order. This is the cheapest moment to correct a misreading. Wait for the go-ahead.

## 3. Build it **red-green**

`tdd` owns the loop: failing test first, minimum code to pass, then refactor. `go-feature` owns the
shape of the work — the slice from migration through sqlc query, OpenAPI, handler, route and test,
with its own reference files for each step. Invoke both; they carry the detail this skill does not
restate.

Between steps, run the narrow command rather than the whole suite:
`go test -race ./internal/api/handlers/`. Speed here is what keeps the loop tight.

Surface each decision the ticket left open — a column type, a status code, an error shape — as a
one-line question with your recommended answer, and keep building once it is answered.

## 4. Validate

```bash
make test    # go test -race ./...
make lint
make gen && git diff --exit-code   # only if the work touched generated sources
```

Both of the first two exit 0 before you go on. The third catches **drift** when the ticket touched
`oapi/openapi.yaml`, `sql/queries/` or `migrations/`.

## 5. Review

Run `code-review` over the diff. Every finding is re-read at `file:line` and marked **confirmed** or
false positive before anything is edited. Fix the confirmed ones, then run `make test` and
`make lint` again. Report the false positives with the reason each was dismissed rather than
dropping them silently.

## 6. Finish

Commit on the working branch, message written under `writing-style`. Then close the ticket with a
comment that points at the change:

```bash
gh issue close <N> --comment "Implemented in <sha or PR link>. <one line on what now works.>"
```

Anything you found outside the ticket's scope goes out as its own issue first:

```bash
gh issue create --label ticket --title "..." --body "... Blocked by #<N>"
```

Report back: the acceptance criteria and how each one is met, the files changed, the review findings
confirmed and fixed, and any issue you opened along the way.
