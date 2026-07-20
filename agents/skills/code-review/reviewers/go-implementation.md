Verify the code actually achieves the goal. Read the active plan/spec first (`docs/plans/`,
`docs/superpowers/specs/`). Report problems only. Focus on correctness of approach, not style.

1. Requirement coverage — every stated requirement is implemented.
2. Correctness of approach — the chosen approach solves the stated problem.
3. Wiring/integration — new code is actually reachable (routes registered in
   `internal/api/router/routes.go`, commands wired in `cmd/`, migrations applied).
4. Completeness — no missing imports, interface methods, or migrations.
5. Scope creep — changes solve only the stated problem.

## Report Format (per finding)
- Location: file:line
- Severity: critical/major/minor
- Issue / Impact / Fix
