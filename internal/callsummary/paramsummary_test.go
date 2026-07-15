package callsummary

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func paramSummary(t *testing.T, src, name string) (ParamSummary, bool) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	return r.paramSummaryOf(ssasupport.Func(pkg, name))
}

func TestParamSummaryMap(t *testing.T) {
	ps, ok := paramSummary(t, `package input
func Map(xs []int, f func(int) int) []int {
	out := make([]int, 0, len(xs))
	for _, v := range xs {
		out = append(out, f(v))
	}
	return out
}`, "Map")
	if !ok {
		t.Fatal("Map must have a param summary")
	}
	if got := ps.PerParam[1].String(); got != "O(len(xs))" {
		t.Errorf("PerParam[f] = %q, want O(len(xs))", got)
	}
	if ps.Base.IsTop() {
		t.Error("Base must be finite (O(len(xs)) loop of O(1) work)")
	}
}

func TestParamSummaryCallOutsideLoop(t *testing.T) {
	ps, ok := paramSummary(t, `package input
func Once(f func()) { f() }`, "Once")
	if !ok || ps.PerParam[0].String() != "O(1)" {
		t.Fatalf("single unlooped invocation must count O(1); got %+v, %v", ps, ok)
	}
}

func TestParamSummaryEscapeStored(t *testing.T) {
	// Pin 6 (unit level): stored func param -> count ⊤.
	ps, ok := paramSummary(t, `package input
var sink func()
func Store(f func()) { sink = f; f() }`, "Store")
	if !ok || !ps.PerParam[0].IsTop() {
		t.Fatalf("stored param must poison its count to ⊤; got %+v, %v", ps, ok)
	}
}

func TestParamSummaryEscapeUnknownCallee(t *testing.T) {
	// Pin 7 (unit level): pass-through to a non-parametric unknown -> ⊤ count.
	ps, ok := paramSummary(t, `package input
func Opaque(f func())
func Fwd(f func()) { Opaque(f) }`, "Fwd")
	if !ok || !ps.PerParam[0].IsTop() {
		t.Fatalf("pass-through to bodyless callee must poison; got %+v, %v", ps, ok)
	}
}

func TestParamSummaryComposition(t *testing.T) {
	// g forwards f to Map: g's count for f = Map's count, in g's vocabulary.
	ps, ok := paramSummary(t, `package input
func Map(xs []int, f func(int) int) []int {
	out := make([]int, 0, len(xs))
	for _, v := range xs {
		out = append(out, f(v))
	}
	return out
}
func Twice(ys []int, g func(int) int) []int {
	Map(ys, g)
	return Map(ys, g)
}`, "Twice")
	if !ok || ps.PerParam[1].String() != "O(len(ys))" {
		t.Fatalf("composed count must be O(len(ys)); got %+v, %v", ps, ok)
	}
}

func TestParamSummaryUnboundedLoopPoisons(t *testing.T) {
	// f invoked under a loop with an unrecognized trip count -> ⊤ count.
	ps, ok := paramSummary(t, `package input
func Spin(f func(), ch chan bool) {
	for <-ch {
		f()
	}
}`, "Spin")
	if !ok || !ps.PerParam[0].IsTop() {
		t.Fatalf("unbounded invocation loop must poison; got %+v, %v", ps, ok)
	}
}
