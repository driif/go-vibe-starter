# Skills Starter Pack — Design Spec

- **Date:** 2026-07-20
- **Status:** Approved (design) — pending spec review
- **Owner:** driif
- **Topic:** A local, portable skills pack for go-vibe-starter that fuses Matt Pocock's
  planning/interview approach, the ralphex agentic loop + multi-agent review, and umputun's
  Go review criteria — tailored to this repo's actual stack.

---

## 1. Goal & context

Give this repository a self-contained set of **skills** (instruction files) that make an AI
coding agent plan, implement, review, and ship changes the way this project wants — without
adding an external dependency. Skills are **copy-pasted and adapted**, not vendored as a package.

Primary target: **Claude Code**. Secondary: **Codex, Pi, opencode** (any AGENTS.md-aware tool).

The pack encodes one lifecycle and one set of Go conventions, so that a fresh agent session in
this repo behaves consistently regardless of who invokes it.

## 2. Non-goals

- Not a fork or dependency of `mattpocock/skills`, `umputun/cc-thingz`, or `umputun/ralphex`.
- Not a reimplementation of the ralphex *binary* loop. The repo already has ralphex available;
  these skills are usable **directly by the agent** (manual or via the `ralph-loop` plugin) and
  remain compatible with the ralphex binary's own prompts.
- Not duplicating skills the environment already provides well (superpowers `brainstorming`,
  `skill-creator`). AGENTS.md points to those rather than copying them.
- No issue-tracker integration. All planning artifacts are **local markdown**.

## 3. Source lineage

What we take from each source (criteria/format kept; foreign libraries discarded):

- **Matt Pocock (`mattpocock/skills`)** — the relentless *interview* front-end (`grilling`,
  `grill-with-docs`), multi-session planning (`wayfinder`, re-homed to local markdown), and the
  handoff pattern. `to-spec`/`to-tickets` fold into `writing-plans`; `domain-modeling`/
  `codebase-design` fold into `grill-with-docs`.
- **Ralphex (`umputun/ralphex`)** — the **plan-file format** (`### Task N:` + `[ ]` checkboxes
  + `## Validation Commands`), **one-task-per-pass** execution discipline, the **parallel
  multi-agent review** with an **adversarial "verify every finding at file:line before fixing"**
  gate, and the **critical/major/minor** severity taxonomy (effort trivial/small/medium/large for
  over-engineering findings). Two review modes: comprehensive (6 reviewers) and critical-only
  (quality + implementation).
- **Umputun (`cc-thingz`)** — the review *criteria*: `smells` (style + classic smells +
  anti-patterns), `simplification` (over-engineering catalog), `quality` (bugs/security/
  correctness), `testing` (incl. fake-test detection), `implementation`, `documentation`; plus
  the `writing-style` AI-speak killer. His **library stack is NOT copied** (see §11).

## 4. The fused workflow

```
UNDERSTAND ──▶ PLAN ─────────▶ IMPLEMENT ─────▶ REVIEW ─────────▶ (DEBUG) ─▶ FINISH
 grilling      writing-plans   executing-plans  code-review        systematic  handoff
 grill-with-   wayfinder       + tdd            (6 reviewers,       -debugging  + finalize
 docs          (big work)                        verify-then-fix)               (fold-in)
```

Each phase maps to one primary skill; AGENTS.md documents when to use which.

## 5. Repository layout

```
.claude/
  skills/<name>/SKILL.md                canonical skills (native Claude Code discovery)
  skills/code-review/reviewers/*.md     the 6 Go-tailored reviewer specs (portable md)
  agents/*.md                           6 thin review subagents (Claude parallel Task fan-out)
agents/skills/<name>/SKILL.md           tool-neutral MIRROR (Codex / Pi / opencode)
AGENTS.md                               THE HUB — conventions + workflow + skill index
CLAUDE.md                               trimmed → points to AGENTS.md (+ minimal-code guard)
scripts/sync-skills.sh                  mirrors .claude/skills → agents/skills
Makefile                                + test, lint, sqlc, sync-skills targets
sqlc.yaml                               new: sqlc config (postgres, database/sql, schema=migrations)
sql/queries/                            new: hand-written SQL for sqlc
docs/
  plans/                                ralphex-format plans (active)
  plans/completed/                      finished plans
  plans/maps/                           wayfinder decision maps
  adr/                                  grill-with-docs ADRs
  glossary.md                           grill-with-docs glossary
  agents/                               existing agent reference docs (kept, linked from AGENTS.md)
  superpowers/specs/                    this spec + future design docs
```

