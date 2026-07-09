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
		return bound.Constant()
	}
	forest := loopnest.Build(fn)

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

// blockCost is O(1) plus the model's cost for each call in the block.
func blockCost(b *ssa.BasicBlock, model CostModel) bound.Bound {
	cost := bound.Constant()
	for _, instr := range b.Instrs {
		if call, ok := instr.(*ssa.Call); ok {
			cost = cost.Join(model.CallCost(&call.Call))
		}
	}
	return cost
}
