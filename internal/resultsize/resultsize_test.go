package resultsize_test

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/callsummary"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func boundOf(t *testing.T, src string) string {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if fn == nil {
		t.Fatal("f not found")
	}
	return engine.Infer(fn, callsummary.New(nil)).String()
}

func rangeOver(expr string) string {
	return `package input
import "strings"
func f(s string, n int) int {
	c := 0
	for range ` + expr + ` {
		c++
	}
	return c
}`
}

// The curated half: every splitter and transformer whose result length is
// bounded by its first argument.
func TestCuratedResultSizesFire(t *testing.T) {
	for _, expr := range []string{
		`strings.Split(s, " ")`,
		`strings.SplitN(s, " ", n)`,
		`strings.Fields(s)`,
	} {
		t.Run(expr, func(t *testing.T) {
			if got := boundOf(t, rangeOver(expr)); got != "O(len(s))" {
				t.Errorf("cost = %q, want O(len(s))", got)
			}
		})
	}
}

// TestUncuratedResultSizesStayTop is the no-fire half, and it is the one that
// keeps the table honest. Each absence below is a REASONED refusal, not an
// oversight — see the package doc.
func TestUncuratedResultSizesStayTop(t *testing.T) {
	tests := []struct{ name, src string }{
		// len(strings.Repeat(s, n)) is len(s)*n. An extent is ONE variable, so
		// naming either factor alone would be a wrong bound.
		{"strings.Repeat is a product", rangeOver(`strings.Repeat(s, n)`)},

		// len(strings.Join(xs, sep)) is the SUM of element lengths, which the
		// unit-element axiom says has no name.
		{"strings.Join is a sum", `package input
import "strings"
func f(xs []string, sep string) int {
	c := 0
	for range strings.Join(xs, sep) {
		c++
	}
	return c
}`},

		// A callback decides the result length.
		{"strings.FieldsFunc takes a callback", `package input
import "strings"
func f(s string, g func(rune) bool) int {
	c := 0
	for range strings.FieldsFunc(s, g) {
		c++
	}
	return c
}`},

		// The durable case: nothing curates a USER function's result, and
		// nothing ever will. This is the shape the analyzer's ParseUser fixture
		// pins end to end.
		{"a user function's result", `package input
func tok(s string) []string { return nil }
func f(s string) int {
	c := 0
	for range tok(s) {
		c++
	}
	return c
}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := boundOf(t, tt.src); got != "unverifiable" {
				t.Errorf("cost = %q, want unverifiable — this result length has no sound single-variable bound", got)
			}
		})
	}
}

// The two DEFERRALS from the package doc, pinned as no-fires so that neither
// silently starts working (or silently stays broken) without the reasoning
// being revisited.
func TestDeferredResultSizesStayTop(t *testing.T) {
	tests := []struct{ name, src string }{
		// A string result ranges via *ssa.Next, a path that never consults
		// lenExtent. The bound is true; the plumbing is absent.
		{"string-returning transformer", rangeOver(`strings.ToLower(s)`)},

		// Correct result size, no COST entry, so the row is ⊤ on the call.
		{"unpriced splitter", rangeOver(`strings.SplitAfter(s, " ")`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := boundOf(t, tt.src); got != "unverifiable" {
				t.Errorf("cost = %q, want unverifiable — this is a documented deferral; if it now works, update the package doc", got)
			}
		})
	}
}
