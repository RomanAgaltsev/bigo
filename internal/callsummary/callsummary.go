// Package callsummary resolves call costs via a cost table plus memoized,
// acyclic interprocedural summaries.
package callsummary

import (
	"go/types"
	"sort"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/assume"
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/costtable"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/recurrence"
	"github.com/RomanAgaltsev/bigo/internal/size"
)

// Resolver implements engine.CostModel.
type Resolver struct {
	memo        map[*ssa.Function]bound.Bound
	onStack     map[*ssa.Function]bool
	overrides   map[*ssa.Function]bound.Bound
	methodCosts map[*types.Func]bound.Bound
	paramMemo   map[*ssa.Function]ParamSummary

	assume *assume.Set
	shadow map[string]bool
	stack  []*ssa.Function
	taint  map[*ssa.Function]bool
}

// New returns a resolver. overrides maps functions to asserted summaries (from
// //bigo:cost and //bigo:ignore), expressed in the callee's own canonical
// param size vars; nil is allowed. Overrides win over body analysis.
func New(overrides map[*ssa.Function]bound.Bound) *Resolver {
	if overrides == nil {
		overrides = map[*ssa.Function]bound.Bound{}
	}
	return &Resolver{
		memo:      map[*ssa.Function]bound.Bound{},
		onStack:   map[*ssa.Function]bool{},
		overrides: overrides,
		paramMemo: map[*ssa.Function]ParamSummary{},
		taint:     map[*ssa.Function]bool{},
	}
}

// NewWithMethods is New plus asserted costs for interface methods, keyed by
// the interface method object (//bigo:cost on the method declaration).
func NewWithMethods(overrides map[*ssa.Function]bound.Bound, methodCosts map[*types.Func]bound.Bound) *Resolver {
	r := New(overrides)
	if methodCosts == nil {
		methodCosts = map[*types.Func]bound.Bound{}
	}
	r.methodCosts = methodCosts
	return r
}

// UseAssumptions attaches an external assumption set (spec 2026-07-24). Nil
// leaves behavior identical to a resolver without assumptions.
func (r *Resolver) UseAssumptions(s *assume.Set) {
	r.assume = s
	r.shadow = map[string]bool{}
}

// AssumeWarnings returns shadowing warnings (an assumption whose target is
// already answered by a directive or a curated entry), sorted and unique.
// Silent shadowing is forbidden by the spec: it means the assumption is
// redundant or contradicts something with higher provenance.
func (r *Resolver) AssumeWarnings() []string {
	out := make([]string, 0, len(r.shadow))
	for w := range r.shadow {
		out = append(out, w)
	}
	sort.Strings(out)
	return out
}

func (r *Resolver) noteShadow(kind string, key string) {
	if r.assume != nil && r.assume.Has(key) {
		r.shadow["assumption for "+key+" is shadowed by a "+kind] = true
	}
}

// assumeCost answers a call from the assumption set, in the same way the
// methodCosts path answers an annotated interface method: normalized summary
// substituted into caller size variables.
func (r *Resolver) assumeCost(c *ssa.CallCommon, callee *ssa.Function) (bound.Bound, bool) {
	if r.assume == nil {
		return bound.Bound{}, false
	}
	key, ok := costtable.CalleeKey(c)
	if !ok {
		return bound.Bound{}, false
	}
	b, names, ok := r.assume.For(key, callee.Signature)
	if !ok {
		return bound.Bound{}, false
	}
	r.markTaint()
	return substArgs(b, names, c.Args), true
}

// Tainted reports whether fn's bound was computed with an assumed summary in
// its support, directly or transitively (provenance "assumption-tainted").
func (r *Resolver) Tainted(fn *ssa.Function) bool { return r.taint[fn] }

// markTaint taints the function whose analysis is currently on top of the
// inference stack — the single point through which every consult flows.
func (r *Resolver) markTaint() {
	if n := len(r.stack); n > 0 {
		r.taint[r.stack[n-1]] = true
	}
}

func (r *Resolver) pushInfer(fn *ssa.Function) { r.stack = append(r.stack, fn) }

// popInfer pops fn and propagates its taint to the new top (its consumer).
func (r *Resolver) popInfer(fn *ssa.Function) {
	r.stack = r.stack[:len(r.stack)-1]
	if r.taint[fn] {
		r.markTaint()
	}
}

// override returns the asserted summary for fn, looking through generic
// instantiations to their origin (annotations sit on the origin declaration).
func (r *Resolver) override(fn *ssa.Function) (bound.Bound, bool) {
	if b, ok := r.overrides[fn]; ok {
		return b, true
	}
	if o := fn.Origin(); o != nil && o != fn {
		if b, ok := r.overrides[o]; ok {
			return b, true
		}
	}
	return bound.Bound{}, false
}

