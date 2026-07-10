package loopnest

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func TestSingleLoop(t *testing.T) {
	const src = `package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ { s += xs[i] }
	return s
}`
	forest := buildForest(t, src, "f")
	if len(forest.Roots) != 1 {
		t.Fatalf("want 1 root loop, got %d", len(forest.Roots))
	}
	if forest.Roots[0].Depth != 0 {
		t.Errorf("root depth = %d, want 0", forest.Roots[0].Depth)
	}
	if len(forest.Roots[0].Children) != 0 {
		t.Errorf("root should have no children")
	}
}

func TestNestedLoops(t *testing.T) {
	const src = `package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs); j++ { s += xs[i] * xs[j] }
	}
	return s
}`
	forest := buildForest(t, src, "f")
	if len(forest.Roots) != 1 {
		t.Fatalf("want 1 root loop, got %d", len(forest.Roots))
	}
	root := forest.Roots[0]
	if len(root.Children) != 1 {
		t.Fatalf("want 1 nested loop, got %d", len(root.Children))
	}
	if root.Children[0].Depth != 1 {
		t.Errorf("inner depth = %d, want 1", root.Children[0].Depth)
	}
}

func TestNoLoops(t *testing.T) {
	const src = `package input
func f(x int) int { return x + 1 }`
	if forest := buildForest(t, src, "f"); len(forest.Roots) != 0 {
		t.Errorf("want 0 loops, got %d", len(forest.Roots))
	}
}

func buildForest(t *testing.T, src, name string) *Forest {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, name)
	if fn == nil {
		t.Fatalf("function %q not found", name)
	}
	return Build(fn)
}

func TestUncoveredCycleIrreducible(t *testing.T) {
	// Two-entry cycle: neither a nor b dominates the other, so no natural
	// loop exists — UncoveredCycle must catch it (review finding B4).
	const src = `package input
func f(n int, c bool) int {
	i := 0
	if c {
		goto b
	}
a:
	i++
b:
	i++
	if i < n {
		goto a
	}
	return i
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	forest := Build(fn)
	if !forest.UncoveredCycle(fn) {
		t.Error("UncoveredCycle = false, want true for irreducible two-entry cycle")
	}
}

func TestUncoveredCycleFalseForNaturalLoops(t *testing.T) {
	const src = `package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs); j++ {
			s += xs[i] * xs[j]
		}
	}
	return s
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if Build(fn).UncoveredCycle(fn) {
		t.Error("UncoveredCycle = true, want false — every cycle here is a natural loop")
	}
}
