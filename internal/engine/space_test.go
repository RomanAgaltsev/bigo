package engine

import (
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// nilSpace resolves every call as O(1) space (no interproc in this unit test).
type nilSpace struct{}

func (nilSpace) CallSpace(*ssa.CallCommon) bound.Bound { return bound.Constant() }

func heapOf(t *testing.T, src string) string {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	sp, _ := InferSpace(ssasupport.Func(pkg, "f"), nilSpace{})
	return sp.Heap.String()
}

func TestInferSpaceHeap(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"make n", `package input
func f(n int) []int { return make([]int, n) }`, "O(n)"},
		{"constant", `package input
func f() int { x := 0; return x }`, "O(1)"},
		{"append in loop", `package input
func f(xs []int) []int { out := []int{}; for i := 0; i < len(xs); i++ { out = append(out, i) }; return out }`, "O(len(xs))"},
		{"unknown make len", `package input
func f(g func() int) []int { return make([]int, g()) }`, "unverifiable"},
	}
	for _, c := range cases {
		if got := heapOf(t, c.src); got != c.want {
			t.Errorf("%s: heap = %q, want %q", c.name, got, c.want)
		}
	}
}
