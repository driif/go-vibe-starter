---
name: handoff
disable-model-invocation: true
description: Compact this session into a handoff another session can start from.
---

Write a handoff dense enough that a fresh session reaches the same understanding without re-deriving
it, and short enough to read in a minute. The reader is an agent with an empty context window and
access to this repo.

If the user passed an argument, treat it as what the next session will focus on and slant the whole
document toward that.

## Where it goes

```bash
$TMPDIR/go-vibe-starter-handoff-$(date +%Y%m%d-%H%M).md
```

Outside the repo: `docs/plans/` is ralphex's execution directory, and a handoff sitting there shows
up as a runnable plan. Print the path when you are done, and paste the document into the
conversation as well so it survives even if the file is lost.

## What it holds

Reference by number and path — `#41`, `docs/plans/20260809-projects-api.md`,
`docs/adr/0003-auth-provider.md`, `internal/api/handlers/projects.go:88`. The next session can open
any of them. Restating their contents is what makes handoffs long and stale at the same time.

- **Goal** — one sentence on what the work is for.
- **Current state** — done, in progress, untouched. Name the last command run and its result
  (`make test` green, `make lint` clean, three handler tests failing).
- **Key decisions** — each one line, each with its why. Decisions already in an ADR or a `decision`
  issue get a pointer instead.
- **Next action** — the single exact thing to do next, specific enough to start on without a choice:
  the file, the function, the command.
- **Open questions** — what is unresolved and what it blocks. Mark the ones only the user settles.
- **Pointers** — issues, plan file, ADRs, the files in play, the branch, any env var that had to be
  set.
- **Suggested skills** — the skills the next session should reach for, in order, one line each on
  why. User-invoked skills need naming here because nothing but a human invokes them.

## How it reads

`writing-style` governs the prose: direct, specific, no filler. Fragments beat sentences when they
carry the same information. Redact secrets — tokens, passwords, connection strings with credentials,
anything out of `.env` — and write the variable name instead of its value.

The last thing to check is the "next action": if a fresh session could read it and still not know
which file to open, sharpen it before you hand off.
