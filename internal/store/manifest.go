package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const manifestFile = "manifest.json"

// Manifest records metadata about a built data directory.
type Manifest struct {
	BuildTime     time.Time       `json:"build_time"`
	DumpSource    string          `json:"dump_source"`
	ProductCount  int64           `json:"product_count"`
	IndexedCount  int64           `json:"indexed_count"`
	SkippedCount  int64           `json:"skipped_count"`
	SchemaVersion int             `json:"schema_version"`
	SkipReasons   map[string]int64 `json:"skip_reasons,omitempty"`
}

// ReadManifest loads the manifest.json from the given data directory.
func ReadManifest(dataDir string) (*Manifest, error) {
	path := filepath.Join(dataDir, manifestFile)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var m Manifest
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// WriteManifest serialises m to manifest.json inside dataDir.
func WriteManifest(dataDir string, m *Manifest) error {
	path := filepath.Join(dataDir, manifestFile)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}
