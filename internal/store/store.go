package store

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/simple"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/cockroachdb/pebble"
)

const (
	pebbleDir = "pebble"
	bleveDir  = "bleve"
)

// bleveDoc is the document structure indexed into Bleve.
type bleveDoc struct {
	NameFolded string `json:"name_folded"`
}

// Store wraps a Pebble KV store and a Bleve full-text index.
type Store struct {
	db    *pebble.DB
	index bleve.Index
}

// OpenReadOnly opens an existing data directory in read-only mode (for the server).
func OpenReadOnly(dataDir string) (*Store, error) {
	db, err := pebble.Open(filepath.Join(dataDir, pebbleDir), &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open pebble (read-only): %w", err)
	}

	idx, err := bleve.Open(filepath.Join(dataDir, bleveDir))
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open bleve index: %w", err)
	}

	return &Store{db: db, index: idx}, nil
}

// Create initialises a fresh data directory for the importer.
// The pebble and bleve sub-directories must not already exist.
func Create(dataDir string) (*Store, error) {
	db, err := pebble.Open(filepath.Join(dataDir, pebbleDir), &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("create pebble: %w", err)
	}

	idx, err := bleve.New(filepath.Join(dataDir, bleveDir), newBleveMapping())
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create bleve index: %w", err)
	}

	return &Store{db: db, index: idx}, nil
}

// Close releases all resources held by the store.
func (s *Store) Close() error {
	var errs []string
	if err := s.index.Close(); err != nil {
		errs = append(errs, "bleve: "+err.Error())
	}
	if err := s.db.Close(); err != nil {
		errs = append(errs, "pebble: "+err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("store close: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Put writes a product to Pebble and indexes its folded name in Bleve.
// If the name is empty the product is stored in Pebble but not indexed.
func (s *Store) Put(p Product) error {
	if p.Barcode == "" {
		return fmt.Errorf("product has empty barcode")
	}

	encoded := p.Encode()
	if err := s.db.Set([]byte(p.Barcode), encoded, pebble.NoSync); err != nil {
		return fmt.Errorf("pebble set: %w", err)
	}

	if p.Name != "" {
		doc := bleveDoc{NameFolded: FoldName(p.Name)}
		if err := s.index.Index(p.Barcode, doc); err != nil {
			return fmt.Errorf("bleve index: %w", err)
		}
	}
	return nil
}

// WriteBatch accumulates products for batched writes to Pebble and Bleve.
type WriteBatch struct {
	s     *Store
	pb    *pebble.Batch
	bb    *bleve.Batch
	count int
}

// NewWriteBatch creates a new WriteBatch backed by the given store.
func (s *Store) NewWriteBatch() *WriteBatch {
	return &WriteBatch{
		s:  s,
		pb: s.db.NewBatch(),
		bb: s.index.NewBatch(),
	}
}

// Put accumulates a product in the batch without flushing.
func (b *WriteBatch) Put(p Product) {
	encoded := p.Encode()
	_ = b.pb.Set([]byte(p.Barcode), encoded, pebble.NoSync)
	if p.Name != "" {
		doc := bleveDoc{NameFolded: FoldName(p.Name)}
		_ = b.bb.Index(p.Barcode, doc)
	}
	b.count++
}

// Flush commits both batches to the underlying stores and resets accumulators.
func (b *WriteBatch) Flush() error {
	if err := b.pb.Commit(pebble.NoSync); err != nil {
		return fmt.Errorf("pebble batch commit: %w", err)
	}
	if err := b.s.index.Batch(b.bb); err != nil {
		return fmt.Errorf("bleve batch commit: %w", err)
	}
	b.pb.Reset()
	b.bb = b.s.index.NewBatch()
	b.count = 0
	return nil
}

// Close flushes any pending data and releases the pebble batch memory.
func (b *WriteBatch) Close() error {
	if b.count > 0 {
		if err := b.Flush(); err != nil {
			b.pb.Close()
			return err
		}
	}
	b.pb.Close()
	return nil
}

// Len returns the number of records accumulated since the last flush.
func (b *WriteBatch) Len() int {
	return b.count
}

// Get retrieves a product by barcode from Pebble.
// Returns (Product, false, nil) when the barcode is not found.
func (s *Store) Get(barcode string) (Product, bool, error) {
	val, closer, err := s.db.Get([]byte(barcode))
	if err == pebble.ErrNotFound {
		return Product{}, false, nil
	}
	if err != nil {
		return Product{}, false, fmt.Errorf("pebble get: %w", err)
	}
	defer closer.Close()

	// val is only valid until closer.Close(); copy it
	data := make([]byte, len(val))
	copy(data, val)

	var p Product
	if err := p.Decode(data); err != nil {
		return Product{}, false, fmt.Errorf("decode product: %w", err)
	}
	p.Barcode = barcode
	return p, true, nil
}

// Search runs a Bleve query and fetches the matching products from Pebble.
// limit caps the number of results (max 100).
func (s *Store) Search(q string, limit int) ([]Product, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	folded := FoldName(q)
	if folded == "" {
		return nil, nil
	}

	boolQ := bleve.NewBooleanQuery()

	// Stage A – exact / prefix (high boost)
	phraseQ := bleve.NewMatchPhraseQuery(folded)
	phraseQ.SetField("name_folded")
	phraseQ.SetBoost(10)
	boolQ.AddShould(phraseQ)

	prefixQ := bleve.NewPrefixQuery(folded)
	prefixQ.SetField("name_folded")
	prefixQ.SetBoost(5)
	boolQ.AddShould(prefixQ)

	// Stage B – per-token fuzzy (only for tokens ≥4 chars)
	for _, token := range strings.Fields(folded) {
		if len(token) < 4 {
			continue
		}
		fuzz := 1
		if len(token) >= 8 {
			fuzz = 2
		}
		fuzzyQ := bleve.NewFuzzyQuery(token)
		fuzzyQ.SetField("name_folded")
		fuzzyQ.Fuzziness = fuzz
		boolQ.AddShould(fuzzyQ)
	}

	req := bleve.NewSearchRequestOptions(boolQ, limit, 0, false)
	res, err := s.index.Search(req)
	if err != nil {
		return nil, fmt.Errorf("bleve search: %w", err)
	}

	products := make([]Product, 0, len(res.Hits))
	for _, hit := range res.Hits {
		p, found, err := s.Get(hit.ID)
		if err != nil || !found {
			continue
		}
		products = append(products, p)
	}
	return products, nil
}

// newBleveMapping builds the index mapping used when creating a fresh index.
func newBleveMapping() mapping.IndexMapping {
	im := bleve.NewIndexMapping()

	textField := bleve.NewTextFieldMapping()
	textField.Analyzer = simple.Name
	textField.Store = false

	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("name_folded", textField)

	im.DefaultMapping = docMapping
	return im
}
