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
		total := 1
		for range depth {
			total *= 4
		}
		for mask := 0; mask < total; mask++ {
			kinds := make([]int, depth)
			m := mask
			for d := range kinds {
				kinds[d] = m % 4
				m /= 4
			}
			t.Run(fmt.Sprintf("depth%d_mask%d", depth, mask), func(t *testing.T) {
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
// kinds: 0 = counted over len(xs), 1 = counted over n,
//
//	2 = countdown from n, 3 = doubling up to n (contributes log n).
func buildNest(kinds []int) (string, bound.Bound) {
	var b strings.Builder
	b.WriteString("package input\n\nfunc f(xs []int, n int) int {\n\ts := 0\n")
	want := bound.Constant()
	for d, kind := range kinds {
		switch kind {
		case 0:
			fmt.Fprintf(&b, "\tfor i%d := 0; i%d < len(xs); i%d++ {\n", d, d, d)
			want = want.Mul(bound.Of(bound.Term("len(xs)")))
		case 1:
			fmt.Fprintf(&b, "\tfor i%d := 0; i%d < n; i%d++ {\n", d, d, d)
			want = want.Mul(bound.Of(bound.Term("n")))
		case 2:
			fmt.Fprintf(&b, "\tfor i%d := n; i%d > 0; i%d-- {\n", d, d, d)
			want = want.Mul(bound.Of(bound.Term("n")))
		case 3:
			fmt.Fprintf(&b, "\tfor i%d := 1; i%d < n; i%d *= 2 {\n", d, d, d)
			want = want.Mul(bound.Of(bound.Mono("n", 0, 1)))
		}
	}
	b.WriteString("\ts++\n")
	for range kinds {
		b.WriteString("\t}\n")
	}
	b.WriteString("\treturn s\n}\n")
	return b.String(), want
}
