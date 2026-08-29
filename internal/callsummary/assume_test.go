package callsummary

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/assume"
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/normalize"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// inferWith is inferF plus an assumption set on the resolver.
func inferWith(t *testing.T, src, assumptions string) (string, *Resolver) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if fn == nil {
		t.Fatal("f not found")
	}
	r := New(nil)
	if assumptions != "" {
		es, err := assume.ParseText(assumptions)
		if err != nil {
			t.Fatal(err)
		}
		r.UseAssumptions(assume.NewSet(es))
	}
	return engine.Infer(fn, r).String(), r
}

func mustBound(t *testing.T, expr string, fn *ssa.Function) bound.Bound {
	t.Helper()
	dir, err := annotation.Parse("//bigo:cost " + expr)
	if err != nil {
		t.Fatal(err)
	}
	b, err := normalize.Budget(dir, fn)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAssumptionBoundsUncuratedStdlib(t *testing.T) {
	src := `package input
import "os"
func f(k string) string { return os.Getenv(k) }`
	got, _ := inferWith(t, src, "") // control: unpriced stdlib is unverifiable
	if got != "unverifiable" {
		t.Fatalf("control = %q, want unverifiable", got)
	}
	got, _ = inferWith(t, src, "os.Getenv O(1)\n")
	if got != "O(1)" {
		t.Fatalf("assumed = %q, want O(1)", got)
	}
}

func TestAssumptionBeatsInference(t *testing.T) {
	src := `package input
import "os"
func blocked(xs []int) int { os.Getenv("x"); return len(xs) }
func f(xs []int) int { return blocked(xs) }`
	got, _ := inferWith(t, src, "input.blocked O(n) where n=len(xs)\n")
	if got != "O(len(xs))" {
		t.Fatalf("got %q, want the assumed parametric bound in caller vars", got)
	}
}

func TestDirectiveBeatsAssumptionAndWarns(t *testing.T) {
	src := `package input
func helper(ys []int) int { s := 0; for i := 0; i < len(ys); i++ { s += ys[i] }; return s }
func f(xs []int) int { return helper(xs) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	helper := ssasupport.Func(pkg, "helper")
	es, err := assume.ParseText("input.helper O(1)\n")
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a //bigo:cost directive: an override on helper.
	r := New(map[*ssa.Function]bound.Bound{helper: mustBound(t, "O(n) where n=len(ys)", helper)})
	r.UseAssumptions(assume.NewSet(es))
	if got := engine.Infer(fn, r).String(); got != "O(len(xs))" {
		t.Fatalf("got %q — the directive must win over the O(1) assumption", got)
	}
	warns := r.AssumeWarnings()
	if len(warns) != 1 || !strings.Contains(warns[0], "input.helper") || !strings.Contains(warns[0], "directive") {
		t.Fatalf("warnings = %v, want one directive-shadow warning naming input.helper", warns)
	}
}

func TestTaintDirectAndTransitive(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
import "os"
func leaf(k string) string { return os.Getenv(k) }
func mid(k string) string { return leaf(k) }
func f(k string) string { return mid(k) }
func clean(xs []int) int { return len(xs) }`)
	if err != nil {
		t.Fatal(err)
	}
	es, err := assume.ParseText("os.Getenv O(1)\n")
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	r.UseAssumptions(assume.NewSet(es))
	// Run every function as a top-level target: taint is per-function state.
	for _, name := range []string{"leaf", "mid", "f", "clean"} {
		fn := ssasupport.Func(pkg, name)
		if fn == nil {
			t.Fatalf("%s not found", name)
		}
		b, _ := r.InferTop(fn)
		if b.IsTop() {
			t.Fatalf("%s = top, want bounded", name)
		}
	}
	for name, want := range map[string]bool{"f": true, "mid": true, "leaf": true, "clean": false} {
		fn := ssasupport.Func(pkg, name)
		if got := r.Tainted(fn); got != want {
			t.Errorf("Tainted(%s) = %v, want %v", name, got, want)
		}
	}
}

