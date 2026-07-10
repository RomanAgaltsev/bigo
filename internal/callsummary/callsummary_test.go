package callsummary

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
	"golang.org/x/tools/go/ssa"
)

func inferF(t *testing.T, src string) string {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if fn == nil {
		t.Fatal("f not found")
	}
	return engine.Infer(fn, New(nil)).String()
}

func TestInterprocedural(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"call resolves to callee bound",
			`package input
func helper(ys []int) int { s := 0; for i := 0; i < len(ys); i++ { s += ys[i] }; return s }
func f(xs []int) int { return helper(xs) }`,
			"O(len(xs))",
		},
		{
			"linear caller over linear callee is quadratic",
			`package input
func helper(ys []int) int { s := 0; for i := 0; i < len(ys); i++ { s += ys[i] }; return s }
func f(xs []int) int { s := 0; for i := 0; i < len(xs); i++ { s += helper(xs) }; return s }`,
			"O(len(xs)^2)",
		},
		{
			"stdlib sort resolves",
			`package input
import "sort"
func f(xs []int) { sort.Ints(xs) }`,
			"O(len(xs) log(len(xs)))",
		},
		{
			"recursion is unverifiable",
			`package input
func f(xs []int) int { if len(xs) == 0 { return 0 }; return f(xs[1:]) }`,
			"unverifiable",
		},
		{
			"interface call is unverifiable",
			`package input
type D interface{ Do(int) int }
func f(xs []int, d D) int { s := 0; for i := 0; i < len(xs); i++ { s += d.Do(xs[i]) }; return s }`,
			"unverifiable",
		},
		{
			"external call not in the table is unverifiable",
			`package input
import "os"
func f(k string) string { return os.Getenv(k) }`,
			"unverifiable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inferF(t, tt.src); got != tt.want {
				t.Errorf("Infer = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCapBoundedCalleeSubstitutesCap(t *testing.T) {
	// cap(ys) in the callee must become cap(xs) — NOT len(xs); a length is
	// not an upper bound on a capacity (review finding B3).
	const src = `package input
func fill(ys []int) int { s := 0; for i := 0; i < cap(ys); i++ { s++ }; return s }
func f(xs []int) int { return fill(xs) }`
	if got, want := inferF(t, src), "O(cap(xs))"; got != want {
		t.Errorf("Infer = %q, want %q", got, want)
	}
}

func TestCapBoundedCalleeWithNonParamArgIsTop(t *testing.T) {
	// The argument is a local slice, so its cap is not expressible in the
	// caller's size vars; the summary depends on cap -> unverifiable.
	const src = `package input
func fill(ys []int) int { s := 0; for i := 0; i < cap(ys); i++ { s++ }; return s }
func mk() []int
func f() int { return fill(mk()) }`
	if got := inferF(t, src); got != "unverifiable" {
		t.Errorf("Infer = %q, want unverifiable", got)
	}
}

func TestOverrideResolvesBodylessCallee(t *testing.T) {
	const src = `package input
func opaque(x int) int
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ { s += opaque(xs[i]) }
	return s
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	opaque := ssasupport.Func(pkg, "opaque")
	f := ssasupport.Func(pkg, "f")
	// //bigo:cost O(1) on opaque, expressed as an override.
	r := New(map[*ssa.Function]bound.Bound{opaque: bound.Constant()})
	if got, want := engine.Infer(f, r).String(), "O(len(xs))"; got != want {
		t.Errorf("Infer = %q, want %q", got, want)
	}
}

func TestOverrideBeatsBodyAnalysis(t *testing.T) {
	// A trusted (//bigo:ignore) quadratic helper must count as O(1).
	const src = `package input
func heavy(ys []int) int {
	s := 0
	for i := 0; i < len(ys); i++ {
		for j := 0; j < len(ys); j++ { s++ }
	}
	return s
}
func f(xs []int) int { return heavy(xs) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	heavy := ssasupport.Func(pkg, "heavy")
	f := ssasupport.Func(pkg, "f")
	r := New(map[*ssa.Function]bound.Bound{heavy: bound.Constant()})
	if got, want := engine.Infer(f, r).String(), "O(1)"; got != want {
		t.Errorf("Infer = %q, want %q", got, want)
	}
}

func TestOverrideSubstitutesParams(t *testing.T) {
	// cost O(k) on opaque(k int): calling opaque(n) must yield O(n).
	const src = `package input
func opaque(k int) int
func f(n int) int { return opaque(n) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	opaque := ssasupport.Func(pkg, "opaque")
	f := ssasupport.Func(pkg, "f")
	r := New(map[*ssa.Function]bound.Bound{opaque: bound.Of(bound.Term("k"))})
	if got, want := engine.Infer(f, r).String(), "O(n)"; got != want {
		t.Errorf("Infer = %q, want %q", got, want)
	}
}
