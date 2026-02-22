# Project Guide for AI Assistants

## Module
`github.com/korjavin/fastfooddb` — Pure Go REST API, stateless (no database yet).

## Key Files

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Entry point — wires server, middleware, routes |
| `internal/api/routes.go` | Route registration — add new endpoints here |
| `internal/api/handlers.go` | HTTP handler methods — add new handlers here |
| `internal/auth/apikey.go` | API key parsing and validation middleware |
| `internal/middleware/chain.go` | Middleware composition helper |
| `internal/middleware/cors.go` | CORS headers |
| `internal/middleware/ratelimit.go` | Per-IP rate limiting (in-memory) |
| `internal/middleware/logging.go` | Structured request logging |

## How to Add a New Endpoint

1. Add a method to `Handler` in `internal/api/handlers.go`.
2. Register the route in `internal/api/routes.go` using `mux.Handle` or `mux.HandleFunc`.
   - Use Go 1.22+ method-qualified patterns: `"GET /api/v1/foo/{id}"`.
   - Wrap with `protected(...)` if the endpoint requires an API key.
3. Use `r.PathValue("id")` to extract path parameters (Go 1.22+).
4. Use `writeJSON(w, status, payload)` to send JSON responses.

## Adding a Storage Layer

The `Handler` struct in `handlers.go` is intentionally empty — add service/repository fields as you implement them:

```go
type Handler struct {
    foodRepo FoodRepository
}
```

Pass dependencies when constructing in `routes.go` or `main.go`.

## Conventions

- Structured logging via `log/slog` (standard library).
- No CGO — keep it that way for simple cross-compilation.
- Middleware is applied globally in `main.go`; per-route auth wrapping is done in `routes.go`.
- Rate limit: 100 req/s, burst 20 (configured in `main.go`, change as needed).
