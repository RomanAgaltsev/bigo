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

// TestParseKeyWithSpaces pins the third measured expressibility limit of the
// assumption format (2026-08-11 review F5), alongside the two campaign 1
// measured: interface methods and builtins cannot be stated at all.
// types.Func.FullName renders multiple type parameters comma-SPACE separated,
// so splitting at the first space made such a method unkeyable and blamed the
// bound for a defect in the key.
func TestParseKeyWithSpaces(t *testing.T) {
	es, err := ParseText("(*example.com/x.Pair[K, V]).Get O(1)\n")
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if len(es) != 1 {
		t.Fatalf("got %d entries, want 1", len(es))
	}
	if want := "(*example.com/x.Pair[K, V]).Get"; es[0].Key != want {
		t.Errorf("key = %q, want %q", es[0].Key, want)
	}
	if want := "O(1)"; es[0].Expr != want {
		t.Errorf("expr = %q, want %q", es[0].Expr, want)
	}
}

// A where-clause must survive the split, and a single-type-parameter key must
// keep working exactly as the shipped candidate files rely on.
func TestParseKeyWithSpacesAndWhereClause(t *testing.T) {
	es, err := ParseText("(*sync/atomic.Pointer[T]).Load O(1)\nstrings.Trim O(a*b) where a=len(s), b=len(cutset)\n")
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if len(es) != 2 {
		t.Fatalf("got %d entries, want 2", len(es))
	}
	if es[0].Key != "(*sync/atomic.Pointer[T]).Load" {
		t.Errorf("key[0] = %q", es[0].Key)
	}
	if es[1].Key != "strings.Trim" {
		t.Errorf("key[1] = %q", es[1].Key)
	}
	if want := "O(a*b) where a=len(s), b=len(cutset)"; es[1].Expr != want {
		t.Errorf("expr[1] = %q, want %q", es[1].Expr, want)
	}
}

// A line with no bound must still produce the shape error, naming the shape
// rather than a parse failure inside a bound that was never there.
func TestParseLineWithoutBound(t *testing.T) {
	_, err := ParseText("os.Getenv\n")
	if err == nil {
		t.Fatal("want an error for a line with no bound")
	}
	if !strings.Contains(err.Error(), "want '<key> O(<expr>)'") {
		t.Errorf("err = %v, want the shape error", err)
	}
}
