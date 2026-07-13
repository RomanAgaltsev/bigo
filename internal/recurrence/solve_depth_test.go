package recurrence

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func TestSolveDepth(t *testing.T) {
	cases := []struct{ name, src, wantDepth string }{
		{"subtractive depth O(n)", `package input
func f(xs []int) int { if len(xs)==0 {return 0}; return xs[0]+f(xs[1:]) }`, "O(len(xs))"},
		{"divisive depth O(log n)", `package input
func f(xs []int, t int) int { if len(xs)==0 {return -1}; m:=len(xs)/2; if xs[m]<t {return f(xs[m+1:],t)}; return f(xs[:m],t) }`, "O(log(len(xs)))"},
	}
	for _, c := range cases {
		pkg, _, err := ssasupport.Build(c.src)
		if err != nil {
			t.Fatal(err)
		}
		_, depth, ok := Solve(ssasupport.Func(pkg, "f"), stubModel{})
		if !ok || depth.String() != c.wantDepth {
			t.Errorf("%s: depth = (%q,%v), want %q", c.name, depth.String(), ok, c.wantDepth)
		}
	}
}
