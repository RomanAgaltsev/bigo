package engine

import (
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// loopCauses returns just the CauseLoop entries.
func loopCauses(cs []Cause) []Cause {
	out := make([]Cause, 0, len(cs))
	for _, c := range cs {
		if c.Kind == CauseLoop {
			out = append(out, c)
		}
	}
	return out
}

// TestLoopCauseCarriesPosition pins that an unrecognized loop names WHERE it
// is. The header's terminating instruction is an *ssa.If, whose Pos is never
// valid, so positioning the cause there produced a blocker nobody could locate
// — and `loop` is the top-ranked blocker class by graduation count.
func TestLoopCauseCarriesPosition(t *testing.T) {
	const src = `package input
func h(int) int
func f(n int) int {
	s := 0
	for n > 0 {
		s += n
		n = h(n)
	}
	return s
}`
	pkg, fset, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")

	for _, tc := range []struct {
		name  string
		cause []Cause
	}{
		{"time", loopCauses(mustCauses(t, fn))},
		{"space", loopCauses(mustSpaceCauses(t, fn))},
	} {
		if len(tc.cause) == 0 {
			t.Fatalf("%s: expected a loop cause", tc.name)
		}
		c := tc.cause[0]
		if !c.Pos.IsValid() {
			t.Fatalf("%s: loop cause position is invalid — the blocker cannot be located", tc.name)
		}
		if got := fset.Position(c.Pos).Line; got != 5 {
			t.Errorf("%s: loop cause at line %d, want 5 (the `for` statement)", tc.name, got)
		}
	}
}

// TestLoopCauseEmittedOncePerLoop pins that one loop yields one cause. The walk
// visits every block and asks for its enclosing loops, so a loop whose body
// spans many blocks used to emit one identical cause per block — measured at
// 104 on a single goldmark function.
func TestLoopCauseEmittedOncePerLoop(t *testing.T) {
	const src = `package input
func h(int) int
func f(n int, xs []int) int {
	s := 0
	for n > 0 {
		if s > 10 {
			s += 2
		} else {
			s += 3
		}
		for _, x := range xs {
			if x > 0 {
				s += x
			}
		}
		n = h(n)
	}
	return s
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")

	// The outer `for n > 0` is unrecognized; the inner range loop is bounded.
	// Exactly one cause, however many blocks the body spans.
	if got := len(loopCauses(mustCauses(t, fn))); got != 1 {
		t.Errorf("time: %d loop causes, want 1", got)
	}
	if got := len(loopCauses(mustSpaceCauses(t, fn))); got != 1 {
		t.Errorf("space: %d loop causes, want 1", got)
	}
}

// TestEveryCauseCarriesPosition is the regression pin for the CLASS of defect,
// not just the loop instance. A cause that cannot be located is a cause a
// consumer cannot act on, and this one survived from introduction because
// nothing asserted it.
//
// CauseNoBody is the one documented exemption: it is emitted for a function
// with no analyzable body, where the engine has no instruction to point at.
func TestEveryCauseCarriesPosition(t *testing.T) {
	exempt := map[CauseKind]string{
		CauseNoBody: "no body to point at",
	}
	srcs := map[string]string{
		"call":   "package input\nfunc h(int) int\nfunc f(n int) int { return h(n) }",
		"loop":   "package input\nfunc h(int) int\nfunc f(n int) int { s := 0; for n > 0 { s += n; n = h(n) }; return s }",
		"defer":  "package input\nfunc h(int) int\nfunc f(n int) int { defer func() { _ = h(n) }(); return 1 }",
		"go":     "package input\nfunc h(int) int\nfunc f(n int) int { go func() { _ = h(n) }(); return 1 }",
		"nobody": "package input\nfunc f(n int) int",
	}
	seen := map[CauseKind]bool{}
	for name, src := range srcs {
		pkg, _, err := ssasupport.Build(src)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		fn := ssasupport.Func(pkg, "f")
		if fn == nil {
			continue
		}
		var all []Cause
		_, tc := InferDetailed(fn, builtinModel{})
		all = append(all, tc...)
		_, sc := InferSpace(fn, nilSpace{})
		all = append(all, sc...)
		for _, c := range all {
			seen[c.Kind] = true
			if _, ok := exempt[c.Kind]; ok {
				continue
			}
			if !c.Pos.IsValid() {
				t.Errorf("%s: cause kind %v (%q) has no position", name, c.Kind, c.What)
			}
		}
	}
	// The test must not pass by covering nothing.
	for _, k := range []CauseKind{CauseCall, CauseLoop, CauseGo} {
		if !seen[k] {
			t.Errorf("cause kind %v was never exercised — the pin covers less than it claims", k)
		}
	}
}

func mustCauses(t *testing.T, fn *ssa.Function) []Cause {
	t.Helper()
	_, cs := InferDetailed(fn, builtinModel{})
	return cs
}

func mustSpaceCauses(t *testing.T, fn *ssa.Function) []Cause {
	t.Helper()
	_, cs := InferSpace(fn, nilSpace{})
	return cs
}

// TestLoopPosPrefersBodyBlockOverHeaderPhi pins the tier order below the guard
// condition. A range loop's guard is a tuple extract with no position, so the
// scan falls through — and the header's first positioned instruction is the
// induction phi, whose position is its VARIABLE'S DECLARATION. Here that is
// `var total int` on line 4, several lines from any loop.
//
// The loop's BODY block is inside the loop by construction, so it cannot name a
// declaration outside it. Measured across grpc-go and prometheus: it supplies a
// position for 96.7% and 97.5% of loops whose guard has none, and is earlier
// than the header phi in 2 cases out of ~23,600.
//
// The guarantee is INSIDE THE LOOP, not exactly the `for` line. Which of the two
// you get depends on how the range lowers: over a slice the element load is
// attributed to the `for` itself, while over a map the tuple extracts sit in the
// header and the body's first POSITIONED instruction is its first statement.
// Both identify the loop to a reader; the declaration and the signature do not.
func TestLoopPosPrefersBodyBlockOverHeaderPhi(t *testing.T) {
	const src = `package input
func h(int) int
func f(m map[string]int) int {
	total := 0
	for _, v := range m {
		total += h(v)
	}
	return total
}`
	pkg, fset, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	cs := loopCauses(mustCauses(t, fn))
	if len(cs) == 0 {
		t.Fatal("expected a loop cause for a range over a map with an unresolvable size")
	}
	got := fset.Position(cs[0].Pos).Line
	switch got {
	case 4:
		t.Fatalf("loop cause anchored on line 4 — `total := 0`, the induction " +
			"variable's declaration. That is the tier this test exists to demote.")
	case 3:
		t.Fatalf("loop cause anchored on line 3 — the function signature.")
	case 5, 6:
		// 5 is the `for`, 6 its first body statement. Both are the loop.
	default:
		t.Errorf("loop cause at line %d, want 5 or 6 (inside the loop)", got)
	}
}
