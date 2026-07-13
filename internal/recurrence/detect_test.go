package recurrence

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/callsummary"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func extractOf(t *testing.T, src string) (rec, bool) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	return extract(ssasupport.Func(pkg, "f"), callsummary.New(nil))
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
