package engine

import (
	"fmt"
	"strings"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// TestNestProperty synthesizes loop nests with bound-by-construction and
// asserts the engine infers exactly the constructed product.
// Kinds: false = `i < len(xs)`, true = `i < n`.
func TestNestProperty(t *testing.T) {
	for depth := 1; depth <= 3; depth++ {
		for mask := 0; mask < 1<<depth; mask++ {
			kinds := make([]bool, depth)
			for d := range kinds {
				kinds[d] = mask&(1<<d) != 0
			}
			t.Run(fmt.Sprintf("depth%d_mask%b", depth, mask), func(t *testing.T) {
				src, want := buildNest(kinds)
				pkg, _, err := ssasupport.Build(src)
				if err != nil {
					t.Fatalf("build: %v\n%s", err, src)
				}
				fn := ssasupport.Func(pkg, "f")
				got := Infer(fn, builtinModel{})
				if got.String() != want.String() {
					t.Errorf("Infer = %q, want %q\n%s", got.String(), want.String(), src)
				}
			})
		}
	}
}

// buildNest emits a function nesting one counted loop per kind and the exact
// bound that nest must have.
func buildNest(kinds []bool) (string, bound.Bound) {
	var b strings.Builder
	b.WriteString("package input\n\nfunc f(xs []int, n int) int {\n\ts := 0\n")
	want := bound.Constant()
	for d, overN := range kinds {
		cond := fmt.Sprintf("i%d < len(xs)", d)
		term := bound.Of(bound.Term("len(xs)"))
		if overN {
			cond = fmt.Sprintf("i%d < n", d)
			term = bound.Of(bound.Term("n"))
		}
		fmt.Fprintf(&b, "\tfor i%d := 0; %s; i%d++ {\n", d, cond, d)
		want = want.Mul(term)
	}
	b.WriteString("\ts++\n")
	for range kinds {
		b.WriteString("\t}\n")
	}
	b.WriteString("\treturn s\n}\n")
	return b.String(), want
}
