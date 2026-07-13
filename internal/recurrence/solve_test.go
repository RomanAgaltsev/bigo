package recurrence

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/size"
)

func TestSolveSubtractive(t *testing.T) {
	n := size.Len("xs")
	// a=1, work O(1) -> O(len(xs))
	r := rec{measure: n, terms: []sizeStep{{kind: stepSub, sub: 1}}, work: bound.Constant()}
	got, ok := solveSubtractive(r)
	if !ok || got.String() != "O(len(xs))" {
		t.Errorf("a=1,f=O(1): got (%q,%v), want O(len(xs))", got.String(), ok)
	}
	// a=1, work O(len(xs)) -> O(len(xs)^2)
	r.work = bound.Of(bound.Term(n))
	got, ok = solveSubtractive(r)
	if !ok || got.String() != "O(len(xs)^2)" {
		t.Errorf("a=1,f=O(n): got (%q,%v), want O(len(xs)^2)", got.String(), ok)
	}
	// a=2 subtractive -> exponential -> ⊤
	r2 := rec{measure: n, terms: []sizeStep{{kind: stepSub, sub: 1}, {kind: stepSub, sub: 2}}, work: bound.Constant()}
	if _, ok := solveSubtractive(r2); ok {
		t.Error("a=2 subtractive must be ⊤ (exponential)")
	}
}
