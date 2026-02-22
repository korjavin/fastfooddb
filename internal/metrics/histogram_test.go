package metrics

import (
	"math"
	"testing"
	"time"
)

func TestHistogram_Empty(t *testing.T) {
	h := NewHistogram(BucketsBarcode)
	snap := h.Snapshot()
	if snap.Total != 0 {
		t.Errorf("expected Total=0, got %d", snap.Total)
	}
	if snap.P50 != 0 || snap.P95 != 0 || snap.P99 != 0 {
		t.Errorf("expected zero percentiles for empty histogram, got p50=%v p95=%v p99=%v",
			snap.P50, snap.P95, snap.P99)
	}
}

func TestHistogram_Observe_Percentiles(t *testing.T) {
	// Use simple bounds: 100, 200, 500, MaxInt64 (microseconds)
	bounds := []int64{100, 200, 500, math.MaxInt64}
	h := NewHistogram(bounds)

	// Observe 100 samples:
	//   50 × 50µs  → bucket[0] (≤100)
	//   30 × 150µs → bucket[1] (≤200)
	//   15 × 300µs → bucket[2] (≤500)
	//   5  × 600µs → bucket[3] (MaxInt64)
	for i := 0; i < 50; i++ {
		h.Observe(50 * time.Microsecond)
	}
	for i := 0; i < 30; i++ {
		h.Observe(150 * time.Microsecond)
	}
	for i := 0; i < 15; i++ {
		h.Observe(300 * time.Microsecond)
	}
	for i := 0; i < 5; i++ {
		h.Observe(600 * time.Microsecond)
	}

	snap := h.Snapshot()
	if snap.Total != 100 {
		t.Fatalf("expected Total=100, got %d", snap.Total)
	}

	// P50 (50th obs) → 50th cumulative hits bucket[0] (cumulative=50)
	if snap.P50 != 100*time.Microsecond {
		t.Errorf("P50: expected 100µs, got %v", snap.P50)
	}

	// P95 (95th obs) → 95th cumulative hits bucket[2] (cumulative=95 at ≤500)
	if snap.P95 != 500*time.Microsecond {
		t.Errorf("P95: expected 500µs, got %v", snap.P95)
	}

	// P99 (99th obs) → hits overflow bucket[3] → returns bound[2]=500µs (best-effort)
	if snap.P99 != 500*time.Microsecond {
		t.Errorf("P99: expected 500µs (best-effort from overflow), got %v", snap.P99)
	}
}

func TestHistogram_SingleObservation(t *testing.T) {
	h := NewHistogram(BucketsSearch)
	h.Observe(3 * time.Millisecond) // 3000µs → bucket for 5000µs

	snap := h.Snapshot()
	if snap.Total != 1 {
		t.Errorf("expected Total=1, got %d", snap.Total)
	}
	// All percentiles point at the same bucket
	if snap.P50 != snap.P95 || snap.P95 != snap.P99 {
		t.Errorf("all percentiles should be equal for single obs: p50=%v p95=%v p99=%v",
			snap.P50, snap.P95, snap.P99)
	}
}

func TestRegistry_Snapshot(t *testing.T) {
	reg := NewRegistry()
	h1 := reg.Register("barcode_get", BucketsBarcode)
	h2 := reg.Register("search", BucketsSearch)

	h1.Observe(200 * time.Microsecond)
	h2.Observe(5 * time.Millisecond)

	// Register same name again returns existing histogram
	h1b := reg.Register("barcode_get", BucketsBarcode)
	if h1b != h1 {
		t.Error("expected same histogram for duplicate Register call")
	}

	snap := reg.Snapshot()
	if len(snap) != 2 {
		t.Errorf("expected 2 entries in snapshot, got %d", len(snap))
	}
	if snap["barcode_get"].Total != 1 {
		t.Errorf("barcode_get Total: expected 1, got %d", snap["barcode_get"].Total)
	}
	if snap["search"].Total != 1 {
		t.Errorf("search Total: expected 1, got %d", snap["search"].Total)
	}
}
