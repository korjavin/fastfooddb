package store_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/korjavin/fastfooddb/internal/store"
)

// seedProducts writes n products into the store via WriteBatch and returns
// the barcode of the middle product for use in point-lookup benchmarks.
func seedProducts(tb testing.TB, s *store.Store, n int) (midBarcode string) {
	tb.Helper()
	batch := s.NewWriteBatch()

	names := []string{
		"Apple Juice", "Chicken Pasta", "Whole Milk", "Oat Bread",
		"Strawberry Yoghurt", "Grilled Salmon", "Dark Chocolate",
		"Brown Rice", "Tomato Soup", "Cheddar Cheese",
	}

	for i := 0; i < n; i++ {
		barcode := fmt.Sprintf("%013d", i+1)
		name := names[i%len(names)] + fmt.Sprintf(" %d", i)
		batch.Put(store.Product{
			Barcode:  barcode,
			Name:     name,
			Kcal100g: float32(50 + i%500),
			Protein:  float32(i%30) + 0.5,
			Fat:      float32(i%40) + 0.1,
			Carbs:    float32(i%80) + 1.0,
		})
		if i == n/2 {
			midBarcode = barcode
		}
		// Flush every 5000 to match importer behaviour
		if batch.Len() >= 5000 {
			if err := batch.Flush(); err != nil {
				tb.Fatalf("flush: %v", err)
			}
		}
	}
	if err := batch.Close(); err != nil {
		tb.Fatalf("batch close: %v", err)
	}
	return midBarcode
}

// openReadOnly creates a seeded store, closes it (write mode), then reopens
// read-only to simulate the real server environment.
func openBenchStore(tb testing.TB, n int) (*store.Store, string) {
	tb.Helper()
	dir := tb.TempDir()

	ws, err := store.Create(dir)
	if err != nil {
		tb.Fatalf("create store: %v", err)
	}
	mid := seedProducts(tb, ws, n)
	if err := ws.Close(); err != nil {
		tb.Fatalf("close write store: %v", err)
	}

	rs, err := store.OpenReadOnly(dir)
	if err != nil {
		tb.Fatalf("open read-only store: %v", err)
	}
	tb.Cleanup(func() { rs.Close() })
	return rs, mid
}

func BenchmarkGet(b *testing.B) {
	s, barcode := openBenchStore(b, 10_000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, found, err := s.Get(barcode)
		if err != nil {
			b.Fatalf("Get: %v", err)
		}
		if !found {
			b.Fatalf("barcode %q not found", barcode)
		}
	}
}

func BenchmarkSearch_CommonTerm(b *testing.B) {
	s, _ := openBenchStore(b, 10_000)
	b.ResetTimer()
	b.ReportAllocs()

	queries := []string{"apple", "chicken pasta", "milk", "chocolate"}
	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		_, err := s.Search(q, 20)
		if err != nil {
			b.Fatalf("Search(%q): %v", q, err)
		}
	}
}

func BenchmarkSearch_FuzzyTerm(b *testing.B) {
	s, _ := openBenchStore(b, 10_000)
	b.ResetTimer()
	b.ReportAllocs()

	// Intentional typos to exercise the fuzzy path
	queries := []string{"chiken bred", "appel juise", "tomatto soup", "salman"}
	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		_, err := s.Search(q, 20)
		if err != nil {
			b.Fatalf("Search(%q): %v", q, err)
		}
	}
}

// Sanity check: ensure seeded data is retrievable (not run by bench runner).
func TestBenchSeed_Sanity(t *testing.T) {
	s, barcode := openBenchStore(t, 100)
	p, found, err := s.Get(barcode)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatalf("barcode %q not found after seeding", barcode)
	}
	if math.IsNaN(float64(p.Kcal100g)) {
		t.Errorf("Kcal100g is NaN, expected a real value")
	}
}
