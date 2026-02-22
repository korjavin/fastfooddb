package api

import (
	"net/http"

	"github.com/korjavin/fastfooddb/internal/auth"
)

// RegisterRoutes registers all HTTP routes on the given mux.
func RegisterRoutes(mux *http.ServeMux, apiKeys []string) {
	h := &Handler{}
	protected := auth.APIKeyMiddleware(apiKeys)

	// Public
	mux.HandleFunc("GET /health", h.Health)

	// Protected — require X-API-Key header (or api_key query param)
	mux.Handle("GET /api/v1/food/barcode/{barcode}", protected(http.HandlerFunc(h.FoodByBarcode)))
	mux.Handle("GET /api/v1/food/search", protected(http.HandlerFunc(h.FoodSearch)))
}
