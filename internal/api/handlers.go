package api

import (
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/korjavin/fastfooddb/internal/store"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	Store    *store.Store
	Manifest *store.Manifest
}

// productResponse is the JSON shape returned for a single product.
type productResponse struct {
	Barcode  string   `json:"barcode"`
	Name     string   `json:"name"`
	Kcal100g *float32 `json:"kcal100g"`
	Protein  *float32 `json:"protein"`
	Fat      *float32 `json:"fat"`
	Carbs    *float32 `json:"carbs"`
}

func toProductResponse(p store.Product) productResponse {
	return productResponse{
		Barcode:  p.Barcode,
		Name:     p.Name,
		Kcal100g: nanToNil(p.Kcal100g),
		Protein:  nanToNil(p.Protein),
		Fat:      nanToNil(p.Fat),
		Carbs:    nanToNil(p.Carbs),
	}
}

// nanToNil converts a float32 NaN to nil (JSON null); otherwise returns a pointer.
func nanToNil(f float32) *float32 {
	if math.IsNaN(float64(f)) {
		return nil
	}
	return &f
}

// Health returns a liveness check with manifest metadata.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{"status": "ok"}
	if h.Manifest != nil {
		resp["schema_version"] = h.Manifest.SchemaVersion
		resp["build_time"] = h.Manifest.BuildTime
	}
	writeJSON(w, http.StatusOK, resp)
}

// FoodByBarcode looks up nutritional info by product barcode.
func (h *Handler) FoodByBarcode(w http.ResponseWriter, r *http.Request) {
	barcode := r.PathValue("barcode")
	slog.Info("food by barcode request", "barcode", barcode)

	p, found, err := h.Store.Get(barcode)
	if err != nil {
		slog.Error("barcode lookup failed", "barcode", barcode, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, toProductResponse(p))
}

// FoodSearch searches for foods by name.
func (h *Handler) FoodSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	limit := 20
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if n, err := strconv.Atoi(ls); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}

	slog.Info("food search request", "query", q, "limit", limit)

	products, err := h.Store.Search(q, limit)
	if err != nil {
		slog.Error("search failed", "query", q, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	results := make([]productResponse, len(products))
	for i, p := range products {
		results[i] = toProductResponse(p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
