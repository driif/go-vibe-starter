---
name: diagnose-bug
description: Use on any bug, test failure, or unexpected behavior before proposing a fix — a structured loop: reproduce, minimize, hypothesize, instrument, fix, verify.
---

# Diagnose bug

Work bugs in a disciplined loop. Do not propose a fix before you can reproduce the problem.

## Loop

1. **Reproduce.** Get a reliable, minimal repro — a failing test is ideal. No fix is proposed
   until a repro exists.
2. **Minimize.** Shrink inputs and state to the smallest case that still fails.
3. **Hypothesize.** Form one specific, falsifiable hypothesis about the cause.
4. **Instrument.** Add targeted `slog` logging, assertions, or a focused test to confirm or kill
   the hypothesis. Confirm the cause — don't guess-and-patch.
5. **Fix.** Make the minimal change. Add a regression test that fails before the fix and passes
   after.
6. **Verify.** `make test` green, `make lint` clean, and the original repro is gone.

If a hypothesis is killed, return to step 3 with what you learned — never widen the change to
"just try things."
