package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// Product is the minimal nutritional record stored per barcode.
type Product struct {
	Barcode  string
	Name     string
	Kcal100g float32
	Protein  float32
	Fat      float32
	Carbs    float32
}

const schemaVersion = 1

// Encode serialises a Product into a compact binary format:
//
//	version   uvarint  (=1)
//	nameLen   uvarint
//	name      []byte (UTF-8)
//	kcal100g  float32 LE  (NaN when missing)
//	protein   float32 LE
//	fat       float32 LE
//	carbs     float32 LE
func (p Product) Encode() []byte {
	var buf bytes.Buffer
	writeUvarint(&buf, schemaVersion)

	nameBytes := []byte(p.Name)
	writeUvarint(&buf, uint64(len(nameBytes)))
	buf.Write(nameBytes)

	writeFloat32LE(&buf, p.Kcal100g)
	writeFloat32LE(&buf, p.Protein)
	writeFloat32LE(&buf, p.Fat)
	writeFloat32LE(&buf, p.Carbs)

	return buf.Bytes()
}

// Decode parses a binary blob produced by Encode and sets fields on p.
// The Barcode field is NOT stored in the blob; the caller must set it.
func (p *Product) Decode(data []byte) error {
	r := bytes.NewReader(data)

	ver, err := binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("read version: %w", err)
	}
	if ver != schemaVersion {
		return fmt.Errorf("unsupported schema version %d", ver)
	}

	nameLen, err := binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("read name length: %w", err)
	}
	nameBuf := make([]byte, nameLen)
	if _, err := r.Read(nameBuf); err != nil {
		return fmt.Errorf("read name: %w", err)
	}
	p.Name = string(nameBuf)

	p.Kcal100g, err = readFloat32LE(r)
	if err != nil {
		return fmt.Errorf("read kcal: %w", err)
	}
	p.Protein, err = readFloat32LE(r)
	if err != nil {
		return fmt.Errorf("read protein: %w", err)
	}
	p.Fat, err = readFloat32LE(r)
	if err != nil {
		return fmt.Errorf("read fat: %w", err)
	}
	p.Carbs, err = readFloat32LE(r)
	if err != nil {
		return fmt.Errorf("read carbs: %w", err)
	}
	return nil
}

func writeUvarint(w *bytes.Buffer, v uint64) {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], v)
	w.Write(buf[:n])
}

func writeFloat32LE(w *bytes.Buffer, f float32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], math.Float32bits(f))
	w.Write(b[:])
}

func readFloat32LE(r *bytes.Reader) (float32, error) {
	var b [4]byte
	if _, err := r.Read(b[:]); err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(b[:])), nil
}

// NaNFloat32 is a sentinel for "value not available".
func NaNFloat32() float32 {
	return float32(math.NaN())
}
