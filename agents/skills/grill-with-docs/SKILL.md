---
name: grill-with-docs
description: Use when sharpening a design or plan that should leave durable docs behind — runs the grilling interview and records decisions as ADRs and glossary terms as they land.
---

# Grill with docs

Sharpen a design or plan through a relentless interview, and capture the durable results as
Architecture Decision Records and glossary terms as you go.

## Process

1. Run the `grilling` loop: one question at a time, recommend an answer, look up facts, decisions
   are the user's. Walk the decision tree until the design is sharp.
2. **Build the domain model actively.** When a term is fuzzy or overloaded, grill until it is
   precise, then record it in `docs/glossary.md` (term + one or two sentences). Reuse agreed terms
   verbatim afterward — shared language keeps later work short and unambiguous.
3. **Record each significant decision** as an ADR at `docs/adr/NNNN-<slug>.md` (next zero-padded
   number). Use this format:

   ```
   # NNNN. <Title>

   - Status: accepted
   - Date: <YYYY-MM-DD>

   ## Context
   <what forced the decision>

   ## Decision
   <what was decided>

   ## Consequences
   <trade-offs and follow-ups>
   ```

## When you're done

Confirm the design is captured and point to the ADRs and glossary entries you wrote. Hand off to
`make-plan` for implementation.
