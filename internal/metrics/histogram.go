package metrics

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// Histogram is a fixed-bucket latency histogram with lock-free observation.
// Bucket boundaries are upper bounds in microseconds; the last bound must be
// math.MaxInt64 to act as the catch-all bucket.
type Histogram struct {
	bounds []int64        // upper bounds in microseconds
	counts []atomic.Int64 // one counter per bucket
	total  atomic.Int64   // total number of observations
}

// NewHistogram creates a Histogram with the given bucket upper bounds
// (in microseconds). The last element should be math.MaxInt64.
func NewHistogram(boundsMicros []int64) *Histogram {
	h := &Histogram{
		bounds: make([]int64, len(boundsMicros)),
		counts: make([]atomic.Int64, len(boundsMicros)),
	}
	copy(h.bounds, boundsMicros)
	return h
}

// Observe records a single latency measurement. Lock-free, no allocation.
func (h *Histogram) Observe(d time.Duration) {
	micros := d.Microseconds()
	for i, bound := range h.bounds {
		if micros <= bound {
			h.counts[i].Add(1)
			h.total.Add(1)
			return
		}
	}
	// Overflow: add to last bucket
	h.counts[len(h.counts)-1].Add(1)
	h.total.Add(1)
}

// Snapshot captures a consistent view of the histogram and computes percentiles.
type Snapshot struct {
	P50   time.Duration `json:"p50"`
	P95   time.Duration `json:"p95"`
	P99   time.Duration `json:"p99"`
	Total int64         `json:"total"`
}

// Snapshot returns a point-in-time snapshot of the histogram.
func (h *Histogram) Snapshot() Snapshot {
	total := h.total.Load()
	if total == 0 {
		return Snapshot{}
	}

	counts := make([]int64, len(h.counts))
	for i := range h.counts {
		counts[i] = h.counts[i].Load()
	}

	p50 := percentile(h.bounds, counts, total, 50)
	p95 := percentile(h.bounds, counts, total, 95)
	p99 := percentile(h.bounds, counts, total, 99)

	return Snapshot{
		P50:   p50,
		P95:   p95,
		P99:   p99,
		Total: total,
	}
}

// percentile computes the pth percentile duration from bucket data.
// Returns the upper bound of the bucket that contains the pth percentile.
func percentile(bounds []int64, counts []int64, total int64, p int) time.Duration {
	target := int64(math.Ceil(float64(total) * float64(p) / 100.0))
	var cumulative int64
	for i, c := range counts {
		cumulative += c
		if cumulative >= target {
			bound := bounds[i]
			if bound == math.MaxInt64 {
				// Return the previous bound as a best-effort upper estimate
				if i > 0 {
					bound = bounds[i-1]
				} else {
					bound = 0
				}
			}
			return time.Duration(bound) * time.Microsecond
		}
	}
	return 0
}

// Registry holds a named set of histograms.
type Registry struct {
	mu   sync.RWMutex
	hists map[string]*Histogram
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{hists: make(map[string]*Histogram)}
}

// Register creates and stores a named Histogram. If the name already exists the
// existing histogram is returned unchanged.
func (r *Registry) Register(name string, bounds []int64) *Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.hists[name]; ok {
		return h
	}
	h := NewHistogram(bounds)
	r.hists[name] = h
	return h
}

// Snapshot returns snapshots for all registered histograms keyed by name.
func (r *Registry) Snapshot() map[string]Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]Snapshot, len(r.hists))
	for name, h := range r.hists {
		out[name] = h.Snapshot()
	}
	return out
}

// Pre-defined bucket sets (upper bounds in microseconds).

// BucketsBarcode suits fast point-lookup operations (sub-millisecond expected).
var BucketsBarcode = []int64{50, 100, 250, 500, 750, 1000, 1500, 2000, 5000, 10000, math.MaxInt64}

// BucketsSearch suits search operations (multi-millisecond expected).
var BucketsSearch = []int64{500, 1000, 2500, 5000, 10000, 15000, 20000, 30000, 50000, 100000, math.MaxInt64}
