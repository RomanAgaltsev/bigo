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
