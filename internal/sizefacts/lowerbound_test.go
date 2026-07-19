package sizefacts

import (
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// appendDstOf returns the destination operand of the first builtin append in f.
func appendDstOf(t *testing.T, src string) ssa.Value {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			c, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			if bi, ok := c.Call.Value.(*ssa.Builtin); ok && bi.Name() == "append" && len(c.Call.Args) == 2 {
				return c.Call.Args[0]
			}
		}
	}
	t.Fatal("no two-operand append in f")
	return nil
}

func TestZeroLen(t *testing.T) {
	cases := []struct {
		name, src string
		want      bool
	}{
		{"nil constant", `package input
func f(s []int) []int { return append([]int(nil), s...) }`, true},
		{"make zero", `package input
func f(s []int) []int { return append(make([]int, 0, len(s)), s...) }`, true},
		{"make nonzero", `package input
func f(s []int) []int { return append(make([]int, 1), s...) }`, false},
		{"parameter dst", `package input
func f(d, s []int) []int { return append(d, s...) }`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ZeroLen(appendDstOf(t, c.src)); got != c.want {
				t.Errorf("ZeroLen = %v, want %v", got, c.want)
			}
		})
	}
}

// tailInitOf returns the non-step init edge of the SECOND sequential loop's
// header phi in f, plus a Facts — the two-phi-cycle value B2 must prove >= 0.
func tailInitOf(t *testing.T, src string) (ssa.Value, *Facts) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	// The tail loop's header phi has one edge that is another *ssa.Phi (the
	// merge loop's carried value) — that edge is the init under test.
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok {
				continue
			}
			for _, e := range phi.Edges {
				if _, isPhi := e.(*ssa.Phi); isPhi {
					return e, &Facts{Stab: fieldpath.Analyze(fn)}
				}
			}
		}
	}
	t.Fatal("no phi-valued init edge found")
	return nil, nil
}

func TestLowerBoundConstTwoPhiCycleFallback(t *testing.T) {
	// The CountInversions tail-loop shape: i's value crosses the &&-lowered
	// merge loop's two-phi cycle. Strict resolution fails (the cycle defeats
	// the phi+c match and the depth cap); the coinductive fallback proves >= 0.
	src := `package input
func f(a, b []int) int {
	i, j, t := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			i++
		} else {
			j++
		}
		t++
	}
	for i < len(a) {
		i++
		t++
	}
	return t
}`
	v, f := tailInitOf(t, src)
	lo, ok := f.LowerBoundConst(v, 0)
	if !ok || lo != 0 {
		t.Errorf("LowerBoundConst = (%d, %v), want (0, true) via fallback", lo, ok)
	}
}

func TestLowerBoundConstStrictFirst(t *testing.T) {
	// Exact constants must survive: the geometric floor checks need init >= 1
	// EXACTLY (i *= 2 from 0 never moves). A preempting fallback returns 0 and
	// broke TestNestProperty in the probe.
	src := `package input
func sink(n int) {}
func f() { one := 1; sink(one + 0) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	f := &Facts{Stab: fieldpath.Analyze(fn)}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			c, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			if sc := c.Call.StaticCallee(); sc != nil && sc.Name() == "sink" {
				lo, ok := f.LowerBoundConst(c.Call.Args[0], 0)
				if !ok || lo != 1 {
					t.Errorf("LowerBoundConst(const-derived 1) = (%d, %v), want (1, true) via strict path", lo, ok)
				}
				return
			}
		}
	}
	t.Fatal("no sink call")
}

func TestLowerBoundConstNegativeStaysRejected(t *testing.T) {
	// A network entered by a negative constant proves nothing: the strict
	// min must survive, never a fabricated 0 from the fallback.
	src := `package input
func f(a []int, c bool) int {
	i := -1
	if c {
		i = 0
	}
	t := 0
	for i < len(a) {
		i++
		t++
	}
	return t
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	f := &Facts{Stab: fieldpath.Analyze(fn)}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok {
				continue
			}
			for _, e := range phi.Edges {
				if _, isPhi := e.(*ssa.Phi); isPhi {
					lo, ok := f.LowerBoundConst(e, 0)
					if !ok || lo != -1 {
						t.Errorf("LowerBoundConst = (%d, %v), want (-1, true) — strict min, not a fabricated 0", lo, ok)
					}
					return
				}
			}
		}
	}
	t.Fatal("no phi-valued init edge found")
}
