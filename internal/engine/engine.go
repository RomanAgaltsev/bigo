// Package engine composes an intraprocedural asymptotic time bound for a function.
package engine

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/tripcount"
)

// CostModel resolves the cost of a call in canonical size variables.
type CostModel interface {
	CallCost(c *ssa.CallCommon) bound.Bound
}

// Cause records why a bound became unverifiable: the source position and a
// human-readable description of the unresolved construct.
type Cause struct {
	Pos  token.Pos
	What string
}

// Infer returns the function's intraprocedural time bound, delegating call
// costs to model. Model: Σ_blocks blockCost(b) × Π(trip-counts of enclosing
// loops); ⊤ is absorbing, so any ⊤ anywhere makes the function ⊤.
func Infer(fn *ssa.Function, model CostModel) bound.Bound {
	b, _ := InferDetailed(fn, model)
	return b
}

// InferDetailed is Infer plus the reasons the bound (when ⊤) is unverifiable.
// Causes are nil when the bound is not ⊤.
func InferDetailed(fn *ssa.Function, model CostModel) (bound.Bound, []Cause) {
	if fn == nil || len(fn.Blocks) == 0 {
		return bound.Top(), []Cause{{What: "function has no analyzable body"}}
	}
	forest := loopnest.Build(fn)
	if forest.UncoveredCycle(fn) {
		return bound.Top(), []Cause{{Pos: fn.Pos(), What: "irreducible control flow (goto into a cycle)"}}
	}

	var causes []Cause
	total := bound.Constant()
	started := false
	for _, b := range fn.Blocks {
		factor := bound.Constant()
		for _, lp := range forest.EnclosingLoops(b) {
			tc := tripcount.Of(lp)
			if tc.IsTop() {
				causes = append(causes, Cause{
					Pos:  lp.Header.Instrs[len(lp.Header.Instrs)-1].Pos(),
					What: "loop with unrecognized trip count",
				})
			}
			factor = factor.Mul(tc)
		}
		bc, bcauses := blockCost(b, model)
		causes = append(causes, bcauses...)
		contrib := bc.Mul(factor)
		if !started {
			total = contrib
			started = true
			continue
		}
		total = total.Join(contrib)
	}
	if !total.IsTop() {
		return total, nil
	}
	return total, causes
}

// blockCost is O(1) plus the model's cost for each call-shaped instruction.
// Deferred calls are joined like plain calls: they all run at function exit,
// and the enclosing-loop factor applied by InferDetailed upper-bounds "one
// deferred call per iteration". A go statement makes the block unverifiable —
// v1 does not model concurrent work (spec §9).
func blockCost(b *ssa.BasicBlock, model CostModel) (bound.Bound, []Cause) {
	cost := bound.Constant()
	var causes []Cause
	for _, instr := range b.Instrs {
		switch v := instr.(type) {
		case *ssa.Call:
			c := model.CallCost(&v.Call)
			if c.IsTop() {
				causes = append(causes, Cause{Pos: v.Pos(), What: "unresolved cost at call to " + calleeName(&v.Call)})
			}
			cost = cost.Join(c)
		case *ssa.Defer:
			c := model.CallCost(&v.Call)
			if c.IsTop() {
				causes = append(causes, Cause{Pos: v.Pos(), What: "unresolved cost at deferred call to " + calleeName(&v.Call)})
			}
			cost = cost.Join(c)
		case *ssa.Go:
			causes = append(causes, Cause{Pos: v.Pos(), What: "goroutine launch (concurrency is unverifiable in v1)"})
			return bound.Top(), causes
		}
	}
	return cost, causes
}

// calleeName is a best-effort human-readable name for a call target.
func calleeName(c *ssa.CallCommon) string {
	if c.Method != nil {
		return c.Method.Name()
	}
	if f := c.StaticCallee(); f != nil {
		return f.Name()
	}
	if c.Value != nil {
		if n := c.Value.Name(); n != "" {
			return n
		}
	}
	return "unknown callee"
}
