package sizefacts

import (
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// innerBoundOperand builds src, finds the INNER-most loop (max depth), and
// returns its condition's bound-side operand plus a Facts instance — the
// exact value UpperExtent sees in real use.
func innerBoundOperand(t *testing.T, src string) (ssa.Value, *Facts) {
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
	return cmp.Y, &Facts{Stab: fieldpath.Analyze(fn)}
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
			got, ok := fx.UpperExtent(v, 0)
			if !ok || string(got) != tt.want {
				t.Errorf("UpperExtent = (%q, %v), want (%q, true)", got, ok, tt.want)
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
			if got, ok := fx.UpperExtent(v, 0); ok {
				t.Errorf("UpperExtent = (%q, true), want rejection — accepting is a wrong-bound bug", got)
			}
		})
	}
}

// headerPhi returns the first *ssa.Phi in the outermost loop's header.
func headerPhi(t *testing.T, src string) *ssa.Phi {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	forest := loopnest.Build(fn)
	if len(forest.Roots) == 0 {
		t.Fatal("no loop")
	}
	for _, in := range forest.Roots[0].Header.Instrs {
		if phi, ok := in.(*ssa.Phi); ok {
			return phi
		}
	}
	t.Fatal("no header phi")
	return nil
}

func TestGuardBoundRejectsInvertedExit(t *testing.T) {
	// The header test is `t = i < n; if t goto EXIT else goto BODY` — phi on the
	// low side of `<`, but the TRUE branch leaves the loop, so the loop CONTINUES
	// while i >= n (infinite for n <= 0) and i is NOT bounded above by n.
	// go/ssa emits exactly this for the negated-guard goto form below.
	phi := headerPhi(t, `package input
func f(n int) {
	i := 0
loop:
	if i < n {
		goto done
	}
	_ = i
	i++
	goto loop
done:
}`)
	if _, ok := guardBound(phi); ok {
		t.Errorf("guardBound accepted an inverted-exit loop: unsound upper bound on an unbounded induction")
	}
}

func TestGuardBoundAcceptsStandardLoop(t *testing.T) {
	// for i := 0; i < n; i++ : true branch stays in the loop; i <= n holds.
	phi := headerPhi(t, `package input
func f(n int) { for i := 0; i < n; i++ { _ = i } }`)
	if _, ok := guardBound(phi); !ok {
		t.Errorf("guardBound rejected a standard counted loop (precision regression)")
	}
}
