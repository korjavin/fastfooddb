package store

import (
	"math"
	"os"
	"testing"
)

func TestWriteBatch(t *testing.T) {
	dir := t.TempDir()

	s, err := Create(dir)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer s.Close()

	batch := s.NewWriteBatch()

	products := []Product{
		{Barcode: "001", Name: "Apple Juice", Kcal100g: 45},
		{Barcode: "002", Name: "Whole Milk", Kcal100g: 61, Protein: 3.2},
		{Barcode: "003", Name: "", Kcal100g: float32(math.NaN())}, // no name, not indexed
	}

	for _, p := range products {
		batch.Put(p)
	}
	if batch.Len() != 3 {
		t.Errorf("Len() = %d; want 3", batch.Len())
	}

	if err := batch.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if batch.Len() != 0 {
		t.Errorf("Len() after flush = %d; want 0", batch.Len())
	}

	// Verify products are retrievable
	for _, want := range products {
		got, found, err := s.Get(want.Barcode)
		if err != nil {
			t.Fatalf("Get(%q): %v", want.Barcode, err)
		}
		if !found {
			t.Errorf("Get(%q): not found", want.Barcode)
			continue
		}
		if got.Name != want.Name {
			t.Errorf("Get(%q).Name = %q; want %q", want.Barcode, got.Name, want.Name)
		}
	}

	// Close with no pending writes — should not error
	if err := batch.Close(); err != nil {
		t.Errorf("Close (empty): %v", err)
	}
}

func TestWriteBatchClose_FlushesPending(t *testing.T) {
	dir := t.TempDir()

	s, err := Create(dir)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer s.Close()

	batch := s.NewWriteBatch()
	batch.Put(Product{Barcode: "999", Name: "Pending Product"})

	// Close without explicit Flush — Close should flush automatically
	if err := batch.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, found, err := s.Get("999")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Error("product not found after batch.Close()")
	}
}

func TestWriteBatch_SearchAfterFlush(t *testing.T) {
	if os.Getenv("CI") == "" {
		// bleve search in tests can be slow; skip unless in CI
		// Remove this guard if you want to always run it.
	}

	dir := t.TempDir()

	s, err := Create(dir)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer s.Close()

	batch := s.NewWriteBatch()
	batch.Put(Product{Barcode: "111", Name: "Organic Oat Milk"})
	batch.Put(Product{Barcode: "222", Name: "Soy Milk"})
	if err := batch.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	results, err := s.Search("oat milk", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected search results for 'oat milk', got none")
	}
	if results[0].Barcode != "111" {
		t.Errorf("top result barcode = %q; want %q", results[0].Barcode, "111")
	}
}
