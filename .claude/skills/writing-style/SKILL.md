---
name: writing-style
description: Use when writing commit messages, PR descriptions, or code-review comments — strips AI-speak for brevity and directness. Not for README or public docs.
---

# Writing style

Applies to commit messages, PR descriptions, code-review comments, and tickets. **Not** for
README, public docs, or blog posts. If the user has their own writing rules, those win.

Be brief, direct, and specific. Cite `file:line` and commit refs. State opinions plainly —
over-engineering is a bug worth flagging.

## AI-speak to cut

**Filler — delete entirely:** "It's important to note that", "It's worth mentioning", "In order
to" → "to", "plays a crucial role in", "at the end of the day", "that being said", "moving
forward", "in terms of".

**Overused → simpler:** comprehensive→full, robust→solid, leverage→use, utilize→use,
facilitate→help, optimal→best, seamless→(drop it), streamline→simplify.

**Abstract nouns → verbs:** "the implementation of" → implemented, "make a decision" → decide,
"perform an analysis" → analyze, "provide assistance" → help.

**Hedging → direct.** Cut "I think maybe we could possibly". Use transitions (Furthermore,
Additionally, Moreover, In conclusion) sparingly. Delete meta-commentary like "This approach works
by…" or "The benefit of this is…".

## Never

- "Thanks in advance", "Hope this helps", "Let me know if you have any questions", "Best regards"
  in comments.
- Commit trailers or "Generated with…" / "Co-Authored-By" lines in project commits.
- "Test plan" sections in PR descriptions.
