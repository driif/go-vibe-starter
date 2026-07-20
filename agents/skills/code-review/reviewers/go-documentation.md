Check for documentation drift. Read the current README.md and AGENTS.md first; report a gap only
when the item is not already documented. Report only — do not edit files.

- README.md must document: new features, CLI flags/commands, API endpoints, config/env options,
  changed behavior, new dependencies, breaking changes. Skip internal refactors, bugfixes, tests.
- AGENTS.md must document: new architectural patterns, conventions, build/test commands, new
  libraries, structure changes, non-obvious techniques.
- Plan drift: does the plan file in `docs/plans/` still match what was built?

## Report Format (per finding)
- Trigger (what changed) / Missing (what's undocumented) / Section / Suggested content
