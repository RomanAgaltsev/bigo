package callsummary

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/recurrence"
)

// SpaceResolver resolves interprocedural heap-space costs, memoized and acyclic,
// satisfying engine.SpaceModel. It mirrors Resolver's memo/onStack shape. A
// self-recursive function's true peak cost is stack depth, not heap; its own
// self-call is held at O(1) heap and its depth is added as the Stack term (top
// level) or, for callers, conservatively into the Within-only heap channel.
type SpaceResolver struct {
	memo      map[*ssa.Function]bound.Bound
	onStack   map[*ssa.Function]bool
	timeModel engine.CostModel // set by SpaceOf; resolves work for recurrence depth
}

// NewSpace returns a heap-space resolver. The parameter mirrors New's overrides
// signature for symmetry; space overrides are not modeled in this slice.
func NewSpace(_ map[*ssa.Function]bound.Bound) *SpaceResolver {
	return &SpaceResolver{
		memo:    map[*ssa.Function]bound.Bound{},
		onStack: map[*ssa.Function]bool{},
	}
}

// SpaceOf returns fn's full Space: the heap upper bound plus, for a self-recursive
// fn, the true peak stack depth from the recurrence solver. timeModel resolves
// call work for the depth computation — recurrence.Solve needs an engine.CostModel
// and the SpaceResolver is not one; the analyzer passes its time resolver. This
// is where the Stack term is injected, keeping engine free of a recurrence import.
func (r *SpaceResolver) SpaceOf(fn *ssa.Function, timeModel engine.CostModel) (engine.Space, []engine.Cause) {
	r.timeModel = timeModel
	sp, causes := engine.InferSpace(fn, r.heapModel(fn))
	if recurrence.IsSelfRecursive(fn) {
		_, depth, ok := recurrence.Solve(fn, timeModel)
		if !ok {
			// Proved recursive, depth unproven ⇒ ⊤. InferSpace's default Stack is
			// O(1), which for a function we have proved recurses is a *positive
			// claim* of constant stack, not the absence of one: every unsolvable
			// recursion (data-dependent partition, pointer structures, 3+ SCCs)
			// silently verified any space budget, O(1) included (issue #76).
			sp.Stack = bound.Top()
			causes = append(causes, engine.Cause{
				Pos:  fn.Pos(),
				Kind: engine.CauseCall,
				What: "recursion depth is unverifiable (no proven size measure)",
			})
			return sp, causes
		}
		sp.Stack = depth
	}
	return sp, causes
}

// CallSpace resolves a call's heap cost: the callee's summary rewritten into the
// caller's size variables. Closures and bodyless (external) callees are ⊤. A
// self-recursive callee also holds a transient peak stack of its own depth while
// it runs; a caller conservatively inherits that depth into the Within-only heap
// channel (never the Exceeds-driving Stack term), so omitting it can never make a
// budget falsely pass. Depth needs a time model; absent one it is skipped.
func (r *SpaceResolver) CallSpace(c *ssa.CallCommon) bound.Bound {
	callee := c.StaticCallee()
	if callee == nil || len(callee.Blocks) == 0 {
		return bound.Top() // closures / external: unknown space
	}
	summary := r.spaceSummary(callee)
	if summary.IsTop() {
		return bound.Top()
	}
	names := paramNames(callee)
	// substArgs is the same callee-summary-in-caller-vars rewrite the time
	// resolver uses (shared kind-for-kind logic — see Resolver.callUser).
	heap := substArgs(summary, names, c.Args)
	if r.timeModel != nil && recurrence.IsSelfRecursive(callee) {
		_, depth, ok := recurrence.Solve(callee, r.timeModel)
		if !ok {
			// The callee provably recurses to an unproven depth, and that depth is
			// live space while it runs. Inheriting nothing would under-count the
			// caller's space and hand it a false Within — the caller-side half of
			// issue #76.
			return bound.Top()
		}
		heap = heap.Join(substArgs(depth, names, c.Args))
	}
	return heap
}

// spaceSummary returns fn's memoized heap bound in its own canonical size vars.
// Self-calls are held at O(1) (heapModel), so a non-allocating recursive function
// summarizes as O(1) heap rather than ⊤; a call-graph cycle is still ⊤.
func (r *SpaceResolver) spaceSummary(fn *ssa.Function) bound.Bound {
	if b, ok := r.memo[fn]; ok {
		return b
	}
	if r.onStack[fn] {
		return bound.Top() // indirect call-graph cycle: recursion
	}
	r.onStack[fn] = true
	sp, _ := engine.InferSpace(fn, r.heapModel(fn))
	r.onStack[fn] = false
	r.memo[fn] = sp.Heap // this resolver summarizes heap only
	return sp.Heap
}

// heapModel is the space model to use when walking fn's heap: in a self-recursive
// fn, self-calls are held at O(1) (their unbounded descent is the Stack term, not
// heap), so the walk measures one frame's heap instead of diverging to ⊤.
func (r *SpaceResolver) heapModel(fn *ssa.Function) engine.SpaceModel {
	if recurrence.IsSelfRecursive(fn) {
		return selfConstSpace{inner: r, self: fn}
	}
	return r
}

// selfConstSpace costs any call to the recursion target self as O(1) heap, so
// InferSpace measures one frame's heap with the recursive descent held constant.
// Mirrors recurrence.selfConst on the space axis.
type selfConstSpace struct {
	inner engine.SpaceModel
	self  *ssa.Function
}

func (s selfConstSpace) CallSpace(c *ssa.CallCommon) bound.Bound {
	if c.StaticCallee() == s.self {
		return bound.Constant()
	}
	return s.inner.CallSpace(c)
}

// paramNames returns fn's parameter names, the substArgs rename keys.
func paramNames(fn *ssa.Function) []string {
	names := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		names[i] = p.Name()
	}
	return names
}
