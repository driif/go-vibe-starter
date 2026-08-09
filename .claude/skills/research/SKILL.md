---
name: research
description: Investigating a question against primary sources and leaving cited findings in the repo. Use when a decision needs facts about a dependency, an API, a spec or an RFC, when a library's real behaviour has to be checked before coding against it, or when the user asks for reading legwork to be delegated.
---

Research runs in a **background agent** so this session keeps working while it reads. Dispatch it,
carry on with the current task, and read the file it leaves behind.

- **Claude Code fast path**: dispatch a subagent in the background with the brief below, then return
  to what you were doing. Do not block on it.
- **Elsewhere (Codex, Pi, opencode)**: run the brief yourself in one focused pass, then resume the
  interrupted work.

## The brief

Hand the agent all five points. It edits exactly one file and nothing else.

**1. The question.** State it, why it matters, and what the answer decides. "Does `go-pkgz/rest`'s
`RenderJSON` set `Content-Type` and can it fail after the first byte, because `/ready` must return
503 with a named failing check" is answerable; "research go-pkgz/rest" is not.

**2. Primary sources only.** Ranked:

1. The dependency's own source, at the version `go.mod` pins.
2. `go doc` for that package.
3. The owning project's official docs, spec, or RFC.
4. First-party behaviour you can observe: the tool's `--help`, a real API response, a `curl`.

A blog post, a StackOverflow answer, or a model's memory is a *lead*, not a source. Follow it back to
the thing it describes and cite that.

**3. Go dependency source is local and read-only.** No network needed:

```sh
go list -m -f '{{.Dir}}' github.com/go-pkgz/rest
# /Users/<you>/go/pkg/mod/github.com/go-pkgz/rest@v1.23.2  — grep and read it
go doc github.com/go-pkgz/rest RenderJSON
go doc -all github.com/go-chi/chi/v5
```

Read the pinned version, not `main` on GitHub — the shipped behaviour is the one in the cache. The
same applies to the toolchain: `go doc net/http Server`, `goose --help`, `sqlc --help`.

**4. Every claim cites its source, inline.** `file:line` for code (module-cache path or repo path),
section or anchor for docs, URL plus retrieval date for anything on the web. A claim that cannot name
its source is an open question, not a finding.

**5. The output is one markdown file**: `docs/research/YYYYMMDD-<slug>.md`, created alongside the
existing `docs/adr/` and `docs/agents/` notes (`mkdir -p docs/research` if it is the first one).
Sections:

```
# <question>
## Answer          — three sentences at most, the decision-grade summary
## Findings        — each claim with its citation
## Open questions  — what the sources did not settle
## Sources         — what was read, with versions and dates
```

## When it returns

Read the file, then act on it here: a finding that settles a design choice becomes an ADR through
`domain-modeling`; a finding that changes scope goes into the spec issue. Where a source contradicts
its own documentation, the source wins and the file says so — that gap is usually the real finding.

**Completion criterion**: every claim in the file traces to a named primary source, and everything
the sources left unsettled sits under Open questions rather than being smoothed into prose.
