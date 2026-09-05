---
name: resolving-merge-conflicts
description: Resolving an in-progress git merge, rebase or cherry-pick that stopped on conflicts. Use when `git status` reports `both modified` or unmerged paths, when a file carries `<<<<<<<` markers, or when the user says a merge or rebase is stuck.
---

Resolve by **intent**, hunk by hunk. Land the operation; the working tree never goes back to where it
started.

## 1. See the state

```sh
git status                                   # which operation is in progress, which paths are unmerged
git diff --name-only --diff-filter=U         # the conflicted set
git log --oneline --left-right HEAD...MERGE_HEAD   # merge: what each side carries
```

Rebasing inverts the sides — "ours" is the upstream commit being replayed onto, "theirs" is your own
commit. Confirm which is which before reading a single hunk. `git rebase --show-current-patch` names
the commit currently being applied.

## 2. Find the primary source of each side's intent

The diff shows what changed; the intent lives elsewhere. For each side of a hunk:

```sh
git log -p --follow -- <path>      # the commits that wrote these lines, with their messages
gh pr view <n>                     # the PR that shipped it
gh issue view <n>                  # the ticket or spec it closed
```

Read the commit body for the *why*. A hunk you cannot explain in one sentence per side is not ready
to resolve.

## 3. Resolve each hunk

- Preserve **both** intents where they compose — two independent additions usually both belong.
- Where they genuinely conflict, keep the side matching the merge's stated goal and record the
  trade-off in the merge or rebase commit body, naming the intent you dropped.
- Keep the resolution to code one side already wrote. If making the merge correct needs behaviour
  neither side has, land the merge first and make that a separate commit.
- Generated files resolve by regeneration, not by hand: take either side of `internal/api/*.gen.go`
  or `internal/db/gen/*.go`, then `make gen` and let `git diff --exit-code` prove no `drift`. Same
  shape for the skills mirror — resolve the canonical `.claude/skills/` copy, then `make sync-skills`
  to rebuild `agents/skills/`. For `go.sum`, resolve `go.mod` and run `go mod tidy`.

Always resolve. Reaching for `git merge --abort` or `git rebase --abort` throws away the reading you
just did; if one side is truly unusable, resolve to the goal-matching side and file an issue for the
part you dropped.

## 4. Verify

`make test` and `make lint` both exit 0, and `gofmt -l .` prints nothing. Expect breakage that no
single hunk shows: a clean per-hunk resolution still misses a call site the other side renamed, or a
handler whose route moved to a `register*` file in the `hub`. Fix what the merge broke before
finishing it.

## 5. Finish

Stage the resolved paths and complete the operation — `git rebase --continue` until every commit is
replayed, or commit the merge. The message follows `writing-style`: why this merge, and which intent
won where the two could not both stand.
