# OpenAPI (step 3 of the slice)

`oapi/openapi.yaml` is the HTTP contract. `make gen-oapi` turns it into
`internal/api/openapi_types.gen.go` (the wire structs) and `internal/api/openapi_server.gen.go`
(`ServerInterface` plus the embedded spec). The file carries a `.yaml` extension and **JSON
content** — keep writing JSON so the whole document stays one syntax.

Worked example: the `/v1/notes` and `/v1/notes/{id}` paths with the `Note` and `CreateNoteRequest`
schemas.

## Where things go

- `paths` — one key per URL, then one key per method. Path parameters shared by every method on a
  path go in a `parameters` array on the path itself, as `/v1/notes/{id}` does with `id`.
- `components.schemas` — every request and response body, `PascalCase` (`Note`,
  `CreateNoteRequest`). The name becomes the Go type name verbatim.
- `components.responses` — the shared error responses (`BadRequest`, `Unauthorized`, `Forbidden`,
  `NotFound`). Reference them, do not redescribe them:
  `"401": { "$ref": "#/components/responses/Unauthorized" }`.
- `tags` — one per domain, declared at the top level with a description.

## An operation

```json
"post": {
  "tags": ["Notes"],
  "summary": "Create a note",
  "description": "Creates a note owned by the authenticated caller.",
  "operationId": "createNote",
  "requestBody": {
    "required": true,
    "content": {
      "application/json": {
        "schema": { "$ref": "#/components/schemas/CreateNoteRequest" },
        "example": { "title": "Shopping list", "body": "milk, bread" }
      }
    }
  },
  "responses": {
    "201": {
      "description": "The created note",
      "content": {
        "application/json": { "schema": { "$ref": "#/components/schemas/Note" } }
      }
    },
    "400": { "$ref": "#/components/responses/BadRequest" },
    "401": { "$ref": "#/components/responses/Unauthorized" }
  }
}
```

`operationId` is `camelCase` and becomes the method name on `ServerInterface` (`createNote` →
`CreateNote`), so it must be unique across the document.

List every status the handler can actually return. The `responses` map is the checklist the handler
test works through — a status missing here is a status nobody tests.

Security is global (`"security": [{ "bearerAuth": [] }]`). An unauthenticated operation overrides it
with `"security": []`.

## Schemas and the Go types they produce

```json
"Note": {
  "type": "object",
  "required": ["id", "title", "body", "createdAt", "updatedAt"],
  "properties": {
    "id": { "type": "string", "format": "uuid" },
    "title": { "type": "string", "maxLength": 200 },
    "body": { "type": "string" },
    "createdAt": { "type": "string", "format": "date-time" },
    "updatedAt": { "type": "string", "format": "date-time" }
  }
}
```

- Property names are `camelCase`; the generated Go field is exported and the JSON tag keeps the
  spec's spelling.
- **`required` decides pointers.** A property in `required` generates a value (`Title string`); one
  outside it generates a pointer (`Body *string`), and the handler dereferences it after a nil
  check. Requiring what is genuinely required is what keeps handler code flat.
- `format: uuid` → `openapi_types.UUID` (an alias of `uuid.UUID`, so a `gen.Note.ID` assigns
  straight across), `format: date-time` → `time.Time`, `format: email` → `openapi_types.Email`.
- Constraints (`maxLength`, `minimum`, `enum`) are documentation to oapi-codegen: it generates no
  validation. Enforce them in the handler and keep the two in step — `maxNoteTitleLen` in
  `internal/api/handlers/notes.go` mirrors `maxLength` here, and says so in a comment.

## Generating and using the output

```bash
make gen-oapi
```

Confirm the operation appears on `ServerInterface` and the structs exist in
`openapi_types.gen.go`. Routes are registered by hand in `internal/api/router/routes_<domain>.go`
rather than through the generated mux, so `ServerInterface` is the contract to conform to, not a
type to implement.

Decode requests into the generated request type and render the generated response type
(`api.CreateNoteRequest`, `api.Note`). Mapping `gen.Note` → `api.Note` in one small function keeps
the database row and the wire shape free to differ, and makes a spec change a compile error rather
than a silent mismatch.
