# Language

The word-level branch of [`writing-for-agents`](SKILL.md): the tokens a document runs on, how to
steer with them, and what to cut. `SKILL.md` decides where material sits; this file decides how it
is worded.

## Leading words

A **leading word** is a compact concept already living in the model's pretraining that the agent
thinks with while running the document. Repeated as a token, never as a sentence, it accumulates a
distributed definition and anchors a region of behaviour in the fewest tokens by recruiting priors
the model already holds. Coining your own works if you define it clearly, but a made-up word recruits
no priors: you pay in definition tokens what a pretrained word gives free.

It anchors twice. In the body, *execution*: the agent reaches for the same behaviour every time the
word appears. In a pointer, *invocation*: when the same word lives in your prompts, your issues and
this codebase, the agent links that shared language to the material and reaches it more reliably.
So hunt for passages that collapse into a token. "Migration, then query, then spec, then handler,
then route, then test" becomes *the slice*.

### Shared vocabulary

Reuse these tokens verbatim across the pack. The shared spelling is what lets one skill's wording
trigger another skill's material.

Doctrine words, for writing documents: `context pointer` · `context load` · `cognitive load` ·
`progressive disclosure` · `co-location` · `completion criterion` · `leading word` · `no-op` ·
`sediment` · `sprawl` · `guardrail`.

Repo words, for writing about the work:

| Word | Means |
|---|---|
| `the slice` | the chain a feature walks: migration → sqlc query → OpenAPI → handler → route → test |
| `tracer bullet` | a ticket thin enough to walk the whole slice end to end in one pass |
| `deep module` | small interface over substantial implementation; the target shape for a Go package |
| `seam` | the boundary where an implementation swaps out (`AUTH_PROVIDER`, a consumer-defined interface) |
| `red-green` | write the failing test, make it pass, then refactor |
| `frontier` | the edge of what is decided; the next interview question sits there |
| `confirmed` | a finding re-read at `file:line` and shown to be real; anything else is a false positive |
| `drift` | generated output no longer matching its source; `make gen` then `git diff --exit-code` |
| `principal` | the authenticated caller carried in the request context (subject + roles) |
| `hub` | `internal/api/router/routes.go`, which only calls per-domain `register*` funcs |
| `DEMO` | the strippable reference slice (`notes`), removed by `make init` |

### Negation

Steering by prohibition drags the forbidden behaviour into context and makes it *more* available.
*Don't think of an elephant*, and the elephant is all there is: the negation is a weak modifier that
the activated concept overruns, so the ban half-reads as an instruction. Prompt the **positive**,
stating the target behaviour ("write lowercase in-code comments describing current state") so the
banned one is never spoken. A prohibition earns its place only as a **guardrail** you cannot phrase
positively ("never hand-edit `*.gen.go`", "never write an AI-attribution trailer"), and even then it
travels with its positive target.

## Pruning

- Keep each meaning in a **single source of truth**, so changing the behaviour is a one-place edit.
  Duplication costs maintenance and tokens, and inflates a meaning's prominence past its real rank.
  Here that source is `AGENTS.md` for conventions and commands, `CONTEXT.md` for domain terms,
  `docs/adr/` for decisions already made.
- The **environment** is a source of truth too: the `Makefile`, `.env.example`, `oapi/openapi.yaml`,
  the directory layout. A document restating it is a cache of a lookup, earning its load only when
  the lookup is expensive. Cache what the agent cannot find by looking — the unwritten convention,
  the reason behind a choice, the gotcha no config confesses. Leave `make test` to the Makefile.
- Check every line for **relevance**: does it still bear on what the document does? Without a pruning
  discipline the default fate is **sediment**, stale layers that settle because adding feels safe and
  removing feels risky.
- Hunt **no-ops** sentence by sentence. An instruction the model already obeys by default pays load
  to say nothing. The test is model-relative: does it change behaviour versus the default? Settle
  disagreements by running the document, not by debate. Delete the whole sentence rather than trim
  words from it. The test also grades leading words: a word too weak to beat the default (*be
  careful*) is a no-op, and the fix is a stronger word (*relentless*), not a different technique.
