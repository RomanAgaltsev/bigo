package smell

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func TestParseRules(t *testing.T) {
	cases := []struct {
		in      string
		wantLen int
		wantErr bool
	}{
		{"all", 8, false},
		{"none", 0, false},
		{"SM1,SM4", 2, false},
		{"SM9", 0, true},
		{"", 8, false},
	}
	for _, c := range cases {
		got, err := ParseRules(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("ParseRules(%q) err = %v, wantErr %v", c.in, err, c.wantErr)
			continue
		}
		if !c.wantErr && len(got) != c.wantLen {
			t.Errorf("ParseRules(%q) = %d rules, want %d (%v)", c.in, len(got), c.wantLen, got)
		}
	}
}

// TestDetectCleanFunctionNoFindings is the zero-spray smoke test: a clean
// function with all rules enabled must produce no findings.
func TestDetectCleanFunctionNoFindings(t *testing.T) {
	src := `package input
func Clean(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	enabled, err := ParseRules("all")
	if err != nil {
		t.Fatal(err)
	}
	got := Detect(ssasupport.Func(pkg, "Clean"), enabled)
	if len(got) != 0 {
		t.Errorf("Clean produced %d findings, want 0: %+v", len(got), got)
	}
}
