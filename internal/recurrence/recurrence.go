// Package recurrence solves the standard recurrence families of self-recursive
// functions. A bound is returned only when the recursion's measure provably
// strictly decreases toward a base — a wrong answer here is a wrong bound on
// (possibly non-terminating) code, the highest-severity bug class.
package recurrence

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
)

// IsSelfRecursive reports whether fn calls itself directly (a static call whose
// callee is fn). Mutual recursion is out of scope and reads as false.
func IsSelfRecursive(fn *ssa.Function) bool {
	return len(selfCalls(fn)) > 0
}

// selfCalls returns every static call site in fn whose callee is fn itself,
// across Call/Defer/Go common instructions.
func selfCalls(fn *ssa.Function) []*ssa.CallCommon {
	var out []*ssa.CallCommon
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			cc := callCommon(instr)
			if cc != nil && cc.StaticCallee() == fn {
				out = append(out, cc)
			}
		}
	}
	return out
}

func callCommon(instr ssa.Instruction) *ssa.CallCommon {
	switch v := instr.(type) {
	case *ssa.Call:
		return &v.Call
	case *ssa.Defer:
		return &v.Call
	case *ssa.Go:
		return &v.Call
	}
	return nil
}

// Solve returns the solved asymptotic time bound of a self-recursive function
// in its own canonical size variables, or ok=false when no recurrence family
// applies (the caller falls back to ⊤). PR1: always (Top, false).
func Solve(fn *ssa.Function, model engine.CostModel) (bound.Bound, bool) {
	return bound.Top(), false
}
