package importer

import (
	"fmt"
	"math"
)

// OFFProduct is the minimal subset of an Open Food Facts JSONL record.
type OFFProduct struct {
	Code             string         `json:"code"`
	ProductName      string         `json:"product_name"`
	ProductNameEn    string         `json:"product_name_en"`
	GenericName      string         `json:"generic_name"`
	ShortDescription string         `json:"short_description"`
	Nutriments       map[string]any `json:"nutriments"`
}

// Name returns the best available product name using the fallback order:
// product_name → product_name_en → generic_name → short_description → "".
func (p *OFFProduct) Name() string {
	if p.ProductName != "" {
		return p.ProductName
	}
	if p.ProductNameEn != "" {
		return p.ProductNameEn
	}
	if p.GenericName != "" {
		return p.GenericName
	}
	return p.ShortDescription
}

// Kcal100g extracts kcal per 100g from nutriments.
// Prefers energy-kcal_100g; falls back to energy-kj_100g / 4.184.
// Returns NaN when not available or outside plausible range [0, 10000].
func (p *OFFProduct) Kcal100g() float32 {
	if v, ok := extractFloat(p.Nutriments, "energy-kcal_100g"); ok {
		return validateNutriment(float32(v), 0, 10000)
	}
	if v, ok := extractFloat(p.Nutriments, "energy-kj_100g"); ok {
		return validateNutriment(float32(v/4.184), 0, 10000)
	}
	return float32(math.NaN())
}

// Protein100g extracts protein per 100g. Returns NaN when missing or outside [0, 100].
func (p *OFFProduct) Protein100g() float32 {
	if v, ok := extractFloat(p.Nutriments, "proteins_100g"); ok {
		return validateNutriment(float32(v), 0, 100)
	}
	return float32(math.NaN())
}

// Fat100g extracts fat per 100g. Returns NaN when missing or outside [0, 100].
func (p *OFFProduct) Fat100g() float32 {
	if v, ok := extractFloat(p.Nutriments, "fat_100g"); ok {
		return validateNutriment(float32(v), 0, 100)
	}
	return float32(math.NaN())
}

// Carbs100g extracts carbohydrates per 100g. Returns NaN when missing or outside [0, 100].
func (p *OFFProduct) Carbs100g() float32 {
	if v, ok := extractFloat(p.Nutriments, "carbohydrates_100g"); ok {
		return validateNutriment(float32(v), 0, 100)
	}
	return float32(math.NaN())
}

// validateNutriment returns NaN if v is outside [min, max], otherwise v.
func validateNutriment(v float32, min, max float32) float32 {
	if math.IsNaN(float64(v)) || v < min || v > max {
		return float32(math.NaN())
	}
	return v
}

// extractFloat coerces a nutriments map value to float64.
func extractFloat(m map[string]any, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return 0, false
		}
		return x, true
	case string:
		var f float64
		if _, err := fmt.Sscanf(x, "%f", &f); err == nil {
			return f, true
		}
	}
	return 0, false
}
