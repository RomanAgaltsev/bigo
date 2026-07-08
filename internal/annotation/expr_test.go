package annotation

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

func TestParseBigO(t *testing.T) {
	tests := []struct {
		in   string
		want string // bound.String()
	}{
		{"O(1)", "O(1)"},
		{"O(n)", "O(n)"},
		{"O(n^2)", "O(n^2)"},
		{"O(n log n)", "O(n log(n))"},
		{"O(n*m)", "O(m n)"},
		{"O(n * m)", "O(m n)"},
		{"O(log(n))", "O(log(n))"},
		{"O(n^2 + n)", "O(n^2)"}, // reduces
		{"O(n*m + n^2)", "O(m n + n^2)"},
		{"O(n log(n)^2)", "O(n log(n)^2)"},
	}
	for _, tt := range tests {
		got, err := parseBigO(tt.in)
		if err != nil {
			t.Errorf("parseBigO(%q) error: %v", tt.in, err)
			continue
		}
		if got.String() != tt.want {
			t.Errorf("parseBigO(%q) = %q, want %q", tt.in, got.String(), tt.want)
		}
	}
}

func TestParseBigOErrors(t *testing.T) {
	bad := []string{"O(", "O()", "n", "O(n))", "O(2n)", "O(n +)", "O(#)", "O(n) x"}
	for _, in := range bad {
		if _, err := parseBigO(in); err == nil {
			t.Errorf("parseBigO(%q) expected error, got nil", in)
		}
	}
}

var _ = bound.One // keep import if unused above
