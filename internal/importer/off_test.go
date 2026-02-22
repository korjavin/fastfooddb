package importer

import (
	"math"
	"testing"
)

func TestValidateNutriment(t *testing.T) {
	nan := float32(math.NaN())

	tests := []struct {
		name    string
		v       float32
		min     float32
		max     float32
		wantNaN bool
		want    float32
	}{
		{"valid mid-range", 50, 0, 100, false, 50},
		{"valid at min", 0, 0, 100, false, 0},
		{"valid at max", 100, 0, 100, false, 100},
		{"below min", -1, 0, 100, true, 0},
		{"above max", 101, 0, 100, true, 0},
		{"NaN input", nan, 0, 100, true, 0},
		{"kcal valid", 500, 0, 10000, false, 500},
		{"kcal above max", 10001, 0, 10000, true, 0},
		{"kcal negative", -1, 0, 10000, true, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateNutriment(tc.v, tc.min, tc.max)
			if tc.wantNaN {
				if !math.IsNaN(float64(got)) {
					t.Errorf("validateNutriment(%v, %v, %v) = %v; want NaN", tc.v, tc.min, tc.max, got)
				}
			} else {
				if got != tc.want {
					t.Errorf("validateNutriment(%v, %v, %v) = %v; want %v", tc.v, tc.min, tc.max, got, tc.want)
				}
			}
		})
	}
}

func TestOFFProductNutriments(t *testing.T) {
	p := &OFFProduct{
		Nutriments: map[string]any{
			"energy-kcal_100g":   float64(250),
			"proteins_100g":      float64(10),
			"fat_100g":           float64(-5),   // invalid: negative
			"carbohydrates_100g": float64(10001), // too high for kcal range but fine for carbs max=100: invalid
		},
	}

	if got := p.Kcal100g(); got != 250 {
		t.Errorf("Kcal100g() = %v; want 250", got)
	}
	if got := p.Protein100g(); got != 10 {
		t.Errorf("Protein100g() = %v; want 10", got)
	}
	if !math.IsNaN(float64(p.Fat100g())) {
		t.Errorf("Fat100g() = %v; want NaN (negative value)", p.Fat100g())
	}
	if !math.IsNaN(float64(p.Carbs100g())) {
		t.Errorf("Carbs100g() = %v; want NaN (above 100)", p.Carbs100g())
	}
}

func TestOFFProductKcalFallback(t *testing.T) {
	// Falls back to kj / 4.184
	p := &OFFProduct{
		Nutriments: map[string]any{
			"energy-kj_100g": float64(418.4),
		},
	}
	got := p.Kcal100g()
	// 418.4 / 4.184 ≈ 100
	if math.IsNaN(float64(got)) || got < 99 || got > 101 {
		t.Errorf("Kcal100g() kj fallback = %v; want ~100", got)
	}
}
