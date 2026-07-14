package recurrence

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
)

// constFor wraps a CostModel, costing any call to one of the recursion targets
// as O(1) so level work can be measured with the recursive descent held
// constant. Sound only because target-calls are required to be at constant
// multiplicity (detection rejects calls under a size-loop).
type constFor struct {
	inner   engine.CostModel
	targets map[*ssa.Function]bool
}

func (s constFor) CallCost(c *ssa.CallCommon) bound.Bound {
	if callee := c.StaticCallee(); callee != nil && s.targets[callee] {
		return bound.Constant()
	}
	return s.inner.CallCost(c)
}

// localWorkExcluding is fn's intraprocedural bound with calls to any of the
// targets costed O(1). ok=false when the remaining body is itself ⊤.
func localWorkExcluding(fn *ssa.Function, model engine.CostModel, targets ...*ssa.Function) (bound.Bound, bool) {
	tset := make(map[*ssa.Function]bool, len(targets))
	for _, t := range targets {
		tset[t] = true
	}
	b := engine.Infer(fn, constFor{inner: model, targets: tset})
	if b.IsTop() {
		return bound.Top(), false
	}
	return b, true
}

// localWork is the single-function form: self-calls costed O(1). ok=false when
// the non-recursive body is itself ⊤.
func localWork(fn *ssa.Function, model engine.CostModel) (bound.Bound, bool) {
	return localWorkExcluding(fn, model, fn)
}
