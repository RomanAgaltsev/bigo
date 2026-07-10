// Package callsummary resolves call costs via a cost table plus memoized,
// acyclic interprocedural summaries.
package callsummary

import (
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/costtable"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/size"
)

// Resolver implements engine.CostModel.
type Resolver struct {
	memo    map[*ssa.Function]bound.Bound
	onStack map[*ssa.Function]bool
}

// New returns an empty resolver.
func New() *Resolver {
	return &Resolver{
		memo:    map[*ssa.Function]bound.Bound{},
		onStack: map[*ssa.Function]bool{},
	}
}

// CallCost resolves a call's cost: cost table first, then user-function summary,
// else ⊤ (unverifiable).
func (r *Resolver) CallCost(c *ssa.CallCommon) bound.Bound {
	if b, ok := costtable.Lookup(c); ok {
		return b
	}
	callee := c.StaticCallee()
	if callee == nil {
		return bound.Top() // interface / closure / dynamic dispatch
	}
	// No body to analyze: external (declared from export data) or an
	// instantiation of one. Pkg is not a proxy for this: instances always have a
	// nil Pkg, and imported functions have a non-nil Pkg with no blocks.
	if len(callee.Blocks) == 0 {
		return bound.Top()
	}
	return r.callUser(callee, c.Args)
}

func (r *Resolver) callUser(callee *ssa.Function, args []ssa.Value) bound.Bound {
	summary := r.summary(callee)
	if summary.IsTop() {
		return bound.Top()
	}
	names := make([]string, len(callee.Params))
	for i, p := range callee.Params {
		names[i] = p.Name()
	}
	return substArgs(summary, names, args)
}

// substArgs rewrites a callee summary into caller size variables, kind for
// kind. len(p) becomes the argument's length var. cap(p) becomes cap(arg)
// only when the argument is itself a slice parameter (the slice header is
// copied, so the capacities are equal) — a length is NOT an upper bound on a
// capacity, so no other substitution for cap is sound. A numeric p becomes
// the argument's numeric var. Any parameter the summary depends on that the
// caller cannot express makes the whole call unverifiable.
func substArgs(summary bound.Bound, paramNames []string, args []ssa.Value) bound.Bound {
	rename := map[bound.Var]bound.Var{}
	for i, name := range paramNames {
		if i >= len(args) {
			return bound.Top()
		}
		av, class, ok := size.ValueClass(args[i])
		switch {
		case ok && class == size.Length:
			rename[size.Len(name)] = av
			if ap, isParam := args[i].(*ssa.Parameter); isParam && isSliceParam(ap) {
				rename[size.Cap(name)] = size.Cap(ap.Name())
			} else if dependsOnVar(summary, size.Cap(name)) {
				return bound.Top()
			}
		case ok: // Numeric
			rename[size.Num(name)] = av
		default:
			if dependsOn(summary, name) {
				return bound.Top()
			}
		}
	}
	return summary.Subst(rename)
}

func isSliceParam(p *ssa.Parameter) bool {
	_, ok := p.Type().Underlying().(*types.Slice)
	return ok
}

// dependsOnVar reports whether the bound references the variable v.
func dependsOnVar(b bound.Bound, v bound.Var) bool {
	for _, m := range b.Terms() {
		for _, mv := range m.Vars() {
			if mv == v {
				return true
			}
		}
	}
	return false
}

// summary returns engine.Infer(fn, r), memoized, with recursion -> ⊤.
func (r *Resolver) summary(fn *ssa.Function) bound.Bound {
	if b, ok := r.memo[fn]; ok {
		return b
	}
	if r.onStack[fn] {
		return bound.Top() // call-graph cycle: recursion
	}
	r.onStack[fn] = true
	b := engine.Infer(fn, r)
	r.onStack[fn] = false
	r.memo[fn] = b
	return b
}

// dependsOn reports whether the bound references any size variable of param p.
func dependsOn(b bound.Bound, p string) bool {
	if b.IsTop() {
		return true
	}
	want := map[bound.Var]bool{size.Len(p): true, size.Cap(p): true, size.Num(p): true}
	for _, m := range b.Terms() {
		for _, v := range m.Vars() {
			if want[v] {
				return true
			}
		}
	}
	return false
}
