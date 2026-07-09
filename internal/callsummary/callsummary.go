// Package callsummary resolves call costs via a cost table plus memoized,
// acyclic interprocedural summaries.
package callsummary

import (
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/costtable"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/size"
	"golang.org/x/tools/go/ssa"
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
	if callee.Pkg == nil { // external (no source): unknown
		return bound.Top()
	}
	return r.callUser(callee, c.Args)
}

func (r *Resolver) callUser(callee *ssa.Function, args []ssa.Value) bound.Bound {
	summary := r.summary(callee)
	if summary.IsTop() {
		return bound.Top()
	}
	rename := map[bound.Var]bound.Var{}
	for i, p := range callee.Params {
		if i >= len(args) {
			return bound.Top()
		}
		av, ok := size.Value(args[i])
		if !ok {
			// If the summary depends on this parameter's size, we cannot express
			// it in caller terms.
			if dependsOn(summary, p.Name()) {
				return bound.Top()
			}
			continue
		}
		// Map all of the parameter's possible size vars to the actual size.
		rename[size.Len(p.Name())] = av
		rename[size.Cap(p.Name())] = av
		rename[size.Num(p.Name())] = av
	}
	return summary.Subst(rename)
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
