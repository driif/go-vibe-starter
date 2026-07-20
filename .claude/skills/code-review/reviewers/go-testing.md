Review test coverage and quality. Report problems only. You may run the test suite and report
failures; do not fix them.

## Coverage
1. New code paths without tests; untested error paths; uncovered branches; integration needs at
   system boundaries.

## Test Quality
1. Tests verify behavior, not implementation details.
2. Independent, order-agnostic; descriptive names; success AND error paths; edge/boundary cases.

## Fake Test Detection
- Tests that always pass regardless of code changes.
- Tests checking hardcoded values instead of actual output.
- Tests verifying mock behavior instead of the code using the mock.
- Ignored errors (`_ =`) or empty error checks.
- Conditional assertions that always pass; commented-out failing cases.

## Report Format (per finding)
- Location: test file:function
- Issue / Impact (what bug could slip through) / Fix
