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

func TestParseBigOExponentLimits(t *testing.T) {
	for _, in := range []string{"O(n^99999999999999999999)", "O(n^65)"} {
		if _, err := parseBigO(in); err == nil {
			t.Errorf("parseBigO(%q) expected error, got nil", in)
		}
	}
	if _, err := parseBigO("O(n^64)"); err != nil {
		t.Errorf("parseBigO(O(n^64)) unexpected error: %v", err)
	}
}

var _ = bound.One // keep import if unused above

func TestParseSizeTerms(t *testing.T) {
	// The report printer renders a length as len(x). Whatever it prints must
	// parse back, so a bound bigo reports can be pasted into a budget.
	tests := []struct {
		in   string
		want string
	}{
		{"O(len(s))", "O(len(s))"},
		{"O(cap(s))", "O(cap(s))"},
		{"O(log(len(s)))", "O(log(len(s)))"},
		{"O(len(s) log(len(s)))", "O(len(s) log(len(s)))"},
		{"O(len(a) len(b))", "O(len(a) len(b))"},
		{"O(len(s)^2)", "O(len(s)^2)"},
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

func TestParseSizeTermsRoundTrip(t *testing.T) {
	// The property that matters, asserted directly rather than via a literal.
	for _, in := range []string{"O(len(s))", "O(log(len(s)))", "O(len(s) log(len(s)))"} {
		b, err := parseBigO(in)
		if err != nil {
			t.Fatalf("parseBigO(%q) error: %v", in, err)
		}
		again, err := parseBigO(b.String())
		if err != nil {
			t.Errorf("printed bound %q does not parse: %v", b.String(), err)
			continue
		}
		if again.String() != b.String() {
			t.Errorf("round trip changed the bound: %q then %q", b.String(), again.String())
		}
	}
}

func TestParseSizeTermsRejectsMalformed(t *testing.T) {
	bad := []string{"O(len())", "O(len(s)", "O(len(1))", "O(len(s)))", "O(cap())"}
	for _, in := range bad {
		if _, err := parseBigO(in); err == nil {
			t.Errorf("parseBigO(%q) expected error, got nil", in)
		}
	}
	// A bare identifier named len or cap is still a plain variable: the size
	// term is recognised only when a parenthesis follows.
	if got, err := parseBigO("O(len)"); err != nil {
		t.Errorf("parseBigO(%q) expected a plain variable, got error: %v", "O(len)", err)
	} else if got.String() != "O(len)" {
		t.Errorf("parseBigO(%q) = %q, want %q", "O(len)", got.String(), "O(len)")
	}
}
