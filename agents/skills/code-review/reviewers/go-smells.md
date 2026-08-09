Review code for style consistency, convention adherence, and code smells. Report problems only —
no positive observations. Judge against this project's own conventions (cite AGENTS.md or existing
code as evidence), not personal preference.

## Project Convention Check
1. Read AGENTS.md (and CLAUDE.md) for project rules and any referenced standards.
2. Check whether the changed code follows the established conventions.

## Style Consistency
1. Naming — do new names follow existing patterns?
2. Code organization — structured like existing code in the same package?
3. Import ordering — matches the project (stdlib, third-party, local)?
4. Comment style — lowercase in-code comments; no history/changelog comments.
5. Error handling — matches the project's established patterns.
6. Logging — slog, consistent with the rest of the codebase.

## Code Smells
1. Dead code — unused functions, variables, imports, parameters.
2. Duplicated logic — copy-paste that should be consolidated.
3. Long functions — doing too many things.
4. Deep nesting — prefer early returns.
5. Magic numbers/strings — unexplained literals.
6. Inconsistent abstraction levels — mixing high and low level operations.

## Anti-patterns
1. God objects — types with too many responsibilities.
2. Shotgun surgery — one change touches many unrelated files.
3. Feature envy — code using another type's data more than its own.
4. Primitive obsession — primitives where a domain type would be clearer.

## Report Format (per finding)
- Location: file:line
- Issue: what's inconsistent or smelly
- Convention: the project convention (cite AGENTS.md or existing code)
- Fix: specific suggestion
