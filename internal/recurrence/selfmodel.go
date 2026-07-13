package recurrence

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
)

// selfConst wraps a CostModel, costing any call to the recursion target `self`
// as O(1) so localWork measures one level's work with the recursive descent
// held constant. Sound only because self-calls are required to be at constant
// multiplicity (detect.go rejects self-calls under a size-loop).
type selfConst struct {
	inner engine.CostModel
	self  *ssa.Function
}

func (s selfConst) CallCost(c *ssa.CallCommon) bound.Bound {
	if c.StaticCallee() == s.self {
		return bound.Constant()
	}
	return s.inner.CallCost(c)
}

// localWork is f(n): the function's intraprocedural bound with self-calls
// costed O(1). ok=false when the non-recursive body is itself ⊤.
func localWork(fn *ssa.Function, model engine.CostModel) (bound.Bound, bool) {
	b := engine.Infer(fn, selfConst{inner: model, self: fn})
	if b.IsTop() {
		return bound.Top(), false
	}
	return b, true
}
