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

func TestLogBase(t *testing.T) {
	cases := []struct {
		a, b, want int
		ok         bool
	}{{1, 2, 0, true}, {2, 2, 1, true}, {4, 2, 2, true}, {8, 2, 3, true}, {2, 4, 0, false}, {3, 2, 0, false}}
	for _, c := range cases {
		got, ok := logBase(c.a, c.b)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("logBase(%d,%d) = (%d,%v), want (%d,%v)", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}

func TestSolveMaster(t *testing.T) {
	n := size.Len("xs")
	lin := bound.Of(bound.Term(n)) // O(n)
	con := bound.Constant()        // O(1)
	cases := []struct {
		name string
		a, b int
		work bound.Bound
		want string
	}{
		{"binary search T(n/2)+O(1)", 1, 2, con, "O(log(len(xs)))"},
		{"merge sort 2T(n/2)+O(n)", 2, 2, lin, "O(len(xs) log(len(xs)))"},
		{"tree recursion 2T(n/2)+O(1)", 2, 2, con, "O(len(xs))"},
	}
	for _, c := range cases {
		got, ok := solveMaster(c.a, c.b, c.work, n)
		if !ok || got.String() != c.want {
			t.Errorf("%s: got (%q,%v), want %q", c.name, got.String(), ok, c.want)
		}
	}
	// non-integer exponent -> ⊤
	if _, ok := solveMaster(2, 4, con, n); ok {
		t.Error("2T(n/4) exponent 1/2 must be ⊤")
	}
}

func TestSolveAkraBazzi(t *testing.T) {
	n := size.Len("xs")
	// T(n)=T(n/3)+T(2n/3)+O(n): Σ b^-p: 3^-1 + (3/2)^-1 = 1/3+2/3 = 1 -> p=1, d=1 -> O(n log n)
	got, ok := solveAkraBazzi([]ratio{{1, 3}, {1, 1.5}}, bound.Of(bound.Term(n)), n)
	if !ok || got.String() != "O(len(xs) log(len(xs)))" {
		t.Errorf("got (%q,%v), want O(len(xs) log(len(xs)))", got.String(), ok)
	}
	// T(n)=T(n/2)+2T(n/4)+O(1): 2^-1 + 2·4^-1 = 1/2+1/2 = 1 -> p=1, d=0 -> O(n)
	got, ok = solveAkraBazzi([]ratio{{1, 2}, {2, 4}}, bound.Constant(), n)
	if !ok || got.String() != "O(len(xs))" {
		t.Errorf("mixed integer divisors: got (%q,%v), want O(len(xs))", got.String(), ok)
	}
}

func TestSolveAkraBazziNoIntegerP(t *testing.T) {
	n := size.Len("xs")
	if _, ok := solveAkraBazzi([]ratio{{1, 3}, {1, 5}}, bound.Constant(), n); ok {
		t.Error("no small integer p solves 3^-p+5^-p=1 -> ⊤")
	}
}

func TestRatiosOf(t *testing.T) {
	rs := ratiosOf([]sizeStep{{kind: stepDiv, div: 2}, {kind: stepDiv, div: 4}, {kind: stepDiv, div: 4}})
	if len(rs) != 2 || rs[0] != (ratio{1, 2}) || rs[1] != (ratio{2, 4}) {
		t.Errorf("ratiosOf = %+v, want [{1 2} {2 4}]", rs)
	}
}
