package recurrence

import (
	"strings"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/size"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// genDivideConquer emits source for T(n) = a·T(n/2) + n^d over a slice: a
// sequential self-calls on xs[:len/2] (so the branching factor is a), with an
// O(len(xs)) scan of the parameter as the per-level work when d == 1.
func genDivideConquer(a, d int) string {
	var b strings.Builder
	b.WriteString("package input\nfunc f(xs []int) int {\n")
	if d == 1 {
		b.WriteString("\ts := 0\n\tfor _, v := range xs {\n\t\ts += v\n\t}\n")
	}
	b.WriteString("\tif len(xs) < 2 {\n\t\treturn 0\n\t}\n")
	b.WriteString("\tm := len(xs) / 2\n\treturn ")
	if d == 1 {
		b.WriteString("s")
	} else {
		b.WriteString("0")
	}
	for range a {
		b.WriteString(" + f(xs[:m])")
	}
	b.WriteString("\n}\n")
	return b.String()
}

// workBound builds the per-level work O(1) (d==0) or O(len(xs)) (d==1).
func workBound(d int) bound.Bound {
	if d == 1 {
		return bound.Of(bound.Term(size.Len("xs")))
	}
	return bound.Constant()
}

// TestMasterProperty is a differential check: for T(n)=a·T(n/2)+n^d over the
// small grid a ∈ {1,2,4}, d ∈ {0,1}, the SSA path (Solve) must agree with the
// closed form (solveMaster) computed directly.
func TestMasterProperty(t *testing.T) {
	for _, a := range []int{1, 2, 4} {
		for _, d := range []int{0, 1} {
			src := genDivideConquer(a, d)
			pkg, _, err := ssasupport.Build(src)
			if err != nil {
				t.Fatalf("a=%d d=%d: build: %v\n%s", a, d, err, src)
			}
			fn := ssasupport.Func(pkg, "f")
			got, ok := Solve(fn, stubModel{})
			want, wok := solveMaster(a, 2, workBound(d), size.Len("xs"))
			if !ok || !wok || !got.Equal(want) {
				t.Errorf("a=%d d=%d: Solve=(%q,%v), solveMaster=(%q,%v)",
					a, d, got.String(), ok, want.String(), wok)
			}
		}
	}
}