func TestMemoHitPropagatesTaint(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
import "os"
func leaf(k string) string { return os.Getenv(k) }
func a(k string) string { return leaf(k) }
func b(k string) string { return leaf(k) }`)
	if err != nil {
		t.Fatal(err)
	}
	es, err := assume.ParseText("os.Getenv O(1)\n")
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	r.UseAssumptions(assume.NewSet(es))
	fa, fb := ssasupport.Func(pkg, "a"), ssasupport.Func(pkg, "b")
	r.InferTop(fa) // populates leaf's memo with taint
	r.InferTop(fb) // must inherit taint from the MEMOIZED leaf summary
	if !r.Tainted(fb) {
		t.Fatal("b not tainted — memo-hit path dropped the taint bit")
	}
}

func TestCuratedTableBeatsAssumptionAndWarns(t *testing.T) {
	src := `package input
import "sort"
func f(xs []int) { sort.Ints(xs) }`
	got, r := inferWith(t, src, "sort.Ints O(1)\n")
	if got != "O(len(xs) log(len(xs)))" {
		t.Fatalf("got %q — the curated entry must win", got)
	}
	warns := r.AssumeWarnings()
	if len(warns) != 1 || !strings.Contains(warns[0], "sort.Ints") {
		t.Fatalf("warnings = %v, want one curated-shadow warning", warns)
	}
}

// TestAssumptionShadowedByParametricEntryWarns pins the THIRD precedence
// holder. The plain table and //bigo: directives already warn; the parametric
// table returns first in CallCost's no-body branch and warned about nothing,
// so an assumption on sort.Slice contributed nothing and said nothing — a
// silent skip, which this package's doc forbids because it under-counts a
// what-if measurement into a fake NO-GO (2026-08-11 review F1).
func TestAssumptionShadowedByParametricEntryWarns(t *testing.T) {
	src := `package input
import "sort"
func f(xs []int) { sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] }) }`

	got, r := inferWith(t, src, "sort.Slice O(1)\n")

	// The curated parametric entry still WINS — precedence is unchanged, and
	// this half is the positive control: a fix that let the assumption through
	// would break the documented precedence order instead of adding a warning.
	if got != "O(len(xs) log(len(xs)))" {
		t.Errorf("bound = %q, want the parametric entry to still outrank the assumption", got)
	}

	warns := r.AssumeWarnings()
	want := "assumption for sort.Slice is shadowed by a parametric cost-table entry"
	found := false
	for _, w := range warns {
		if w == want {
			found = true
		}
	}
	if !found {
		t.Errorf("warnings = %q, want one naming the shadowed key: %q", warns, want)
	}
}

// taintOfPair infers two top-level functions on ONE resolver, in the given
// order, and reports whether each ended up tainted.
func taintOfPair(t *testing.T, src, assumptions, first, second string) (bool, bool) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	r := New(nil)
	es, err := assume.ParseText(assumptions)
	if err != nil {
		t.Fatal(err)
	}
	r.UseAssumptions(assume.NewSet(es))

	fa, fb := ssasupport.Func(pkg, first), ssasupport.Func(pkg, second)
	if fa == nil || fb == nil {
		t.Fatalf("fixture functions %q/%q not found", first, second)
	}
	r.InferTop(fa)
	r.InferTop(fb)
	return r.Tainted(fa), r.Tainted(fb)
}

// TestParamSummaryTaintIsOrderIndependent is a DIFFERENTIAL PAIR and must stay
// one: the defect it pins (2026-08-11 review F2) tainted only whichever
// consumer ran first, so pinning a single order passes throughout the defect's
// life.
//
// Run has a func-typed parameter, so its cost goes through paramSummaryOf,
// whose memo cached the VALUE without the taint that produced it.
func TestParamSummaryTaintIsOrderIndependent(t *testing.T) {
	src := `package input
