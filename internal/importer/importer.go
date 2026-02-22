package importer

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/korjavin/fastfooddb/internal/store"
)

const (
	maxBarcodeLen = 100
	batchSize     = 5_000
)

// Import reads a gzip-compressed JSONL Open Food Facts dump, builds a Pebble
// KV store and Bleve full-text index inside outputDir, and returns the
// resulting manifest.
//
// Every product with a non-empty barcode is written to Pebble.
// Products whose resolved name is non-empty are also indexed in Bleve.
// Products with an empty or over-long barcode are skipped entirely.
func Import(dumpPath, outputDir string, verbose bool) (*store.Manifest, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	s, err := store.Create(outputDir)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}
	defer s.Close()

	f, err := os.Open(dumpPath)
	if err != nil {
		return nil, fmt.Errorf("open dump: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("open gzip reader: %w", err)
	}
	defer gz.Close()

	var (
		productCount int64
		indexedCount int64
		skippedCount int64
		skipReasons  = make(map[string]int64)
		startTime    = time.Now()
	)

	batch := s.NewWriteBatch()

	scanner := bufio.NewScanner(gz)
	// Some OFF lines can be very large; allocate a generous buffer.
	buf := make([]byte, 0, 4*1024*1024)
	scanner.Buffer(buf, 16*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var off OFFProduct
		if err := json.Unmarshal(line, &off); err != nil {
			skippedCount++
			skipReasons["parse_error"]++
			slog.Debug("json unmarshal error, skipping line", "error", err)
			continue
		}

		barcode := off.Code
		if barcode == "" {
			skippedCount++
			skipReasons["empty_barcode"]++
			continue
		}

		if len(barcode) > maxBarcodeLen {
			skippedCount++
			skipReasons["barcode_too_long"]++
			continue
		}

		name := off.Name()

		p := store.Product{
			Barcode:  barcode,
			Name:     name,
			Kcal100g: off.Kcal100g(),
			Protein:  off.Protein100g(),
			Fat:      off.Fat100g(),
			Carbs:    off.Carbs100g(),
		}

		batch.Put(p)
		productCount++
		if name != "" {
			indexedCount++
		}

		if batch.Len() >= batchSize {
			if err := batch.Flush(); err != nil {
				return nil, fmt.Errorf("batch flush: %w", err)
			}
		}

		if verbose && productCount%100_000 == 0 {
			elapsed := time.Since(startTime)
			rate := float64(productCount) / elapsed.Seconds()
			slog.Info("import progress",
				"products", productCount,
				"indexed", indexedCount,
				"skipped", skippedCount,
				"rate_per_s", int(rate),
				"elapsed", elapsed.Round(time.Second),
			)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	if err := batch.Close(); err != nil {
		return nil, fmt.Errorf("final batch flush: %w", err)
	}

	m := &store.Manifest{
		BuildTime:     time.Now().UTC(),
		DumpSource:    dumpPath,
		ProductCount:  productCount,
		IndexedCount:  indexedCount,
		SkippedCount:  skippedCount,
		SchemaVersion: 1,
		SkipReasons:   skipReasons,
	}

	if err := store.WriteManifest(outputDir, m); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return m, nil
}
