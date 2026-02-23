# fastfooddb

> Blazingly fast API to get nutritional macros by barcode and food name.

## Motivation & Architecture

This project was built to solve two specific, narrow use cases:
1. **Barcode Lookup:** Fetching food nutritional data instantly by scanning a product's barcode.
2. **Fast Search:** Providing an extremely fast, responsive search by food name for UI typeaheads.

To achieve this without relying on heavy external database services or hitting upstream APIs at runtime, we built a custom **importer for the OpenFoodFacts database**. The importer processes the massive OpenFoodFacts dump offline, extracts only the essential macros (calories, protein, fat, carbohydrates), and packs them into two embedded engines:
- **[Pebble](https://github.com/cockroachdb/pebble):** A fast embedded key-value store used to serve food payloads by their exact barcode instantly. The data is compressed in a highly optimized custom binary layout.
- **[Bleve](https://github.com/blevesearch/bleve):** A text indexing library used to provide flexible, fuzzy, lightning-fast full-text searches across product names.

Because of the aggressive filtering and binary packing, the **entire database size is reduced to around 750Mb**. This allows the server to run locally with minimal RAM, relying heavily on the OS page cache for sub-millisecond data retrieval.

## Performance Metrics

Based on recent benchmarks, the local data serving delivers the following latency:

| Endpoint | p50 | p95 | p99 |
|----------|-----|-----|-----|
| **Barcode Lookup** | < 1ms | < 1ms | < 1ms |
| **Search (Name)** | 30ms | 100ms | 100ms |

## Quick Start

```bash
cp .env.example .env
# Edit .env and set your API_KEYS
./start.sh
```

The server starts on port `8080` by default.

## API Endpoints

All endpoints except `/health` require the `X-API-Key` header (or `?api_key=` query parameter).

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness check — returns `{"status":"ok"}` |
| `GET` | `/api/v1/food/barcode/{barcode}` | Look up food by product barcode |
| `GET` | `/api/v1/food/search?q={query}` | Search foods by name |

### Example

```bash
# Health check (no auth needed)
curl http://localhost:8080/health

# Lookup by barcode
curl -H "X-API-Key: your-key" http://localhost:8080/api/v1/food/barcode/5000112637922

# Search
curl -H "X-API-Key: your-key" "http://localhost:8080/api/v1/food/search?q=banana"
```

## API Documentation

The full API specification is available in the [openapi.yaml](openapi.yaml) file. You can view it using any OpenAPI/Swagger compatible viewer (like Swagger Editor or Postman).

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Listen port |
| `API_KEYS` | _(empty — no auth)_ | Comma-separated list of valid API keys |
| `CORS_ORIGINS` | `*` | Comma-separated allowed CORS origins, or `*` |
| `DOMAIN` | — | Domain for Traefik routing (production only) |

## Project Structure

```
cmd/server/main.go          — entry point, wires everything together
internal/api/               — HTTP handlers and route registration
internal/auth/apikey.go     — API key validation middleware
internal/middleware/        — CORS, rate limiting, request logging
```

## Development

```bash
go test ./...
go build ./cmd/server
```

## Deployment

The project ships with GitHub Actions workflows:

- **`deploy.yml`** — builds on `master` push, updates the `deploy` branch with the new image tag, triggers Portainer webhook (`PORTAINER_WEBHOOK` secret).
- **`dev-deploy.yml`** — same for any other branch, uses `deploy-dev` branch and `PORTAINER_WEBHOOK_DEV` secret.

Required GitHub secrets:

| Secret | Description |
|--------|-------------|
| `PORTAINER_WEBHOOK` | Portainer stack webhook URL (production) |
| `PORTAINER_WEBHOOK_DEV` | Portainer stack webhook URL (dev) |

`GITHUB_TOKEN` is used automatically for GHCR image push.
