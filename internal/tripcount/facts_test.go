package tripcount

import (
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// innerBoundOperand builds src, finds the INNER-most loop (max depth), and
// returns its condition's bound-side operand plus a facts instance — the
// exact value upperExtent sees in real use.
func innerBoundOperand(t *testing.T, src string) (ssa.Value, *facts) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	forest := loopnest.Build(fn)
	var deepest *loopnest.Loop
	var walk func(l *loopnest.Loop)
	walk = func(l *loopnest.Loop) {
		if deepest == nil || l.Depth > deepest.Depth {
			deepest = l
		}
		for _, c := range l.Children {
			walk(c)
		}
	}
	for _, r := range forest.Roots {
		walk(r)
	}
	if deepest == nil {
		t.Fatal("no loop")
	}
	ifi := deepest.Header.Instrs[len(deepest.Header.Instrs)-1].(*ssa.If)
	cmp := ifi.Cond.(*ssa.BinOp)
	return cmp.Y, &facts{stab: fieldpath.Analyze(fn)}
}

func TestUpperExtent(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"len minus one minus induction (bubble bound)",
			`package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs)-1-i; j++ { s++ }
	}
	return s
}`,
			"len(xs)",
		},
		{
			"guard-bounded outer induction (triangular bound)",
			`package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < i; j++ { s++ }
	}
	return s
}`,
			"len(xs)",
		},
		{
			"halved length",
			`package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs)/2; i++ { s++ }
	return s
}`,
			"len(xs)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, fx := innerBoundOperand(t, tt.src)
			got, ok := fx.upperExtent(v, 0)
			if !ok || string(got) != tt.want {
				t.Errorf("upperExtent = (%q, %v), want (%q, true)", got, ok, tt.want)
			}
		})
	}
}

func TestUpperExtentRejects(t *testing.T) {
	tests := []struct{ name, src string }{
		{
			"len minus a PARAMETER is not dominated by len (param may be negative)",
			`package input
func f(xs []int, k int) int {
	s := 0
	for i := 0; i < len(xs)-k; i++ { s++ }
	return s
}`,
		},
		{
			"outer induction with a PARAMETER start has no guard-bound",
			`package input
func f(xs []int, m int) int {
	s := 0
	for i := m; i < len(xs); i++ {
		for j := 0; j < i; j++ { s++ }
	}
	return s
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, fx := innerBoundOperand(t, tt.src)
			if got, ok := fx.upperExtent(v, 0); ok {
				t.Errorf("upperExtent = (%q, true), want rejection — accepting is a wrong-bound bug", got)
			}
		})
	}
}
