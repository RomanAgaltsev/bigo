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

func TestCallCostParametricStaticArg(t *testing.T) {
	src := `package input
func Map(xs []int, f func(int) int) []int {
	out := make([]int, 0, len(xs))
	for _, v := range xs {
		out = append(out, f(v))
	}
	return out
}
func double(x int) int { return x * 2 }
func UseConst(zs []int) []int { return Map(zs, double) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	got := r.summary(ssasupport.Func(pkg, "UseConst"))
	if got.String() != "O(len(zs))" {
		t.Errorf("Map with O(1) arg = %q, want O(len(zs))", got.String())
	}
}

func TestCallCostParametricSizedArgRefused(t *testing.T) {
	// scanAll's cost is O(len(ys)) in ITS OWN param — unresolvable at Map's site.
	src := `package input
var global []int
func scanAll(x int, ys []int) bool {
	for _, y := range ys {
		if y == x { return true }
	}
	return false
}
func wrapper(x int) int { if scanAll(x, global) { return 1 }; return 0 }
func Map(xs []int, f func(int) int) []int {
	out := make([]int, 0, len(xs))
	for _, v := range xs {
		out = append(out, f(v))
	}
	return out
}
func UseSized(zs []int) []int { return Map(zs, wrapper) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	if got := r.summary(ssasupport.Func(pkg, "UseSized")); !got.IsTop() {
		t.Errorf("size-dependent func arg must refuse to price: got %q", got.String())
	}
}

func TestClosureCostO1Comparator(t *testing.T) {
	// The sort.Slice shape: captures xs, but the BODY bound is O(1) — no free
	// var appears in the bound, so nothing needs renaming.
	src := `package input
func each(xs []int, f func(int)) {
	for _, v := range xs { f(v) }
}
func Use(xs []int) {
	each(xs, func(i int) { _ = xs[0] + i })
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	if got := r.summary(ssasupport.Func(pkg, "Use")); got.String() != "O(len(xs))" {
		t.Errorf("O(1)-closure each = %q, want O(len(xs))", got.String())
	}
}

func TestClosureCostCaptureSizedRefused(t *testing.T) {
	// Sound-subset outcome (see closure.go): the closure body loops over the
	// captured slice, so its plain summary is ⊤ (free-var sizes are not
	// canonical roots), and closureCost refuses. Product-bound pricing of
	// capture-sized closures is deferred.
	src := `package input
func each(ys []int, f func(int)) {
	for _, v := range ys { f(v) }
}
func Use(xs, ys []int) {
	each(ys, func(int) {
		s := 0
		for _, v := range xs { s += v }
		_ = s
	})
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	if got := r.summary(ssasupport.Func(pkg, "Use")); !got.IsTop() {
		t.Errorf("capture-sized closure must refuse (deferred): got %q", got.String())
	}
}

func TestClosureCostMutatedCaptureRefused(t *testing.T) {
	// Pin 3: the captured slice is reassigned between MakeClosure and the call.
	src := `package input
func each(ys []int, f func(int)) {
	for _, v := range ys { f(v) }
}
func Use(xs, ys []int) {
	f := func(int) {
		s := 0
		for _, v := range xs { s += v }
		_ = s
	}
	xs = append(xs, 1) // capture no longer entry-stable at the consuming call
	each(ys, f)
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	if got := r.summary(ssasupport.Func(pkg, "Use")); !got.IsTop() {
		t.Errorf("mutated capture must refuse: got %q", got.String())
	}
}

func TestSortSliceParametric(t *testing.T) {
	src := `package input
import "sort"
func Sort(xs []int) {
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	got := r.summary(ssasupport.Func(pkg, "Sort")).String()
	if got != "O(len(xs) log(len(xs)))" {
		t.Errorf("sort.Slice with O(1) comparator = %q, want O(len(xs) log(len(xs)))", got)
	}
}

func TestSortSliceUnresolvedCallbackTop(t *testing.T) {
	// Pin 8: the comparator is a func value from a struct field -> ⊤.
	src := `package input
import "sort"
var held struct{ less func(i, j int) bool }
func Sort(xs []int) {
	sort.Slice(xs, held.less)
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	if got := r.summary(ssasupport.Func(pkg, "Sort")); !got.IsTop() {
		t.Errorf("sort.Slice with unresolved comparator must be ⊤: got %q", got.String())
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
