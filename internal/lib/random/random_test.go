package random

import (
	"strings"
	"testing"
)

func TestGenerateString(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{
			"size 0",
			0,
		},
		{
			"size 1",
			1,
		},
		{
			"size 99",
			99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := GenerateString(tt.size)
			if err != nil {
				t.Fatalf("failed to generate string with size %d: %v", tt.size, err)
			}
			if len(res) != tt.size {
				t.Errorf("want %d, got %d", tt.size, len(res))
			}
		})
	}
}

func TestGenerateString_Unique(t *testing.T) {
	seen := make(map[string]struct{})
	const n = 1000
	const size = 10

	for i := 0; i < n; i++ {
		s, err := GenerateString(size)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := seen[s]; ok {
			t.Fatalf("duplicate string generated: %q", s)
		}
		seen[s] = struct{}{}
	}
}

func TestGenerateString_ValidChars(t *testing.T) {
	s, err := GenerateString(50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range s {
		if !strings.ContainsRune(idAlphabet, c) {
			t.Errorf("unexpected character %q in generated string", c)
		}
	}
}