// CallCost resolves a call's cost: cost table first, then user-function summary,
// else ⊤ (unverifiable).
func (r *Resolver) CallCost(c *ssa.CallCommon) bound.Bound {
	// Precedence (assumption spec §2): in-source directive > curated cost
	// table > assumption > inference. The plain table outranks directives in
	// this walk order, but their key spaces cannot collide (directives sit on
	// user functions, the table on builtins/stdlib), so the observable order
	// is the specified one.
	if b, ok := costtable.Lookup(c); ok {
		if key, kok := costtable.CalleeKey(c); kok {
			r.noteShadow("curated cost-table entry", key)
		}
		return b
	}
	callee := c.StaticCallee()
	if callee == nil {
		if c.Method != nil { // invoke mode: interface method call
			if summary, ok := r.methodCosts[c.Method]; ok {
				sig := c.Method.Type().(*types.Signature)
				names := make([]string, sig.Params().Len())
				for i := range names {
					names[i] = sig.Params().At(i).Name()
				}
				return substArgs(summary, names, c.Args)
			}
		}
		if b, ok := r.rangeFuncCost(c); ok { // range-over-func: seq(body) call
			return b
		}
		return bound.Top() // closure / func value / unannotated interface
	}
	if _, ok := r.override(callee); ok {
		if key, kok := costtable.FuncKey(callee); kok {
			r.noteShadow("//bigo: directive", key)
		}
		return r.callUser(callee, c.Args) // summary() will return the override
	}
	// No body to analyze: external (declared from export data) or an
	// instantiation of one. Pkg is not a proxy for this: instances always have
	// a nil Pkg, and imported functions have a non-nil Pkg with no blocks.
	if len(callee.Blocks) == 0 {
		if b, ok := r.parametricTableCost(c); ok {
			return b
		}
		if b, ok := r.assumeCost(c, callee); ok {
			return b
		}
		return bound.Top()
	}
	if b, ok := r.assumeCost(c, callee); ok {
		return b
	}
	if b, ok := r.parametricCallCost(callee, c); ok {
		return b
	}
	return r.callUser(callee, c.Args)
}

// InferTop returns fn's own asymptotic bound with diagnostic causes, for the
// top-level check of a function (as opposed to resolving a call to it). A
// self-recursive function's body walk mis-costs its self-call — that is the
// recursion, not a single call — so its true bound is the solved recurrence,
// the same value summary hands a caller. When the solver declines, the ⊤ and
// causes from the body walk drive the diagnostic. Non-recursive functions defer
// straight to engine.InferDetailed.
func (r *Resolver) InferTop(fn *ssa.Function) (bound.Bound, []engine.Cause) {
	r.pushInfer(fn)
	defer r.popInfer(fn)
	if recurrence.IsSelfRecursive(fn) {
		if solved, _, ok := recurrence.Solve(fn, r); ok {
			return solved, nil
		}
	}
	if partner, ok := recurrence.MutualPartner(fn); ok {
		if solved, _, ok := recurrence.SolvePair(fn, partner, r); ok {
			return solved, nil
		}
	}
	return engine.InferDetailed(fn, r)
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
	// Field-path sizes are frame-local: re-rooting a callee's len(s.items)
	// into caller vocabulary needs caller-side stability from entry to the
	// call site, which does not exist yet (design spec §2 non-goal). Refuse
	// rather than leak a var the caller cannot interpret.
	for _, m := range summary.Terms() {
		for _, mv := range m.Vars() {
			if size.IsFieldPath(mv) {
				return bound.Top()
			}
		}
	}
	rename := make(map[bound.Var]bound.Var, len(paramNames))
	for i, name := range paramNames {
		if i >= len(args) {
			return bound.Top()
		}
		av, class, ok := size.ValueClass(args[i])
		sliceParam, _ := args[i].(*ssa.Parameter)
		isSliceArg := ok && sliceParam != nil && isSliceParam(sliceParam)
		if !ok {
			// A parameter captured by a read-only closure reads back as a load
			// of its spill cell; recover its entry-stable size (never its cap).
			if sv, cl, sok := fieldpath.SpillArgSize(args[i]); sok {
				av, class, ok = sv, cl, true
			}
		}
		switch {
		case ok && class == size.Length:
			rename[size.Len(name)] = av
			if isSliceArg {
				rename[size.Cap(name)] = size.Cap(sliceParam.Name())
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

// summary returns the function's asserted or inferred bound, memoized, with
// recursion -> ⊤. An override (//bigo:cost, //bigo:ignore) short-circuits
// body analysis entirely — that is the point of the annotation.
func (r *Resolver) summary(fn *ssa.Function) bound.Bound {
	if b, ok := r.override(fn); ok {
		return b
	}
	if b, ok := r.memo[fn]; ok {
		if r.taint[fn] {
			r.markTaint() // a memoized tainted summary taints its consumer
		}
		return b
	}
	if r.onStack[fn] {
		return bound.Top() // call-graph cycle: recursion
	}
	r.onStack[fn] = true
	r.pushInfer(fn)
	defer r.popInfer(fn)
	if recurrence.IsSelfRecursive(fn) {
		if b, _, ok := recurrence.Solve(fn, r); ok {
			r.onStack[fn] = false
			r.memo[fn] = b
			return b
		}
	}
	if partner, ok := recurrence.MutualPartner(fn); ok {
		if b, _, ok := recurrence.SolvePair(fn, partner, r); ok {
			r.onStack[fn] = false
			r.memo[fn] = b
			return b
		}
	}
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
