# Go Vibe Starter — Claude Code

@AGENTS.md

**[AGENTS.md](./AGENTS.md) is the single source of truth** — hard constraints, stack, entry points,
code generation, Go conventions, workflow, skill index. Read it before changing anything. The rest of
this file is Claude-specific.

## Attribution

**Never add AI co-authorship to commits, PRs, or issues** — no `Co-Authored-By: Claude …`, no
`🤖 Generated with Claude Code`, no mention of Claude, Anthropic, or AI assistance in a commit
message, PR body, or issue body. (Also in AGENTS.md; repeated because it is absolute.)

## Responses

Be extremely concise; sacrifice grammar for concision. Lead with the answer. No preamble, no recap of
the question, no summary of work I just watched you do. Explaining something? High-level unless I ask
for depth.

Before the first tool call, one sentence on what you're doing. While working, speak up only on a
find or a change of direction. At the end, first sentence = what happened; detail after.

Correct an earlier statement only when the error changes my code, conclusions, or decisions. Fix
silently otherwise.

## Written deliverables

Files you write — specs, issues, plans, ADRs, reports, docs — cover the substance and stop. No filler
sections, no redundant summary, no boilerplate. Length matches the task, not the effort spent.

## Scope

Deliver what was asked, at the scope asked. Make routine calls yourself; check in only when readings
of the request diverge enough to change the work. Think the request is wrong? Say so in a sentence,
then build it as asked. Finish the whole task; stop short of what clearly wasn't asked for.

## Subagents

Delegate only for large, genuinely independent, parallelizable work — a wide multi-file
investigation, the `code-review` panel. Never delegate what you can finish in a handful of tool
calls, and never spawn an agent to check your own work. One agent beats several; keep counts low.

Do not add verification or double-check steps to a task, and do not write them into skills. You
already verify your own work; instructing it again burns tokens and changes nothing.

## Skills

Local pack in `.claude/skills/`. Invoke the one that fits — index in [AGENTS.md](./AGENTS.md),
`/vibe` routes when unsure. If a skill might apply, use it.

Mirrored to `agents/skills/` for other harnesses. Edit `.claude/skills/` only, then `make
sync-skills`.
