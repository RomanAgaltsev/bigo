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