The existing `docs/agents/{README,env-reference,middleware-api}.md` are kept and referenced from
AGENTS.md — no duplication of their content.

## 6. Portability model

- **AGENTS.md is the portability contract.** It contains the always-on conventions + workflow +
  a **skill index** (each skill: one-line when-to-use + relative path). Any AGENTS.md-aware tool
  reads the matching skill file on demand.
- **Claude Code** additionally auto-discovers `.claude/skills/*/SKILL.md` (model-invoked) and
  `.claude/agents/*.md` (review subagents).
- **Sync:** skills are authored **once** in `.claude/skills/`. `make sync-skills`
  (→ `scripts/sync-skills.sh`) copies the tree to `agents/skills/`. The neutral path exists so
  non-Claude users aren't pointed at a Claude-branded directory. Editing rule: **edit
  `.claude/skills/`, then run `make sync-skills`** (documented in AGENTS.md + script header).

## 7. AGENTS.md contents / CLAUDE.md reduction

**AGENTS.md** (new root file) sections:
1. Project context + tech stack (accurate: Go 1.25, cobra, chi v5, keycloak OIDC, postgres,
   goose, oapi-codegen, sqlc, slog, testify).
2. Core principles (minimal & pragmatic — carried from current CLAUDE.md).
3. Go conventions (§11).
4. Configuration (env var table — carried from current CLAUDE.md; links `docs/agents/env-reference.md`).
5. The workflow (§4) and **skill index**.
6. Review engine summary + severity taxonomy.
7. Editing/sync note.

**CLAUDE.md** shrinks to: one-line intro, a directive to read **AGENTS.md** for all guidance, a
retained inline **minimal-code guard** (safety net), and the existing skill-invocation note. No
duplicated conventions — AGENTS.md is the single source.

## 8. Skill inventory

Each skill file uses portable frontmatter (§13) and plain-markdown body.

**Planning / understanding**
- `grilling` (Matt) — relentless one-question-at-a-time interview loop. Look up *facts* from the
  environment; put *decisions* to the user; recommend an answer per question; don't act until
  shared understanding is confirmed.
- `grill-with-docs` (Matt) — grilling that also produces/updates `docs/adr/NNNN-<slug>.md` and
  `docs/glossary.md` as decisions land.
- `wayfinder` (Matt→local) — plan work too big for one session as a **decision map** in
  `docs/plans/maps/<name>.md` (destination, decisions-so-far, open fog, out-of-scope), resolved
  one frontier ticket at a time. No external tracker.

**Plan**
- `writing-plans` (ralphex + Matt) — **central skill.** Explore context → propose 2–3 approaches →
  draft plan → **get explicit approval** → write `docs/plans/YYYYMMDD-<slug>.md` in the plan
  format (§10). Includes the YAGNI/simplicity self-check gate.

**Implement**
- `executing-plans` (ralphex + Matt) — find the FIRST `### Task N:` with `[ ]`; do ONLY that
  task; write/update tests; run Validation Commands until green; flip `[ ]`→`[x]`; commit
  `feat: <task>`; stop. When no `[ ]` remain, move the plan to `docs/plans/completed/`. Compatible
  with the `ralph-loop` plugin for autonomous looping.
- `tdd` (Matt/Umputun) — red-green-refactor with this repo's test conventions.

**Review**
- `code-review` (ralphex + Umputun) — see §9.

**Debug**
- `systematic-debugging` (Matt) — reproduce → minimize → hypothesize → instrument → fix; no fix
  proposed before reproduction.

**Meta / ship**
- `writing-style` (Umputun) — AI-speak killer for commit messages, PR descriptions, and review
  comments (NOT README/public docs). Includes the filler/overused-word/hedging lists.
- `handoff` (Matt) — compact handoff doc for continuation by another session/agent.

**Repo-specific scaffolding**
- `add-middleware` (converted command) — scaffold `internal/server/middleware/<snake>.go`;
  show registration in `server.go` `Initialize()`; optional env toggle pattern. slog only.
