package sizefacts

import (
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// argOf returns the first argument of the first call to callee inside f,
// plus a Facts for f — the shape ArgSize sees from the cost table.
func argOf(t *testing.T, src, callee string) (ssa.Value, *Facts) {
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
			if sc := c.Call.StaticCallee(); sc != nil && sc.Name() == callee {
				return c.Call.Args[0], &Facts{Stab: fieldpath.Analyze(fn)}
			}
		}
	}
	t.Fatalf("no call to %s in f", callee)
	return nil, nil
}

func TestArgSize(t *testing.T) {
	cases := []struct {
		name, src, want string
		ok              bool
	}{
		{"slice param", `package input
func sink(xs []int) {}
func f(s []int) { sink(s) }`, "len(s)", true},
		{"append copy idiom", `package input
func sink(xs []int) {}
func f(s []int) { local := append([]int(nil), s...); sink(local) }`, "len(s)", true},
		{"make with param length", `package input
func sink(xs []int) {}
func f(n int) { sink(make([]int, n)) }`, "n", true},
		{"slice expression high bound", `package input
func sink(xs []int) {}
func f(s []int, i, j int) { sink(s[i:j]) }`, "j", true},
		{"string slice", `package input
func sink(x string) {}
func f(s string) { sink(s[1:]) }`, "len(s)", true},
		{"integer derived from len", `package input
func sink(n int) {}
func f(s []int) { mid := len(s) / 2; sink(mid) }`, "len(s)", true},
		{"call result fails", `package input
func sink(xs []int) {}
func g() []int { return nil }
func f() { sink(g()) }`, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, f := argOf(t, c.src, "sink")
			got, ok := f.ArgSize(v)
			if ok != c.ok || string(got) != c.want {
				t.Errorf("ArgSize = (%q, %v), want (%q, %v)", got, ok, c.want, c.ok)
			}
		})
	}
}
