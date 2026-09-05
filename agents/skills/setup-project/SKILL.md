---
name: setup-project
disable-model-invocation: true
description: Run once in a repo created from this template — check gh, create the issue labels, seed CONTEXT.md, confirm the layout.
---

One pass over a fresh repo so the planning flow has the substrate it assumes: an authenticated `gh`
pointed at a real repository, the label vocabulary, a renamed module, a `CONTEXT.md` that says
something, and the docs directories. `/to-spec`, `/to-tickets` and `/triage` all name this skill as
their prerequisite.

Every step is idempotent — on an already-set-up repo this reports and changes nothing. Work the
steps in order; each step's check is its completion criterion. A failed check stops the run and
hands the user the one command that fixes it, because the later steps write to real issues.

## 1. GitHub is reachable

```sh
gh auth status
gh repo view --json nameWithOwner -q .nameWithOwner
```

Done when both exit 0 and the second prints `owner/repo`.

- Auth fails → the user runs `gh auth login`.
- No repository detected → the clone has no GitHub remote yet. The user runs `gh repo create` (or
  adds the remote), then re-runs this skill.

## 2. The module path is this project's

```sh
awk '$1 == "module" { print $2 }' go.mod
```

Done when it prints anything other than `github.com/driif/go-vibe-starter`.

Still the template path → `make init` has not run. It renames the module, rewrites the imports,
strips the `DEMO` slice, and resets `README.md`, `CONTEXT.md`, `docs/plans/` and `docs/adr/`.
Have the user run it, then resume at step 3 — `make init` undoes step 4's work if it runs after.

## 3. The label vocabulary exists

```sh
make labels
gh label list --limit 100 --json name -q '.[].name'
```

Done when the listing carries all nine: `spec` `ticket` `decision` `bug` `needs-info`
`needs-grilling` `ready` `blocked` `wontfix`. `make labels` uses `gh label create --force`, so a
second run only refreshes colours and descriptions.

## 4. CONTEXT.md says something

```sh
grep -n '^| | |' CONTEXT.md
```

A match means the ubiquitous-language table is still the empty stub `make init` wrote. Fill it:

1. Read the code for the terms this project already uses — the domain packages under `internal/`,
   the schemas in `oapi/openapi.yaml`, the tables in `migrations/`, the queries in `sql/queries/`.
   Add the terms the repo itself supplies: `principal`, `the slice`, `hub`, and any `DEMO` naming
   that survived.
2. Propose the rows to the user as a table, one term and one meaning per row, and let them cut,
   rename and correct before anything is written. The words here become the words every later spec,
   ticket and handler uses, so the user owns them.
3. Write the agreed rows into the **Ubiquitous language** table. Leave the **Module map** as it is
   unless packages have been added or removed since `make init`.

Done when the table holds at least the terms a stranger would have to ask about, and the stub row
is gone. No match on the `grep` means someone already filled it — read it, say what is there, and
offer to extend rather than overwrite.

## 5. The layout is intact

```sh
ls -d docs/plans docs/plans/completed docs/adr .github/ISSUE_TEMPLATE
```

Done when all four exist. Recreate any that do not:

```sh
mkdir -p docs/plans/completed docs/adr
touch docs/plans/.gitkeep docs/plans/completed/.gitkeep docs/adr/.gitkeep
```

`.github/ISSUE_TEMPLATE/` missing means the template's forms were deleted — say so and stop; the
issue forms are what keep hand-filed issues in the same shape the skills write.

## 6. Report

Print one line per step with what it found or changed, then the next move:

- A feature already discussed in this conversation → `/to-spec`.
- Nothing discussed yet → `/grill-me` first, then `/to-spec`.

Leave every change in the working tree, uncommitted.
