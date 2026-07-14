package recurrence

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func extractOf(t *testing.T, src string) (rec, bool) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	return extract(ssasupport.Func(pkg, "f"), stubModel{})
}

func TestExtractSliceSubtractive(t *testing.T) {
	r, ok := extractOf(t, `package input
func f(xs []int) int { if len(xs) == 0 { return 0 }; return xs[0] + f(xs[1:]) }`)
	if !ok || len(r.terms) != 1 || r.terms[0].kind != stepSub || string(r.measure) != "len(xs)" {
		t.Fatalf("extract = %+v, %v", r, ok)
	}
}

func TestExtractSliceHalving(t *testing.T) {
	r, ok := extractOf(t, `package input
func f(xs []int, t int) int {
	if len(xs) == 0 { return -1 }
	m := len(xs) / 2
	if xs[m] == t { return m }
	if xs[m] < t { return f(xs[m+1:], t) }
	return f(xs[:m], t)
}`)
	if !ok || len(r.terms) != 2 {
		t.Fatalf("extract = %+v, %v", r, ok)
	}
	for _, tm := range r.terms {
		if tm.kind != stepDiv || tm.div != 2 {
			t.Errorf("term = %+v, want Div/2", tm)
		}
	}
}

func TestExtractUnguardedIntegerRejected(t *testing.T) {
	// No base guard: f(n-1) with no dominating `n > c` — must NOT extract.
	_, ok := extractOf(t, `package input
func f(n int) int { return f(n - 1) }`)
	if ok {
		t.Error("unguarded integer recursion must be rejected (may not terminate)")
	}
}

func TestExtractGuardedInteger(t *testing.T) {
	_, ok := extractOf(t, `package input
func f(n int) int { if n <= 0 { return 0 }; return 1 + f(n-1) }`)
	if !ok {
		t.Error("guarded integer recursion must extract")
	}
}

func TestExtractDivisiveNonPositiveFloorRejected(t *testing.T) {
	// n/2 guarded only by n>=0: 0/2==0 is a fixed point -> non-terminating.
	_, ok := extractOf(t, `package input
func f(n int) int { if n >= 0 { return f(n / 2) }; return 0 }`)
	if ok {
		t.Error("divisive recursion with a 0 fixed point (n>=0 guard) must be rejected")
	}
}

func TestExtractDivisiveNegativeFloorRejected(t *testing.T) {
	// n/2 guarded by n>-5: still infinite at n=0.
	_, ok := extractOf(t, `package input
func f(n int) int { if n > -5 { return f(n / 2) }; return 0 }`)
	if ok {
		t.Error("divisive recursion guarded by a negative floor must be rejected")
	}
}

func TestExtractDivisiveGuardedPositive(t *testing.T) {
	// n>0 => n>=1: the divisive step reaches the n<=0 base. Must extract.
	r, ok := extractOf(t, `package input
func f(n int) int { if n > 0 { return f(n / 2) }; return 0 }`)
	if !ok || len(r.terms) != 1 || r.terms[0].kind != stepDiv {
		t.Fatalf("n>0-guarded divisive recursion must extract: %+v, %v", r, ok)
	}
}

func TestExtractDivisiveSliceNoBaseRejected(t *testing.T) {
	// xs[:len/2] with no base: xs[:0] stays empty, a fixed point with no panic.
	_, ok := extractOf(t, `package input
func f(xs []int) int { m := len(xs) / 2; return f(xs[:m]) }`)
	if ok {
		t.Error("divisive slice recursion with no empty-slice base must be rejected")
	}
}

func TestExtractDivisiveSliceWithBase(t *testing.T) {
	// A len==0 base keeps the recursing side at len>=1. Must extract.
	r, ok := extractOf(t, `package input
func f(xs []int) int { if len(xs) == 0 { return 0 }; m := len(xs) / 2; return f(xs[:m]) }`)
	if !ok || len(r.terms) != 1 || r.terms[0].kind != stepDiv {
		t.Fatalf("len==0-based divisive slice must extract: %+v, %v", r, ok)
	}
}

func TestExtractGrowingArgRejected(t *testing.T) {
	_, ok := extractOf(t, `package input
func f(n int) int { if n > 1000 { return 0 }; return f(n + 1) }`)
	if ok {
		t.Error("growing argument is not a decrease")
	}
}

func TestExtractSelfCallUnderLoopRejected(t *testing.T) {
	_, ok := extractOf(t, `package input
func f(xs []int) int { s := 0; for range xs { s += f(xs[1:]) }; return s }`)
	if ok {
		t.Error("self-call under a size loop is not constant multiplicity")
	}
}
