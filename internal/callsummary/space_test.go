package callsummary

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func TestSpaceResolverInterproc(t *testing.T) {
	src := `package input
func alloc(n int) []int { return make([]int, n) }
func f(m int) []int { return alloc(m) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	sp, _ := engine.InferSpace(fn, NewSpace(nil))
	if sp.Heap.String() != "O(m)" {
		t.Errorf("heap = %q, want O(m)", sp.Heap.String())
	}
}

// TestSpaceStackTerm pins that a recursive, non-allocating function is all-stack:
// heap O(1) (self-call held constant, no allocation), stack O(len(xs)) from the
// recurrence depth. SpaceOf, not engine.InferSpace, injects the stack term
// because engine must not import recurrence (the plan's import-direction rule).
func TestSpaceStackTerm(t *testing.T) {
	src := `package input
func f(xs []int) int { if len(xs)==0 {return 0}; return xs[0]+f(xs[1:]) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	sp, _ := NewSpace(nil).SpaceOf(fn, New(nil))
	if sp.Stack.String() != "O(len(xs))" || sp.Heap.String() != "O(1)" {
		t.Errorf("space = {heap %q, stack %q}, want {O(1), O(len(xs))}", sp.Heap.String(), sp.Stack.String())
	}
}

// TestSpaceInterprocRecursiveCallee pins the conservative interprocedural rule:
// a non-recursive caller of a self-recursive helper inherits the helper's stack
// depth into its (Within-only) heap channel, never into its Exceeds-driving Stack.
func TestSpaceInterprocRecursiveCallee(t *testing.T) {
	src := `package input
func rec(xs []int) int { if len(xs)==0 {return 0}; return xs[0]+rec(xs[1:]) }
func g(xs []int) int { return rec(xs) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "g")
	sp, _ := NewSpace(nil).SpaceOf(fn, New(nil))
	if sp.Heap.String() != "O(len(xs))" || sp.Stack.String() != "O(1)" {
		t.Errorf("space = {heap %q, stack %q}, want {O(len(xs)), O(1)}", sp.Heap.String(), sp.Stack.String())
	}
}
