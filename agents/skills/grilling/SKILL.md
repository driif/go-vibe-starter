---
name: grilling
description: Use when the user wants to stress-test a plan, decision, or idea, or says "grill me/this" — runs a relentless one-question-at-a-time interview until shared understanding.
---

# Grilling

Interview the user relentlessly about every aspect of the plan, decision, or idea until you
reach a shared understanding. Walk down each branch of the decision tree, resolving dependencies
between decisions one at a time.

## Rules

- **One question at a time.** Wait for the answer before asking the next. Asking several at once
  is bewildering.
- **Recommend an answer** for every question. Say what you'd choose and why, then let the user
  decide.
- **Look up facts; ask only for decisions.** If something can be discovered from the filesystem,
  the code, or tooling, find it yourself. The *decisions* are the user's — put each one to them
  and wait.
- **Resolve dependencies in order.** When one decision depends on another, settle the upstream one
  first.
- **Don't act until the user confirms** you have reached a shared understanding.

## When you're done

Summarize the shared understanding in a few lines and confirm it with the user before moving on
(usually to `make-plan`, or to `grill-with-docs` if the decisions should be recorded).