- `add-endpoint` (converted from `add-route`, OpenAPI-first) — edit `oapi/openapi.yaml` →
  `make gen-oapi` → implement handler in `internal/api/handlers/` → register in
  `internal/api/router/routes.go` with the right auth middleware. Notes the raw-chi fallback.
- `add-migration` (new) — scaffold a goose migration in `migrations/` (`-- +goose Up/Down`),
  naming + `make db-migrate` (or existing `app db migrate`) reminder.
- `add-sqlc-query` (new) — write SQL in `sql/queries/<domain>.sql` with sqlc annotations
  (`-- name: X :one/:many/:exec`) → `make sqlc` → use generated types. Requires the sqlc setup in §12.

## 9. Review engine

`code-review` orchestrates:
1. **Context** — the orchestrator/agents run `git log <base>..HEAD` and `git diff <base>...HEAD`
   themselves; diffs are NOT embedded into subagent prompts.
2. **Fan-out** — reviewers run **read-only, in parallel**. On Claude, as subagents in
   `.claude/agents/` launched in a single message (Task tool). On other tools, applied
   sequentially from the same `reviewers/*.md` specs.
   Reviewers: `go-quality`, `go-smells`, `go-simplification`, `go-testing`, `go-implementation`,
   `go-documentation`. Modes: **comprehensive** (all 6) / **critical-only** (quality +
   implementation, critical/major only).
3. **Collect + dedup** — same `file:line` + same issue merged.
4. **Verify EVERY finding** — read the actual code at `file:line` (±20–30 lines); classify
   **CONFIRMED** (fix) or **FALSE POSITIVE** (discard). Pre-existing issues are in scope.
5. **Fix + validate + commit** — fix confirmed; run `make test` + `make lint` until green;
   `git commit -m "fix: address code review findings"`. Re-review to verify fixes converge to
   zero new findings.

**Severity:** critical / major / minor. **Over-engineering effort:** trivial / small / medium /
large; large → recommend a follow-up plan instead of fixing in-review. Format:
`SEVERITY: file:line — description` (unmarked ⇒ minor). Reviewers **report only**; the
orchestrator fixes. Reviewer subagents are **read-only** (must not modify the working tree).

## 10. Plan file format (contract)

`writing-plans` produces (`docs/plans/YYYYMMDD-<slug>.md`):

```
# <Title>

## Overview
<what will be implemented>

## Context
- Files involved: ...
- Related patterns: ...
- Dependencies: ...

## Development Approach
- Testing approach: Regular (code then tests) or TDD (test first)
- Complete each task fully before the next
- Every task MUST include new/updated tests; all tests pass before the next task

## Validation Commands
- Test: `make test`   (go test -race ./...)
- Lint: `make lint`   (golangci-lint run)

## Implementation Steps

### Task 1: <title>
**Files:** Modify/Create: `path`
- [ ] step
- [ ] write tests for this task
- [ ] run make test + make lint — must pass before Task 2

### Task N: Verify acceptance criteria
- [ ] full test suite passes
- [ ] linter passes

### Task N+1: Update documentation
- [ ] update README.md if user-facing
- [ ] update AGENTS.md if internal patterns/conventions changed
```

Rules: checkboxes live **only** inside Task sections; `### Task N:` is a structural token (English);
tasks sized 3–7 items, one component each, linear dependencies; YAGNI (no future-proofing,
no backwards-compat unless asked); no manual/deploy items as `[ ]` inside Task sections.

## 11. Go conventions (this repo's stack)

Kept from Umputun (stack-agnostic): return early / shallow nesting; interfaces defined in the
**consumer** package and kept small; check errors immediately, wrap with context only when it
helps; **no history/changelog comments**; lowercase in-code comments; exported identifiers
documented; table-driven tests; don't over-shrink functions.

