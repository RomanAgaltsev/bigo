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
