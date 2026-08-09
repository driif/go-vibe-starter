Detect over-engineered code — code that works but is more complex than necessary. Report problems
only.

Scope: report only over-engineering THIS branch introduces or makes worse. Pre-existing complexity
the branch does not touch is out of scope. Complexity explicitly required by the active plan is not
a finding. Skip generated code (`*.gen.go`, mocks), vendored deps, and test fixtures. Before
reporting any "unused / no callers / never triggers", verify with a project-wide search and cite
the result.

## Excessive Abstraction Layers
- Wrapper adds nothing; factory for a single implementation; interface on the producer side
  (define interfaces in the consumer); handler→service→repository layers that only pass through;
  DTO/mapper overkill.

## Premature Generalization
- Generic solution for a specific problem; config/options objects for 2–3 options; plugin points
  nothing extends; overloaded struct with many optional fields.

## Unnecessary Indirection
- Pass-through wrappers; builder for simple construction; custom types wrapping stdlib primitives;
  multiple middlewares that could be one.

## Future-Proofing Excess
- Unused extension points/hooks; internal v1/v2 with one version; feature flags for permanent
  decisions.

## Unnecessary Fallbacks
- Fallback that never triggers; legacy mode always disabled; dual old+new implementations with no
  callers for old; silent fallbacks hiding problems instead of failing fast.

## Premature Optimization
- Caching rarely-accessed data; custom data structures where slices/maps work; worker pools for
  occasional tasks; connection pooling overkill.

## Report Format (per finding)
- Location: file:line
- Pattern: which over-engineering pattern
- Problem: why it adds unnecessary complexity
- Simplification: what simpler code looks like
- Effort: trivial/small/medium/large  (large → recommend a follow-up plan, don't fix in review)
