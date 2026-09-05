---
name: writing-style
description: Prose that lands in the project's permanent record. Use when writing a commit message, a PR title or description, a GitHub issue body (`gh issue create`), or a code-review comment.
---

Governs what this repo records about its own work: commits, PRs, issues, review comments. Reference
documents are out of scope — `README.md` and public docs are human-facing prose, and documents an
agent consumes (`AGENTS.md`, skills, `docs/agents/*`) belong to `writing-for-agents`. A rule the user
states for their own repo outranks this file.

## Guardrail: the record names no assistant

Every commit message, PR body, issue and review comment reads as the work of the human who owns the
branch. Say what changed and why, sign nothing.

Never write an AI-attribution trailer or credit line: no `Co-Authored-By: Claude`, no
`🤖 Generated with …`, no "written with AI assistance", no mention of Claude, Anthropic or an
assistant anywhere in git history or on GitHub. `.ralphex/config` leaves `commit_trailer` empty for
this reason. Before handing over a message or body, re-read its last block: the final line is
content.

## Voice

Brief, direct, specific. State opinions plainly — "this interface has one implementation, drop it"
beats "we might consider possibly simplifying here". Concrete anchors carry the weight:
`internal/api/handlers/notes.go:42`, commit `92891fa`, issue `#17`. Vary sentence length; a run of
em-dash asides reads as machine cadence.

## Commit messages

- Subject: `<type>: <imperative summary>`, matching what `git log` already shows (`feat`, `fix`,
  `docs`, `chore`, `refactor`, `test`). Lowercase, no trailing period.
- Body explains **why**: the reason the change exists, what it broke or unblocked, what was traded.
  The diff already shows what.
- Name the anchors — the issue it closes (`Closes #17`), the commit it fixes by short SHA, the
  `file:line` the reader should open first.

```
fix: return 401 for unsupported auth scheme, keep 400 for malformed token

RFC 6750 treats an unknown scheme as an authentication failure, not a bad
request. internal/server/auth/auth.go:164 collapsed both onto 400, so
clients retried the same credentials instead of re-authenticating.

Closes #17
```

## PR descriptions

- Title is the commit subject when there is one commit; otherwise the subject the squash should get.
- Open with why, in one or two sentences. Then what changed, grouped by area with paths.
- Link the issue (`Closes #17`) and the ADR if a decision drove it (`docs/adr/0003-auth-provider.md`).
- Verification goes in one line when it is not obvious — "`make test`, `make lint` and the codegen
  `drift` check all clean". A checkbox "Test plan" section is noise; the reviewer reads CI.

## Issue bodies

The issue's *shape* comes from its template and from the skill that files it (`to-spec`,
`to-tickets`). This skill governs the prose inside it: state the observable behaviour, the expected
behaviour, and the smallest reproduction. For a `bug`, give the command and the output verbatim. Put
what you know into the body and what you do not into an "Open questions" list rather than guessing on
the reader's behalf.

## Review comments

- Anchor every comment at `file:line`, state the concrete failure ("`ListNotesByOwner` ignores
  `limit`, so a 10k-row owner buffers the whole table"), then the fix.
- Report only `confirmed` findings — a suspicion re-read at the line and shown to be real. A hunch
  goes in the summary as a question, not in the diff as a demand.
- Severity in plain words. Over-engineering is a finding worth raising.

## AI-speak to cut

**Filler — delete the whole sentence.** "It's important to note that", "It's worth mentioning",
"plays a crucial role in", "at the end of the day", "that being said", "moving forward", "in terms
of". "In order to" → "to".

**Overused → simpler.** comprehensive → full · robust → solid · leverage → use · utilize → use ·
facilitate → help · optimal → best · streamline → simplify · seamless → cut it · delve → look at.

**Abstract noun → verb.** "the implementation of X" → "implemented X" · "make a decision" → "decide"
· "perform an analysis" → "analyze" · "provide assistance" → "help".

**Hedge → claim.** "I think maybe we could possibly" → say the thing. Keep "Furthermore",
"Additionally", "Moreover", "In conclusion" for the rare case that earns them.

**Closers — cut.** "Thanks in advance", "Hope this helps", "Let me know if you have any questions",
"Best regards", and meta-commentary like "This approach works by…" or "The benefit of this is…".
