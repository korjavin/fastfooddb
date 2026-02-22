package store

import "testing"

func TestFoldName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Basic lowercase
		{"Hello World", "hello world"},
		// ß → ss
		{"ß", "ss"},
		{"Grüße", "grusse"},
		// Accents stripped
		{"café", "cafe"},
		{"naïve", "naive"},
		{"Crème Brûlée", "creme brulee"},
		// Mixed diacritics
		{"über", "uber"},
		{"Ångström", "angstrom"},
		// Non-letter/digit → space, collapsed
		{"foo--bar", "foo bar"},
		{"  lots   of   spaces  ", "lots of spaces"},
		// Digits preserved
		{"Vitamin B12", "vitamin b12"},
		// Empty string
		{"", ""},
		// Already clean
		{"apple juice", "apple juice"},
	}

	for _, tc := range tests {
		got := FoldName(tc.input)
		if got != tc.want {
			t.Errorf("FoldName(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}
