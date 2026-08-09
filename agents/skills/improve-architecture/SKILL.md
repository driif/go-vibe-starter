---
name: improve-architecture
disable-model-invocation: true
description: Scan the repo for deepening opportunities, rank them in a short markdown report, then grill the one you pick.
---

Periodic maintenance, not a step in the main flow. Nothing in the ticket lifecycle calls this — it
runs when a codebase has accumulated enough friction to be worth a look, typically after a run of
features has landed. Its output is a ranked report and one grilled decision, never a refactor
applied on the spot.

Every opportunity is a **deepening opportunity**: a place where a shallow module could become a
deep one. Read the `codebase-design` skill first and use its words exactly —
module, interface, depth, seam, adapter, leverage, locality — in the scan and in every line of the
report.

## 1. Scope before scanning

Deepening pays off on code that keeps changing, so aim at what moves.

- The user named a direction — a package, a subsystem, a pain point? Take it and skip the rest.
- Otherwise read the hot spots out of history: `git log --oneline -n 200 --name-only` and let the
  paths that keep reappearing pull first. Scattered history with no hot spot means widening to the
  whole of `internal/`.

Then load the standing context: `CONTEXT.md` for the domain terms, `docs/adr/` for decisions already
settled in the area you are about to touch.

## 2. Scan for friction

Read the code in scope — on Claude, dispatch a read-only subagent to walk it and report back;
elsewhere, walk it inline. Explore rather than run a checklist, and note where reading hurts:

- Understanding one concept means bouncing between several packages.
- A package whose exported surface is nearly as complicated as what it hides.
- A handler that reaches through three packages to assemble behaviour nobody else can reuse.
- Helpers extracted for testability alone, while the real bugs live in how they are called.
- Code with no test, or with tests that had to reach past the exported surface to get a grip.
- Packages that import each other's internals — the seam is drawn in the wrong place.

Apply the **deletion test** to anything you suspect is shallow: delete it in your head, and see
whether the complexity concentrates elsewhere or simply vanishes. "Vanishes" is the signal.

## 3. Report in markdown

Print the report in the conversation. Markdown, at most five opportunities, ranked strongest first.
No file lands in the repo and no interfaces get proposed yet.

```markdown
## Deepening opportunities — <scope, and why this scope>

### 1. <name the module, in CONTEXT.md's words> — Strong
- **Seam** — where the interface would sit once this is deep
- **Now** — the depth problem, cited at `path/file.go:NN`
- **Change** — the smallest change that deepens it
- **Pays off in** — leverage for callers, locality for maintainers, and what becomes testable

### 2. <…> — Worth exploring
### 3. <…> — Speculative

## Top recommendation
<which one first, and why it beats the others>
```

Each entry names a seam, cites at least one `file:line` you actually read, and proposes the
**smallest** change that deepens the module — not the ideal end state. Strength is one of `Strong`,
`Worth exploring`, `Speculative`, and a `Strong` needs the deletion test behind it.

An opportunity that contradicts an ADR appears only when the friction is real enough to justify
reopening that ADR, and says so in its entry: *contradicts ADR-0007, worth reopening because …*.

Close by asking which one the user wants to explore.

## 4. Grill the pick

Run `grilling` on the chosen opportunity: the constraints, the dependency category (`codebase-design`
names the four), the shape of the deepened interface, what stays behind the seam, which tests
survive. Keep the domain model current as decisions land by running `domain-modeling`:

- A deepened module named after a concept `CONTEXT.md` does not carry → add the term.
- The user rejects the opportunity for a reason a future scan would need in order to stop
  re-suggesting it → offer an ADR.
- The user wants alternative interfaces explored → `codebase-design` and its design-it-twice
  reference.

The grilling ends in a decision, not a diff. Once the shape is settled, the work becomes a spec and
tickets like anything else: `/to-spec`, then `/to-tickets`.
