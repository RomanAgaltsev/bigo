// Package engine composes an intraprocedural asymptotic time bound for a function.
package engine

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/tripcount"
)

// CostModel resolves the cost of a call in canonical size variables.
type CostModel interface {
	CallCost(c *ssa.CallCommon) bound.Bound
}

// Infer returns the function's intraprocedural time bound, delegating call costs
// to model. Model: Σ_blocks blockCost(b) × Π(trip-counts of enclosing loops);
// ⊤ is absorbing, so any ⊤ call cost inside a loop makes the function ⊤.
func Infer(fn *ssa.Function, model CostModel) bound.Bound {
	if fn == nil || len(fn.Blocks) == 0 {
		return bound.Top() // no body: nothing is known (assembly, external linkage)
	}
	forest := loopnest.Build(fn)

	if forest.UncoveredCycle(fn) {
		return bound.Top() // irreducible control flow: no trip count exists
	}

	total := bound.Constant()
	started := false
	for _, b := range fn.Blocks {
		factor := bound.Constant()
		for _, lp := range forest.EnclosingLoops(b) {
			factor = factor.Mul(tripcount.Of(lp))
		}
		contrib := blockCost(b, model).Mul(factor)
		if !started {
			total = contrib
			started = true
			continue
		}
		total = total.Join(contrib)
	}
	return total
}

// blockCost is O(1) plus the model's cost for each call-shaped instruction.
// Deferred calls are joined like plain calls: they all run at function exit,
// and the enclosing-loop factor applied by Infer upper-bounds "one deferred
// call per iteration". A go statement makes the block unverifiable — v1 does
// not model concurrent work, even when the callee is resolvable.
func blockCost(b *ssa.BasicBlock, model CostModel) bound.Bound {
	cost := bound.Constant()
	for _, instr := range b.Instrs {
		switch v := instr.(type) {
		case *ssa.Call:
			cost = cost.Join(model.CallCost(&v.Call))
		case *ssa.Defer:
			cost = cost.Join(model.CallCost(&v.Call))
		case *ssa.Go:
			return bound.Top()
		}
	}
	return cost
}
