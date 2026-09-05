---
name: code-review
description: Reviewing Go changes in this repo before they land. Use when asked to review a branch, a PR, a diff or work in progress, when an implementation is finished and needs checking, or when another skill hands finished code back for review.
---

Review runs on **two axes**, dispatched as read-only agents that never see each other's context, over
a panel of six Go reviewer briefs in [`reviewers/`](reviewers/). Nothing they report is acted on
until it has been re-read at `file:line` and marked `confirmed`.

## The two axes

| Axis | Judged against | Briefs |
|---|---|---|
| **Standards** | `AGENTS.md`, `docs/agents/go-style.md`, and the code already in the package | `go-quality`, `go-smells`, `go-simplification`, `go-testing`, `go-documentation` |
| **Spec** | the originating issue, ticket or plan | `go-implementation` |

The axis is the source of truth the reviewer judges against, which is why they run apart. Code can
follow every convention and implement the wrong thing (Standards pass, Spec fail); it can do exactly
what the issue asked while breaking the repo's conventions (Spec pass, Standards fail). Separate
contexts stop one verdict from excusing the other.

## Modes

- **Comprehensive** — all six briefs. The default, and what a branch gets before it lands.
- **Critical-only** — `go-quality` and `go-implementation`, critical and major findings only. A fast
  last pass over a change that has already been reviewed.

## Steps

1. **Pin the base.** Default `main`; take whatever ref the user names. Confirm it resolves
   (`git rev-parse <base>`) and that `git diff <base>...HEAD --stat` is non-empty. Capture
   `git log <base>..HEAD --oneline` for the commit list. A bad ref fails here, not inside six agents.
2. **Find the spec.** In order: an issue number the user gave; a `Closes #N` / `#N` reference in the
   commits or PR body; the issue named by the branch. Fetch it with
   `gh issue view <n> --json title,body,labels`. Failing that, the matching plan in `docs/plans/`.
   With no spec anywhere, the Spec axis reports `no spec available` and the final report says so.
3. **Dispatch, read-only.** Both axes at once.
   - *Claude Code fast path:* the subagents in `.claude/agents/go-*.md` are thin wrappers over these
     same briefs. Launch every one of them **in a single message** so they run in parallel.
   - *Every other harness:* apply each `reviewers/*.md` brief in turn, one agent at a time.
   - Give each reviewer the base ref, the commit list and the spec — not the diff text. Reviewers run
     their own `git diff` so their context holds only what they chose to read.
   - Reviewers report findings and leave the working tree untouched.
4. **Collect and dedup.** Same `file:line` plus the same claim is one finding; note every brief that
   raised it, since agreement is signal for step 5.
5. **The gate — verify every finding.** Open the file at the line and read around it (±25 lines).
   Mark each finding `CONFIRMED` with the evidence that makes it real, or `FALSE POSITIVE` with the
   line that disproves it. A finding whose `file:line` does not exist or does not say what the
   reviewer claimed is a false positive.
   - **Completion criterion:** every finding carries a verdict, and no edit happens before the last
     verdict is written.
   - A pre-existing problem the branch did not introduce is still in scope — verify it and mark it
     `minor` unless the branch made it worse.
   - Findings inside generated files (`internal/api/*.gen.go`, `internal/db/gen/`) are really
     findings about their source (`oapi/openapi.yaml`, `sql/queries/`, `migrations/`) or about
     `drift`. Re-point them there or discard.
6. **Report.** Two headings, `## Standards` and `## Spec`, each listing its confirmed findings worst
   severity first. Do not rerank across the axes — that merge is exactly what the split prevents.
   Close with the false positives, one line each, so the reader sees what was ruled out and why.
7. **Fix, when the user asked for fixes.** Confirmed findings only, worst severity first. Rerun
   `make test` and `make lint` until both exit 0, and leave the result in the working tree. Then run
   the panel again: a fix is itself unreviewed code. You are done when a full pass raises nothing
   new.

## Taxonomy

Every finding is one line plus its detail: `CRITICAL: internal/api/handlers/notes.go:64 — <claim>`.
Unmarked reads as minor.

| Severity | Means |
|---|---|
| `critical` | Data loss, a security hole, or a broken path that ships to users |
| `major` | Wrong behaviour on a real input, a missing requirement, an unhandled error path |
| `minor` | Convention, naming, clarity, a gap that costs a future reader time |

`go-simplification` also grades effort — `trivial` / `small` / `medium` / `large`. A large-effort
simplification becomes a follow-up issue rather than a change made inside the review.

## The panel

| Brief | Looks for |
|---|---|
| [`go-quality`](reviewers/go-quality.md) | Bugs, concurrency, resource leaks, security |
| [`go-implementation`](reviewers/go-implementation.md) | The spec's requirements, wiring, scope creep |
| [`go-testing`](reviewers/go-testing.md) | Coverage of the change, and tests that cannot fail |
| [`go-simplification`](reviewers/go-simplification.md) | Complexity this branch introduced |
| [`go-smells`](reviewers/go-smells.md) | This repo's conventions, style consistency, smells |
| [`go-documentation`](reviewers/go-documentation.md) | Docs, `.env.example` and godoc left behind by the change |
