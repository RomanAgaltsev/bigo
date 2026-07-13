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

// Solve returns the solved asymptotic time bound (work) and the recursion-tree
// height (depth) of a self-recursive function in its own canonical size
// variables, or ok=false when no recurrence family applies (the caller falls
// back to ⊤). Depth is the true peak stack the space slice needs; work is the
// time bound every existing caller consumes.
func Solve(fn *ssa.Function, model engine.CostModel) (work bound.Bound, depth bound.Bound, ok bool) {
	r, ok := extract(fn, model)
	if !ok {
		return bound.Top(), bound.Top(), false
	}
	work, ok = solveWork(r)
	if !ok {
		return bound.Top(), bound.Top(), false
	}
	return work, depthOf(r), true
}

// solveWork solves the recurrence's closed-form time bound, selecting the solver
// family by step kind. (The former body of Solve, split out so Solve can also
// return depth.)
func solveWork(r rec) (bound.Bound, bool) {
	switch kindOf(r.terms) {
	case allSub:
		return solveSubtractive(r)
	case allDiv:
		if b, uniform := uniformDiv(r.terms); uniform {
			return solveMaster(r.mult, b, r.work, r.measure)
		}
		return solveAkraBazzi(ratiosOf(r.terms), r.work, r.measure) // unbalanced splits
	default:
		return bound.Top(), false // mixed subtractive/divisive: out of scope
	}
}

// depthOf is the recurrence-tree height — the true peak recursion (stack) depth:
// a subtractive step (n−c) unwinds O(n) levels; any divisive step (n/b, b>1)
// unwinds O(log n) levels. Called only when a recurrence family solved, so the
// step kind is allSub or allDiv (mixed is rejected upstream).
func depthOf(r rec) bound.Bound {
	switch kindOf(r.terms) {
	case allSub:
		return bound.Of(bound.Term(r.measure))
	default: // any divisive ratio > 1 gives logarithmic height
		return bound.Of(bound.Mono(r.measure, 0, 1))
	}
}