func slow(m map[int]int, k int) int {
	for k != 0 {
		k = m[k]
	}
	return k
}
func Run(f func(), m map[int]int) {
	slow(m, 1)
	f()
}
func A(m map[int]int) { Run(func() {}, m) }
func B(m map[int]int) { Run(func() {}, m) }`

	const assumptions = "input.slow O(1)\n"

	ta, tb := taintOfPair(t, src, assumptions, "A", "B")
	if !ta || !tb {
		t.Errorf("A-then-B: tainted A=%v B=%v, want both true", ta, tb)
	}
	tb2, ta2 := taintOfPair(t, src, assumptions, "B", "A")
	if !tb2 || !ta2 {
		t.Errorf("B-then-A: tainted B=%v A=%v, want both true", tb2, ta2)
	}
}

// inferWithOverlay is inferWith for a cost-model overlay instead of an
// assumption set: same shape, opposite precedence.
func inferWithOverlay(t *testing.T, src, overlay string) (string, *Resolver) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if fn == nil {
		t.Fatal("f not found")
	}
	r := New(nil)
	es, err := assume.ParseText(overlay)
	if err != nil {
		t.Fatal(err)
	}
	r.UseOverlay(assume.NewSet(es))
	return engine.Infer(fn, r).String(), r
}

// An overlay entry must beat a curated entry — that is its entire purpose —
// while a plain assumption must still lose to one.
func TestOverlayOutranksCuratedTable(t *testing.T) {
	// strings.Compare is curated linear in its first operand.
	const src = `package input
import "strings"
func f(a, b string) int { return strings.Compare(a, b) }`
	const entry = "strings.Compare O(1)\n"

	t.Run("no overlay: curated bound stands", func(t *testing.T) {
		got, _ := inferWith(t, src, "")
		if got == "O(1)" {
			t.Fatalf("default precedence moved: got %q, want the curated bound", got)
		}
	})

	t.Run("assumption loses to curated", func(t *testing.T) {
		got, _ := inferWith(t, src, entry)
		if got == "O(1)" {
			t.Fatalf("assumption beat the curated entry: got %q", got)
		}
	})

	t.Run("overlay beats curated", func(t *testing.T) {
		got, _ := inferWithOverlay(t, src, entry)
		if got != "O(1)" {
			t.Fatalf("overlay did not beat the curated entry: got %q, want O(1)", got)
		}
	})

	t.Run("overlay does not warn about shadowing", func(t *testing.T) {
		_, r := inferWithOverlay(t, src, entry)
		if w := r.AssumeWarnings(); len(w) != 0 {
			t.Fatalf("overlay keys must not warn: shadowing is the point, got %v", w)
		}
	})
}

// An overlay must reach INVOKE-MODE call sites — interface dispatch — because a
// cost model that means to neutralize a method must not silently miss the half
// of its call sites that go through an interface. overlayCost documents exactly
// this, and its `case c.Method != nil` signature branch was unreachable until
// CallKey existed: it keyed through CalleeKey, which returns ok=false whenever
// StaticCallee() is nil, so the branch below it could never run.
//
// Measured before the fix (bigo v1.54.0, kata profile + the hashtable kata):
// `(hash.Hash32).Sum32 O(1)` in the profile changed nothing — the explain line
// still read `call at :48 → (hash.Hash32).Sum32 → unresolved`.
func TestOverlayReachesInterfaceDispatch(t *testing.T) {
	const src = `package input
type Hasher interface{ Sum32() uint32 }
func f(h Hasher) uint32 { return h.Sum32() }`

	t.Run("no overlay: an unannotated interface method is Top", func(t *testing.T) {
		got, _ := inferWith(t, src, "")
		if got != "unverifiable" {
			t.Fatalf("got %q, want unverifiable", got)
		}
	})

	t.Run("overlay entry prices the dispatch", func(t *testing.T) {
		got, _ := inferWithOverlay(t, src, "(input.Hasher).Sum32 O(1)\n")
		if got != "O(1)" {
			t.Fatalf("got %q, want O(1) — the overlay did not reach the invoke-mode call", got)
		}
	})
}
