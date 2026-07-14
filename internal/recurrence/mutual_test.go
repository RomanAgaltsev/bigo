package recurrence

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// The mutual-recursion unit tests use stubModel (costmodel_test.go), not
// callsummary: callsummary imports recurrence, so a callsummary import here
// would be a cycle. stubModel costs every call O(1), which is neutral for these
// functions — cycle calls are held O(1) by constFor anyway, and level work comes
// from loops/indexing, not calls.

// evenOddSrc uses `n <= 0` (not `n == 0`): with an `== 0` base the cycle never
// terminates for negative int (IsEven(-1)→IsOdd(-2)→…), which the engine
// soundly keeps ⊤. `n <= 0` bases the recursion for all int, so it graduates.
const evenOddSrc = `package input
func IsEven(n int) bool { if n <= 0 { return true }; return IsOdd(n - 1) }
func IsOdd(n int) bool { if n <= 0 { return false }; return IsEven(n - 1) }`

func TestLocalWorkExcludingPartner(t *testing.T) {
	pkg, _, err := ssasupport.Build(evenOddSrc)
	if err != nil {
		t.Fatal(err)
	}
	even := ssasupport.Func(pkg, "IsEven")
	odd := ssasupport.Func(pkg, "IsOdd")
	w, ok := localWorkExcluding(even, stubModel{}, even, odd)
	if !ok || w.String() != "O(1)" {
		t.Errorf("localWorkExcluding = (%q, %v), want (O(1), true)", w.String(), ok)
	}
}

func partnerOf(t *testing.T, src, name string) (string, bool) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := MutualPartner(ssasupport.Func(pkg, name))
	if !ok {
		return "", false
	}
	return p.Name(), true
}

func TestMutualPartnerEvenOdd(t *testing.T) {
	if p, ok := partnerOf(t, evenOddSrc, "IsEven"); !ok || p != "IsOdd" {
		t.Errorf("partner = (%q,%v), want IsOdd", p, ok)
	}
}

func TestMutualPartnerRejectsSelfRecursiveMember(t *testing.T) {
	src := `package input
func A(n int) int { if n <= 0 { return 0 }; return A(n-1) + B(n-1) }
func B(n int) int { if n <= 0 { return 0 }; return A(n - 1) }`
	if _, ok := partnerOf(t, src, "B"); ok {
		t.Error("A self-recurses: multi-cycle SCC must be rejected")
	}
}

func TestMutualPartnerRejectsTwoPartners(t *testing.T) {
	src := `package input
func A(n int) int { if n <= 0 { return 0 }; return B(n-1) + C(n-1) }
func B(n int) int { if n <= 0 { return 0 }; return A(n - 1) }
func C(n int) int { if n <= 0 { return 0 }; return A(n - 1) }`
	if _, ok := partnerOf(t, src, "A"); ok {
		t.Error("A cycles with both B and C: ambiguous, must be rejected")
	}
}

func TestMutualPartnerRejectsThreeCycle(t *testing.T) {
	src := `package input
func A(n int) int { if n <= 0 { return 0 }; return B(n - 1) }
func B(n int) int { if n <= 0 { return 0 }; return C(n - 1) }
func C(n int) int { if n <= 0 { return 0 }; return A(n - 1) }`
	if _, ok := partnerOf(t, src, "A"); ok {
		t.Error("A->B->C->A is a 3-cycle, out of scope")
	}
}

func pairOf(t *testing.T, src, name string) (rec, bool) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	a := ssasupport.Func(pkg, name)
	b, ok := MutualPartner(a)
	if !ok {
		t.Fatalf("no partner for %s", name)
	}
	return extractPair(a, b, stubModel{})
}

func TestExtractPairEvenOdd(t *testing.T) {
	r, ok := pairOf(t, evenOddSrc, "IsEven")
	if !ok {
		t.Fatal("even/odd pair must extract")
	}
	// Composed cycle: Sub(1) then Sub(1) => one term Sub(2).
	if len(r.terms) != 1 || r.terms[0].kind != stepSub || r.terms[0].sub != 2 {
		t.Errorf("terms = %+v, want [Sub 2]", r.terms)
	}
	if string(r.measure) != "n" {
		t.Errorf("measure = %q, want n", r.measure)
	}
}

func TestExtractPairIdentityEdge(t *testing.T) {
	// One edge passes the measure through unchanged; the other decreases.
	src := `package input
func Ping(n int) int { if n <= 0 { return 0 }; return Pong(n) }
func Pong(n int) int { return Ping(n - 1) }`
	r, ok := pairOf(t, src, "Ping")
	if !ok || len(r.terms) != 1 || r.terms[0].kind != stepSub || r.terms[0].sub != 1 {
		t.Fatalf("identity∘Sub(1) must compose to Sub(1); got %+v, %v", r, ok)
	}
}

func TestExtractPairGrowingEdgeRejected(t *testing.T) {
	src := `package input
func A(n int) int { if n <= 0 { return 0 }; return B(n + 1) }
func B(n int) int { return A(n - 2) }`
	if _, ok := pairOf(t, src, "A"); ok {
		t.Error("growing edge must be rejected")
	}
}

func TestExtractPairBothIdentityRejected(t *testing.T) {
	src := `package input
func A(n int) int { if n <= 0 { return 0 }; return B(n) }
func B(n int) int { return A(n) }`
	if _, ok := pairOf(t, src, "A"); ok {
		t.Error("no strict edge: cycle never decreases, must be rejected")
	}
}

func TestExtractPairMixedSubDivRejected(t *testing.T) {
	src := `package input
func A(n int) int { if n > 0 { return B(n - 1) }; return 0 }
func B(n int) int { return A(n / 2) }`
	if _, ok := pairOf(t, src, "A"); ok {
		t.Error("mixed Sub∘Div cycle is out of scope (spec §4.2), must be rejected")
	}
}

func TestExtractPairDivisiveGEZeroRejected(t *testing.T) {
	// The F1 class through the mutual path: n>=0 floor does NOT prove >=1.
	src := `package input
func A(n int) int { if n >= 0 { return B(n) }; return 0 }
func B(n int) int { return A(n / 2) }`
	if _, ok := pairOf(t, src, "A"); ok {
		t.Error("divisive cycle guarded only by n>=0 must be rejected (fixed point at 0)")
	}
}

func TestExtractPairDivisiveGuardedAccepted(t *testing.T) {
	src := `package input
func A(n int) int { if n > 0 { return B(n) }; return 0 }
func B(n int) int { return A(n / 2) }`
	r, ok := pairOf(t, src, "A")
	if !ok || len(r.terms) != 1 || r.terms[0].kind != stepDiv || r.terms[0].div != 2 {
		t.Fatalf("n>0-guarded divisive cycle must extract as Div(2); got %+v, %v", r, ok)
	}
}

func TestExtractPairDivisiveSliceNoBaseRejected(t *testing.T) {
	src := `package input
func A(xs []int) int { return B(xs) }
func B(xs []int) int { return A(xs[:len(xs)/2]) }`
	if _, ok := pairOf(t, src, "A"); ok {
		t.Error("divisive slice cycle with no empty-slice base must be rejected")
	}
}

func TestExtractPairMeasureNotThreadedRejected(t *testing.T) {
	src := `package input
var other []int
func A(n int) int { if n <= 0 { return 0 }; return B(len(other)) }
func B(m int) int { return A(m - 1) }`
	if _, ok := pairOf(t, src, "A"); ok {
		t.Error("A's argument to B is not derived from A's measure: no threading")
	}
}
