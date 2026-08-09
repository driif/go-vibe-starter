---
name: code-review
description: Use when reviewing changes before merge or on request — fans out Go reviewers, verifies every finding at file:line, then fixes confirmed issues by severity.
---

# Code review

Review a branch's changes with a panel of Go reviewers, verify every finding against the real
code, then fix the confirmed ones. This is ralphex's review engine with umputun's Go criteria.

## Steps

1. **Establish context.** Run `git log <base>..HEAD --oneline` and `git diff <base>...HEAD`
   (base = `main`). Do NOT embed diffs into reviewer prompts — reviewers fetch their own git
   context.
2. **Choose the mode.**
   - **Comprehensive** — all 6 reviewers (default for a full review).
   - **Critical-only** — `go-quality` + `go-implementation`, critical/major issues only (fast
     final pass).
3. **Fan out (read-only).**
   - On **Claude Code**: launch the matching subagents in `.claude/agents/go-*.md` **in parallel
     in a single message** (Task tool). Reviewers must not modify the working tree.
   - On other tools: apply each `reviewers/*.md` spec in turn.
4. **Collect + dedup.** Merge findings; same `file:line` + same issue → one entry, noting both
   sources.
5. **Verify EVERY finding (the gate).** Open the actual code at `file:line` (±20–30 lines).
   Classify **CONFIRMED** (fix it) or **FALSE POSITIVE** (discard). Pre-existing issues are in
   scope — don't reject an issue just because it predates the branch.
6. **Fix + validate + commit.** Fix confirmed issues, worst severity first. Run `make test` and
   `make lint` until green. Commit `fix: address code review findings`.
7. **Converge.** After fixing, review again. You're done only when a full pass finds zero new
   issues (your fixes may have introduced new ones).

## Taxonomy

- **Severity:** critical / major / minor. Format `SEVERITY: file:line — description`
  (unmarked ⇒ minor).
- **Over-engineering effort:** trivial / small / medium / large. Large-effort simplifications →
  recommend a follow-up plan rather than fixing in review.

## Reviewers

Specs live in `reviewers/`: `go-quality` (bugs/security/correctness), `go-smells` (style +
conventions + smells), `go-simplification` (over-engineering), `go-testing` (coverage + fake-test
detection), `go-implementation` (meets the plan/spec), `go-documentation` (README/AGENTS.md drift).
