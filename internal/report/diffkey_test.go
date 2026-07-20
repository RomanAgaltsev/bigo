package report

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// twoInits builds a package declaring two init functions — legal, idiomatic Go
// — one bounded and one unverifiable. This is the shape that collapsed under
// the old package.func key: both mapped to "example.com/m.init", the map kept
// one, and the other was then compared against the wrong partner.
func twoInits() []Function {
	return []Function{
		{
			Package: "example.com/m", Func: "init", File: "a.go", Line: 10,
			Time: boundJSON(bound.Of(bound.Term("n"))),
		},
		{
			Package: "example.com/m", Func: "init", File: "b.go", Line: 20,
			Time: BoundJSON{Top: true},
			Causes: []CauseJSON{
				{Kind: "call", Detail: "unresolved cost at call to x"},
			},
		},
	}
}

// TestDiffIdenticalDocumentIsSilentWithDuplicateNames is the regression test
// for the shipped CI gate. Diffing a document against ITSELF must report
// nothing, whatever names it contains — that property is what `-fail-on
// regression` rests on, and multiple init()s per package broke it.
func TestDiffIdenticalDocumentIsSilentWithDuplicateNames(t *testing.T) {
	d := doc(twoInits()...)
	findings, _, err := Diff(d, d)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("Diff(d, d) = %d findings, want 0 — a document cannot differ from itself", len(findings))
		for _, f := range findings {
			t.Logf("  spurious: %s %s: %s", f.Key, f.File, f.Message)
		}
	}
}

// TestDiffDuplicateNamesDoNotMaskARegression is the other half, and the more
// dangerous one: a real regression in one init must not be hidden by a bounded
// sibling in the same package.
func TestDiffDuplicateNamesDoNotMaskARegression(t *testing.T) {
	base := doc(twoInits()...)
	head := doc(twoInits()...)
	// a.go's init regresses from O(n) to unverifiable.
	head.Functions[0].Time = BoundJSON{Top: true}
	head.Functions[0].Causes = []CauseJSON{{Kind: "loop", Detail: "loop with unrecognized trip count"}}

	findings, _, err := Diff(base, head)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("Diff = %d findings, want exactly 1 (a.go's init regressed)", len(findings))
	}
	if findings[0].File != "a.go" {
		t.Errorf("finding is on %s:%d, want a.go — the regression was attributed to the wrong init",
			findings[0].File, findings[0].Line)
	}
}

// TestDiffKeyIsStableForOrdinaryFunctions guards the property the fix must NOT
// break: a uniquely-named function's identity must not depend on its position,
// or inserting a blank line above it would read as remove-plus-add.
func TestDiffKeyIsStableForOrdinaryFunctions(t *testing.T) {
	fn := Function{
		Package: "example.com/m", Func: "Handle", File: "h.go", Line: 12,
		Time: boundJSON(bound.Of(bound.Term("n"))),
	}
	base := doc(fn)
	moved := fn
	moved.Line = 400 // the function moved down the file; nothing else changed
	head := doc(moved)

	findings, _, err := Diff(base, head)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("Diff after a pure line move = %d findings, want 0 — identity must not be positional", len(findings))
	}
}
