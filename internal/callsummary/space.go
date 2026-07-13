package callsummary

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
)

// SpaceResolver resolves interprocedural heap-space costs, memoized and acyclic,
// satisfying engine.SpaceModel. It mirrors Resolver's memo/onStack shape;
// recursive space is ⊤ in this slice (recursion's true cost is stack depth,
// injected by the analyzer via recurrence).
type SpaceResolver struct {
	memo    map[*ssa.Function]bound.Bound
	onStack map[*ssa.Function]bool
}

// NewSpace returns a heap-space resolver. The parameter mirrors New's overrides
// signature for symmetry; space overrides are not modeled in this slice.
func NewSpace(_ map[*ssa.Function]bound.Bound) *SpaceResolver {
	return &SpaceResolver{
		memo:    map[*ssa.Function]bound.Bound{},
		onStack: map[*ssa.Function]bool{},
	}
}

// CallSpace resolves a call's heap cost: the callee's summary rewritten into the
// caller's size variables. Closures and bodyless (external) callees are ⊤.
func (r *SpaceResolver) CallSpace(c *ssa.CallCommon) bound.Bound {
	callee := c.StaticCallee()
	if callee == nil || len(callee.Blocks) == 0 {
		return bound.Top() // closures / external: unknown space
	}
	summary := r.spaceSummary(callee)
	if summary.IsTop() {
		return bound.Top()
	}
	names := make([]string, len(callee.Params))
	for i, p := range callee.Params {
		names[i] = p.Name()
	}
	// substArgs is the same callee-summary-in-caller-vars rewrite the time
	// resolver uses (shared kind-for-kind logic — see Resolver.callUser).
	return substArgs(summary, names, c.Args)
}

// spaceSummary returns fn's memoized heap bound in its own canonical size vars.
// A call-graph cycle (recursion) is ⊤ in this slice.
func (r *SpaceResolver) spaceSummary(fn *ssa.Function) bound.Bound {
	if b, ok := r.memo[fn]; ok {
		return b
	}
	if r.onStack[fn] {
		return bound.Top() // recursive space: ⊤ here (stack depth arrives via recurrence)
	}
	r.onStack[fn] = true
	sp, _ := engine.InferSpace(fn, r)
	r.onStack[fn] = false
	r.memo[fn] = sp.Heap // this slice summarizes heap only
	return sp.Heap
}
