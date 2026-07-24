package assume

import (
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	in := "# comment\n\nstrings.Trim O(a*b) where a=len(s), b=len(cutset)\n(*sync.Once).Do O(1)\n"
	es, err := parse("t.assume", strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 2 {
		t.Fatalf("entries = %d, want 2", len(es))
	}
	if es[0].Key != "strings.Trim" || es[0].Expr != "O(a*b) where a=len(s), b=len(cutset)" || es[0].Line != 3 {
		t.Errorf("entry 0 = %+v", es[0])
	}
	if es[1].Key != "(*sync.Once).Do" || es[1].Expr != "O(1)" {
		t.Errorf("entry 1 = %+v", es[1])
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct{ name, in, wantSub string }{
		{"missing bound", "strings.Trim\n", "line 1"},
		{"malformed bound", "strings.Trim O(((\n", "line 1"},
		{"duplicate key", "a.F O(1)\na.F O(n)\n", "duplicate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parse("t.assume", strings.NewReader(tt.in))
			if err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantSub)
			}
		})
	}
}
