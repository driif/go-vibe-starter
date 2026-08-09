---
name: make-plan
description: Use when you have a spec, an approved design, or a clear change to build — explores context, proposes approaches, then writes an approved ralphex-format plan file with tasks, checkboxes, and validation commands.
---

# Make plan

Turn a spec or a clear change into an approved, executable plan file that `execute-plan` (or the
ralphex loop) can drive task by task.

## Flow

1. **Explore context.** Read the relevant files, existing patterns, and dependencies. Look up
   facts — don't ask what you can find. See `AGENTS.md` for conventions.
2. **Propose 2–3 approaches** with trade-offs; lead with your recommendation and why. Skip this
   when the path is obvious.
3. **Draft, then get explicit approval before writing the file.** Present the draft plan; accept,
   revise, or reject. Do not write the file until the user approves.
4. **Write** `docs/plans/YYYYMMDD-<slug>.md` using the template below.
5. **Self-check before finalizing** — Scope & feasibility, Completeness, and **Simplicity
   (YAGNI)**: no future-proofing that wasn't requested; no backwards-compat or fallbacks unless
   asked.

## Plan template

```
# <Title>

## Overview
<what will be implemented>

## Context
- Files involved: <list>
- Related patterns: <existing patterns to follow>
- Dependencies: <external deps if any>

## Development Approach
- Testing approach: Regular (code then tests) or TDD (test first)
- Complete each task fully before the next
- Every task MUST include new/updated tests
- All tests must pass before starting the next task

## Validation Commands
- Test: `make test`
- Lint: `make lint`

## Implementation Steps

### Task 1: <title>
**Files:**
- Modify: `path`
- Create: `path` (if any)
- [ ] first step
- [ ] write tests for this task
- [ ] run make test + make lint — must pass before Task 2

### Task N: Verify acceptance criteria
- [ ] full test suite passes (make test)
- [ ] linter passes (make lint)

### Task N+1: Update documentation
- [ ] update README.md if user-facing changes
- [ ] update AGENTS.md if internal patterns/conventions changed
```

## Rules

- Checkboxes live **only** inside `### Task N:` sections — never in Overview, Context, or
  acceptance prose, or the loop runs extra iterations.
- `### Task N:` is a structural token; keep it in English even if the body is another language.
- Size tasks at 3–7 items, one component each, with linear dependencies.
- Put manual / deploy / external-verification items in a `## Post-Completion` note (no checkboxes),
  not as `[ ]` inside a Task section.
