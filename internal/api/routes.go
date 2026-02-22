package api

import (
	"net/http"

	"github.com/korjavin/fastfooddb/internal/auth"
	"github.com/korjavin/fastfooddb/internal/metrics"
	"github.com/korjavin/fastfooddb/internal/store"
)

// RegisterRoutes registers all HTTP routes on the given mux.
func RegisterRoutes(mux *http.ServeMux, apiKeys []string, s *store.Store, m *store.Manifest, reg *metrics.Registry) {
	h := &Handler{Store: s, Manifest: m}
	if reg != nil {
		h.BarcodeHist = reg.Register("barcode_get", metrics.BucketsBarcode)
		h.SearchHist = reg.Register("search", metrics.BucketsSearch)
	}
	protected := auth.APIKeyMiddleware(apiKeys)

	// Public
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /metrics", h.Metrics(reg))

	// Protected — require X-API-Key header (or api_key query param)
	mux.Handle("GET /api/v1/food/barcode/{barcode}", protected(http.HandlerFunc(h.FoodByBarcode)))
	mux.Handle("GET /api/v1/food/search", protected(http.HandlerFunc(h.FoodSearch)))
}