Adapted to THIS repo (NOT umputun's libraries):
- **Logging:** `log/slog` (structured) — never fmt.Println, never a third-party logger.
- **HTTP:** chi v5 router; middleware are `func(http.Handler) http.Handler`.
- **CLI:** cobra.
- **Auth:** keycloak/OIDC via `pkg/keycloak` + `internal/server/auth`.
- **DB:** postgres via `pkg/db`; migrations via **goose** (`migrations/`); queries via **sqlc**
  (`sql/queries/` → generated types). SQL over ORM.
- **API:** OpenAPI-first via `oapi-codegen` (`oapi/openapi.yaml` → `make gen-oapi`).
- **Tests:** stdlib `testing` + `stretchr/testify` (`require` for fatal, `assert` for non-fatal);
  `go test -race ./...`.
- **Config:** all via env vars (`internal/server/config/env`).
- No commit trailers or "Generated with…" lines; no "Test plan" sections in PRs.

## 12. Project changes (beyond skill files)

- **Makefile targets:** `test` (`go test -race ./...`), `lint` (`golangci-lint run`),
  `sqlc` (`sqlc generate`), `sync-skills` (`./scripts/sync-skills.sh`). Keep existing
  build/run/clean/gen-oapi.
- **sqlc setup:** add `sqlc.yaml` (engine postgres, `database/sql` output to match `pkg/db`
  helpers, schema = `migrations/`, queries = `sql/queries/`, output package under
  `internal/db` or `pkg/db/gen` — final path chosen in the plan), create empty `sql/queries/`
  with a `.gitkeep` + a README stub. Generation yields nothing until migrations define schema —
  acceptable for a starter; the `add-sqlc-query` skill drives real use.
- **docs dirs:** create `docs/plans/{,completed,maps}`, `docs/adr/`, `docs/glossary.md` (stub).
- **`.golangci.yml`:** currently empty; add a minimal, sensible config so `make lint` works
  (final rule set decided in the plan; conservative defaults).
- **`.gitignore`:** no change needed (ralphex progress already ignored).

## 13. Skill file format (portable)

Each `SKILL.md`:
```
---
name: <kebab-case>
description: <one line — when to use>
---
<body: plain markdown, imperative instructions>
```
This frontmatter is the common subset Claude Code requires and that Codex/Pi/opencode tolerate.
No Claude-only frontmatter keys in the canonical files, so the mirror is byte-identical.

## 14. Risks / open questions (resolved inline)

- **sqlc output package path** — decide in the plan (`internal/db/gen` proposed). Low risk.
- **`.golangci.yml` rule set** — start conservative (govet, staticcheck, errcheck, ineffassign,
  unused); expand later. Low risk.
- **add-endpoint vs existing add-route command** — the old command scaffolded raw chi; the repo
  is OpenAPI-first now. The skill documents the codegen flow as primary and raw-chi as fallback.
- **Subagent portability** — the 6 reviewers are Claude subagents *and* plain reference specs;
  non-Claude tools use the specs sequentially. No functionality lost, only parallelism.

## Appendix A — reviewer criteria (adapted, summary)

- **go-quality:** logic errors, edge cases (nil/empty/boundary/concurrent), error handling,
  resource cleanup, races/deadlocks, data integrity; security (input validation, authz,
  injection, secret exposure, info disclosure). Defers over-engineering to go-simplification.
- **go-smells:** judged **against this project's own conventions** (cite AGENTS.md or existing
  code). Style consistency (naming/org/imports/comments/error/log patterns); classic smells (dead
  code, duplication, long functions, deep nesting, magic values, mixed abstraction levels);
  anti-patterns (god objects, shotgun surgery, feature envy, primitive obsession).
- **go-simplification:** over-engineering catalog — excessive abstraction layers, premature
  generalization, unnecessary indirection, future-proofing excess, unnecessary fallbacks,
  premature optimization. Scope = what THIS branch adds/worsens; complexity required by the plan
  is not a finding; skip generated/vendored/fixture code; prove "unused/no-callers" with a
  project-wide search.
- **go-testing:** missing/undertested paths, test quality, **fake-test detection** (always-pass
  tests, hardcoded-output checks, mock-verifying-mock, ignored errors, commented-out cases), test
  independence, edge coverage. Reports; does not fix.
- **go-implementation:** reads the plan; requirement coverage, correctness of approach, wiring/
  integration, completeness (imports/interfaces/migrations), scope creep.
- **go-documentation:** README (user-facing changes) vs AGENTS.md (patterns/conventions/commands)
  drift + plan drift. Reports; does not edit.

## Appendix B — loop / signal semantics (adapted)

State lives **outside** the model: plan-file checkboxes = task state, git = work, an optional
progress log = trace. `executing-plans` does one task per pass and stops, so it composes with any
external loop (ralph-loop plugin, ralphex binary, or manual re-invocation). Review convergence
rule: after fixing issues, re-review; done only when a full pass finds **zero** new issues.
Severity/verify semantics per §9.
