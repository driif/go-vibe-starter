# Go Vibe Starter — Claude Code

@AGENTS.md

**[AGENTS.md](./AGENTS.md) is the single source of truth** — hard constraints, tech stack, entry
points, code generation, Go conventions, the development workflow, and the skill index. Read it
before making changes. The rest of this file is Claude-specific.

## How to talk to me

When reporting information, be extremely concise and sacrifice grammar for the sake of concision.
Lead with the answer. Skip preamble, skip recaps of what I just asked, skip closing summaries of
work I watched you do.

## Attribution

**Never add AI co-authorship to commits, PRs, or issues** — no `Co-Authored-By: Claude …`, no
`🤖 Generated with Claude Code`, no mention of Claude, Anthropic, or AI assistance anywhere in a
commit message, PR body, or issue body. (Also stated in AGENTS.md; repeated here because it is
absolute and must never be one import away.)

## Skills

This repo ships a local skill pack in `.claude/skills/`. Invoke the one that matches the task — the
index lives in [AGENTS.md](./AGENTS.md), and `/vibe` routes you to the right one when you are not
sure. If there is any chance a skill applies, use it.

Skills are mirrored to `agents/skills/` for other harnesses. Edit `.claude/skills/` only, then run
`make sync-skills`.
