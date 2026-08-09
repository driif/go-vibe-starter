---
name: execute-plan
description: Use when executing a plan file from docs/plans — completes ONE task per pass (implement, test, validate, commit), then stops, so it composes with any loop.
---

# Execute plan

Drive a plan file from `docs/plans/` one task at a time. Completing exactly one task per pass
keeps context small and lets any outer loop (manual re-invocation, the `ralph-loop` plugin, or the
ralphex binary) call you again for the next task.

## Per-pass steps

1. **Find the task.** Read the plan. Pick the FIRST `### Task N:` section that still has `[ ]`
   checkboxes.
2. **Announce** (≤200 words): which task, what it accomplishes, the key files involved.
3. **Implement** every item in that ONE section. Write or update tests as you go (see the `tdd`
   skill). Follow `AGENTS.md` conventions.
4. **Validate.** Run the plan's Validation Commands (`make test`, `make lint`). Fix until green.
5. **Complete.** Flip that section's `[ ]` → `[x]`. Commit code + updated plan together:
   `feat: <task description>`.
6. **Stop or finish.** If more `[ ]` remain anywhere, STOP — let the next pass handle them. If no
   `[ ]` remain, move the plan to `docs/plans/completed/` and report done.

## Rules

- One task section per pass. Never roll into the next section.
- Non-automatable items inside a Task (manual test, deployment): mark `[x]` with a short skip note
  (e.g. `[x] manual smoke test (skipped — not automatable)`). Never loop forever on them.
- If a task can't be completed after reasonable attempts, stop and report the blocker — don't
  guess past it.
