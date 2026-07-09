// Package engine composes an intraprocedural asymptotic time bound for a function.
package engine

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/tripcount"
)

// Infer returns the function's intraprocedural time bound.
func Infer(fn *ssa.Function) bound.Bound {
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
		contrib := blockCost(b).Mul(factor)
		if !started {
			total = contrib
			started = true
			continue
		}
		total = total.Join(contrib)
	}
	return total
}

// blockCost os O(1) unless the block contains an unresolved call (any call other
// than the len/cap builtins, in which case it is Top (unverifiable).
func blockCost(b *ssa.BasicBlock) bound.Bound {
	for _, instr := range b.Instrs {
		call, ok := instr.(*ssa.Call)
		if !ok {
			continue
		}
		if bi, ok := call.Call.Value.(*ssa.Builtin); ok {
			switch bi.Name() {
			case "len", "cap":
				continue
			}
		}
		return bound.Top()
	}
	return bound.Constant()
}
