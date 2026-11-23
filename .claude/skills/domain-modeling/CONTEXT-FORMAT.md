# CONTEXT.md format

CONTEXT.md holds the domain language and links to the package map in AGENTS.md §3.
`/setup-project` detects an empty vocabulary through the exact row `| | |`.

```md
# Context

What this service does and who it serves.

## Ubiquitous language

| Term | Means |
|---|---|
| Submission | A completed form sent by a caller. Avoid: response, entry. |
```

Define a term in one or two sentences. Choose one word and name rejected synonyms when useful.
Keep infrastructure terms separate from the product vocabulary. Split tables when natural clusters
appear. Preserve the empty stub only while there are no agreed domain terms. Initialization leaves
this file intact. The layout belongs in AGENTS.md, not in a second module map here.
