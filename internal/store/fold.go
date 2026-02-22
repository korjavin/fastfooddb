package store

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// stripCombiningMarks is a transform.Transformer that removes Unicode
// combining marks (category M) after NFD decomposition.
type stripCombining struct{ transform.NopResetter }

func (stripCombining) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		r, size := utf8.DecodeRune(src[nSrc:])
		if unicode.Is(unicode.M, r) {
			nSrc += size
			continue
		}
		if nDst+size > len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}
		copy(dst[nDst:], src[nSrc:nSrc+size])
		nDst += size
		nSrc += size
	}
	return nDst, nSrc, nil
}

// FoldName normalises a product name for indexing and querying:
//  1. Lowercase
//  2. ß → ss
//  3. NFD decomposition, then strip combining marks (removes accents)
//  4. Replace non-letter/non-digit with space
//  5. Collapse runs of spaces, trim
func FoldName(s string) string {
	// 1. Lowercase
	s = strings.ToLower(s)

	// 2. ß → ss (must happen before NFD, otherwise ß might decompose oddly)
	s = strings.ReplaceAll(s, "ß", "ss")

	// 3. NFD + strip combining marks
	t := transform.Chain(norm.NFD, stripCombining{}, norm.NFC)
	result, _, err := transform.String(t, s)
	if err != nil {
		result = s // fallback: use as-is
	}

	// 4. Replace non-letter/non-digit with space
	var sb strings.Builder
	sb.Grow(len(result))
	for _, r := range result {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		} else {
			sb.WriteByte(' ')
		}
	}
	cleaned := sb.String()

	// 5. Collapse spaces and trim
	fields := strings.Fields(cleaned)
	return strings.Join(fields, " ")
}
