package recurrence

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/callsummary"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func selfRecursive(t *testing.T, src, name string) bool {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	return IsSelfRecursive(ssasupport.Func(pkg, name))
}

func TestIsSelfRecursive(t *testing.T) {
	yes := `package input
func f(n int) int { if n <= 0 { return 0 }; return 1 + f(n-1) }`
	no := `package input
func f(n int) int { return n + 1 }`
	if !selfRecursive(t, yes, "f") {
		t.Error("f is self-recursive")
	}
	if selfRecursive(t, no, "f") {
		t.Error("f is not self-recursive")
	}
}

func TestLocalWorkTreatsSelfCallAsConstant(t *testing.T) {
	// A recursive call inside an O(len(xs)) loop, plus a self-call: localWork
	// must cost the self-call as O(1), yielding O(len(xs)) for the level work.
	src := `package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ { s += xs[i] }
	if len(xs) == 0 { return s }
	return s + f(xs[1:])
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	w, ok := localWork(fn, callsummary.New(nil))
	if !ok || w.String() != "O(len(xs))" {
		t.Errorf("localWork = (%q, %v), want (O(len(xs)), true)", w.String(), ok)
	}
}
