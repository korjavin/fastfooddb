package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Handler holds dependencies for HTTP handlers.
// Add your storage/service fields here when the storage layer is implemented.
type Handler struct{}

// Health returns a simple liveness check response.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// FoodByBarcode looks up nutritional info by product barcode.
// TODO: implement storage lookup.
func (h *Handler) FoodByBarcode(w http.ResponseWriter, r *http.Request) {
	barcode := r.PathValue("barcode")
	slog.Info("food by barcode request", "barcode", barcode)
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// FoodSearch searches for foods by name.
// TODO: implement storage lookup.
func (h *Handler) FoodSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}
	slog.Info("food search request", "query", q)
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
