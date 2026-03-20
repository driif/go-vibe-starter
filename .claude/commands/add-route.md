Add a new chi route handler to this project.

Ask the user for the following if not already provided:
1. HTTP method (GET, POST, PUT, PATCH, DELETE)
2. URL path (e.g. `/users`, `/users/{id}`)
3. Handler function name (e.g. `ListUsers`, `CreateUser`)
4. Package/domain (e.g. `users`, `orders`) — used to determine the file path

Then:

1. Create the handler file at `internal/api/<domain>/<snake_case_name>.go` with this structure:

```go
package <domain>

import (
    "net/http"
)

func <HandlerName>(w http.ResponseWriter, r *http.Request) {
    // TODO: implement
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
}
```

2. Show the user the line to add in `internal/server/server.go` inside `Initialize()`, after the middleware block:

```go
r.<METHOD>("<path>", <domain>.<HandlerName>)
```

Or if it belongs to a route group:
```go
r.Route("/<domain>", func(r chi.Router) {
    r.<METHOD>("/", <domain>.<HandlerName>)
})
```

3. Remind the user to add the import for the new package:
```go
"github.com/driif/go-vibe-starter/internal/api/<domain>"
```

Rules to follow:
- Handler signature is always `func(w http.ResponseWriter, r *http.Request)`
- Use `chi.URLParam(r, "id")` to read path parameters
- Use `json.NewEncoder(w).Encode(v)` to write JSON responses
- Use `log/slog` for logging inside handlers — never fmt.Println
- Do not add error wrapping layers without clear benefit
