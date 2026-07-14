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

const evenOddSrc = `package input
func IsEven(n int) bool { if n == 0 { return true }; return IsOdd(n - 1) }
func IsOdd(n int) bool { if n == 0 { return false }; return IsEven(n - 1) }`

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
