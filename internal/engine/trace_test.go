package engine

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// The guarantee the whole design exists to provide: a traced run and an
// untraced run share one code path, so they cannot disagree.
func TestTracedEqualsUntraced(t *testing.T) {
	for _, src := range []string{
		`package input
func f(xs []int) int { n := 0; for i := 0; i < len(xs); i++ { n += xs[i] }; return n }`,
		`package input
func f(xs []int) int { n := 0; for len(xs) > 0 { xs = xs[n:]; n++ }; return n }`,
		`package input
func f() int { return 1 }`,
	} {
		pkg, _, err := ssasupport.Build(src)
		if err != nil {
			t.Fatal(err)
		}
		fn := ssasupport.Func(pkg, "f")

		wantB, wantC := InferDetailed(fn, builtinModel{})
		var tr Trace
		gotB, gotC := InferTraced(fn, builtinModel{}, &tr)

		if wantB.String() != gotB.String() {
			t.Errorf("bound: untraced %s, traced %s", wantB.String(), gotB.String())
		}
		if len(wantC) != len(gotC) {
			t.Errorf("causes: untraced %d, traced %d", len(wantC), len(gotC))
		}
	}
}

func TestTraceRecordsTheLoopRule(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
func f(xs []int) int { n := 0; for i := 0; i < len(xs); i++ { n += xs[i] }; return n }`)
	if err != nil {
		t.Fatal(err)
	}
	var tr Trace
	InferTraced(ssasupport.Func(pkg, "f"), builtinModel{}, &tr)

	if len(tr.Loops) != 1 {
		t.Fatalf("Loops = %+v, want exactly one", tr.Loops)
	}
	if tr.Loops[0].Rule != "increasing (unit step)" {
		t.Errorf("Rule = %q", tr.Loops[0].Rule)
	}
	if !tr.Loops[0].Pos.IsValid() {
		t.Error("a loop step must name where the loop is")
	}
}

// A loop no rule bounded records an empty Rule; the renderer turns that into
// "no rule matched" and must never see a fabricated name (spec 2.3).
func TestTraceRecordsNoRuleForAnUnboundedLoop(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
func f(xs []int) int { n := 0; for len(xs) > 0 { xs = xs[n:]; n++ }; return n }`)
	if err != nil {
		t.Fatal(err)
	}
	var tr Trace
	InferTraced(ssasupport.Func(pkg, "f"), builtinModel{}, &tr)

	if len(tr.Loops) != 1 {
		t.Fatalf("Loops = %+v, want exactly one", tr.Loops)
	}
	if tr.Loops[0].Rule != "" {
		t.Errorf("Rule = %q, want empty", tr.Loops[0].Rule)
	}
}

// Each loop is recorded ONCE, however many blocks it encloses.
func TestTraceRecordsEachLoopOnce(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
func f(xs []int) int {
	n := 0
	for i := 0; i < len(xs); i++ {
		if xs[i] > 0 {
			n++
		} else {
			n--
		}
	}
	return n
}`)
	if err != nil {
		t.Fatal(err)
	}
	var tr Trace
	InferTraced(ssasupport.Func(pkg, "f"), builtinModel{}, &tr)
	if len(tr.Loops) != 1 {
		t.Errorf("Loops = %d, want 1 — a loop enclosing several blocks is still one loop", len(tr.Loops))
	}
}

// Every surviving term must be attributed. reduce keeps maximal monomials from
// the union of block contributions, so each surviving term appears verbatim in
// some contribution and attribution is a lookup, never a guess.
func TestTraceAttributesEverySurvivingTerm(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
func f(xs []int) int { n := 0; for i := 0; i < len(xs); i++ { n += xs[i] }; return n }`)
	if err != nil {
		t.Fatal(err)
	}
	var tr Trace
	b, _ := InferTraced(ssasupport.Func(pkg, "f"), builtinModel{}, &tr)

	if len(tr.Terms) != len(b.Terms()) {
		t.Fatalf("Terms = %d, bound has %d surviving terms", len(tr.Terms), len(b.Terms()))
	}
	if len(tr.Terms) > 0 && len(tr.Terms[0].Loops) != 1 {
		t.Errorf("term %q loops = %v, want the one enclosing loop", tr.Terms[0].Term, tr.Terms[0].Loops)
	}
}

// A nil trace must be the status quo and must not panic.
func TestNilTraceIsTheStatusQuo(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
func f(xs []int) int { n := 0; for i := 0; i < len(xs); i++ { n++ }; return n }`)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	a, _ := InferTraced(fn, builtinModel{}, nil)
	b, _ := InferDetailed(fn, builtinModel{})
	if a.String() != b.String() {
		t.Errorf("nil-trace bound %s != InferDetailed %s", a.String(), b.String())
	}
}
