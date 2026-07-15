package callsummary

import (
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/tripcount"
)

// SSA shape (recorded per plan Task 1 Step 1, verified against go/ssa for
// `func Map(xs []int, f func(int) int)` with body `for _, v := range xs {
// out = append(out, f(v)) }`): the invocation `f(v)` lowers to an *ssa.Call
// whose Call.Value IS the *ssa.Parameter f (StaticCallee() == nil, Method ==
// nil). f's only referrer is that call. A func param handed to another call
// (composition) appears in that call's Call.Args. Passing a param to a
// non-call instruction (Store, MakeClosure, Phi, return, Send, MapUpdate)
// makes it escape.

// ParamSummary is a function's cost split into work not attributable to its
// func-typed parameters (Base) and, per func param, an upper bound on how many
// times that parameter is invoked (PerParam). Total cost at a call site is
// Base ⊔ Σ PerParam[i] × cost(argᵢ). Both parts are in the function's own
// canonical size variables and are renamed by substArgs at the call site.
//
// SOUNDNESS: PerParam[i] must upper-bound the true invocation count. The rule
// is a whitelist — a use of the param is either (a) a direct call, counted at
// the block's loop multiplier, (b) an argument to a static callee whose own
// ParamSummary is known (counts compose), or (c) anything else, which forces
// PerParam[i] = ⊤. Undercounting is structurally impossible: unrecognized
// uses poison, they are never skipped.
type ParamSummary struct {
	Base     bound.Bound
	PerParam map[int]bound.Bound
}

// funcParamIndex returns the index of v among fn's parameters when v is a
// func-typed parameter of fn.
func funcParamIndex(fn *ssa.Function, v ssa.Value) (int, bool) {
	p, ok := v.(*ssa.Parameter)
	if !ok {
		return 0, false
	}
	for i, fp := range fn.Params {
		if fp == p {
			if _, isSig := p.Type().Underlying().(*types.Signature); isSig {
				return i, true
			}
			return 0, false
		}
	}
	return 0, false
}

// paramSummaryOf computes fn's ParamSummary. ok=false when fn has no
// func-typed params, no body, or is already being summarized (cycle).
func (r *Resolver) paramSummaryOf(fn *ssa.Function) (ParamSummary, bool) {
	if fn == nil || len(fn.Blocks) == 0 {
		return ParamSummary{}, false
	}
	hasFuncParam := false
	for _, p := range fn.Params {
		if _, ok := p.Type().Underlying().(*types.Signature); ok {
			hasFuncParam = true
			break
		}
	}
	if !hasFuncParam {
		return ParamSummary{}, false
	}
	if ps, ok := r.paramMemo[fn]; ok {
		return ps, true
	}
	if r.onStack[fn] {
		return ParamSummary{}, false // cycle through a parametric function: ⊤ path
	}
	r.onStack[fn] = true
	defer func() { r.onStack[fn] = false }()

	forest := loopnest.Build(fn)
	if forest.UncoveredCycle(fn) {
		return ParamSummary{}, false
	}
	stab := fieldpath.Analyze(fn)
	per := map[int]bound.Bound{}
	joinPer := func(i int, b bound.Bound) {
		if cur, ok := per[i]; ok {
			per[i] = cur.Join(b)
		} else {
			per[i] = b
		}
	}

	// Pass A — count invocations and compositions, block by block.
	for _, blk := range fn.Blocks {
		factor := bound.Constant()
		for _, lp := range forest.EnclosingLoops(blk) {
			factor = factor.Mul(tripcount.Of(lp, stab))
		}
		for _, instr := range blk.Instrs {
			cc := callInstrCommon(instr) // Call/Defer common; Go handled by Base pass (⊤)
			if cc == nil {
				continue
			}
			if i, ok := funcParamIndex(fn, cc.Value); ok {
				joinPer(i, factor) // direct invocation (⊤ factor poisons, correctly)
				continue
			}
			// Composition: our func params appearing as arguments.
			callee := cc.StaticCallee()
			for ai, arg := range cc.Args {
				i, ok := funcParamIndex(fn, arg)
				if !ok {
					continue
				}
				if callee == nil {
					joinPer(i, bound.Top()) // handed to an unknown func value
					continue
				}
				calleePS, ok := r.paramSummaryOf(callee)
				if !ok {
					joinPer(i, bound.Top()) // callee not parametric/known: escape
					continue
				}
				cnt, ok := calleePS.PerParam[ai]
				if !ok {
					continue // callee provably never invokes it: contributes 0
				}
				names := paramNamesOf(callee)
				renamed := substArgs(cnt, names, cc.Args) // callee vocab -> ours
				joinPer(i, renamed.Mul(factor))
			}
		}
	}

	// Pass B — every OTHER use of a func param is an escape.
	for i, p := range fn.Params {
		if _, ok := p.Type().Underlying().(*types.Signature); !ok {
			continue
		}
		for _, ref := range *p.Referrers() {
			if !whitelistedUse(p, ref) {
				per[i] = bound.Top()
				break
			}
		}
	}

	// Base — the body with param invocations held O(1) and composed callees
	// costed at their Base.
	base := engine.Infer(fn, paramBaseModel{r: r, owner: fn})
	ps := ParamSummary{Base: base, PerParam: per}
	r.paramMemo[fn] = ps
	return ps, true
}

// callInstrCommon returns the CallCommon of Call/Defer instructions. Go is
// excluded on purpose: the Base pass scores *ssa.Go as ⊤ via the engine, and
// a goroutine-invoked param must not be "counted" as if synchronous (pin 4).
func callInstrCommon(instr ssa.Instruction) *ssa.CallCommon {
	switch v := instr.(type) {
	case *ssa.Call:
		return &v.Call
	case *ssa.Defer:
		return &v.Call
	}
	return nil
}

// whitelistedUse: the referrer is a Call/Defer that either invokes p directly
// or passes it as a plain argument (Pass A already priced that edge — as
// composition or as ⊤). Everything else (Store, MakeClosure capture, Phi,
// return, Go, comparison, map/chan ops) is an escape.
func whitelistedUse(p *ssa.Parameter, ref ssa.Instruction) bool {
	cc := callInstrCommon(ref)
	if cc == nil {
		return false
	}
	if cc.Value == ssa.Value(p) {
		return true
	}
	for _, a := range cc.Args {
		if a == ssa.Value(p) {
			return true
		}
	}
	return false
}

// paramBaseModel costs fn's own body for Base: direct param invocations are
// O(1) (their real cost is priced by PerParam × arg at the call site), and a
// static callee with a ParamSummary receiving our params is costed at its
// substituted Base for the same reason. Everything else defers to the
// resolver (⊤ propagates as usual, incl. *ssa.Go via the engine).
type paramBaseModel struct {
	r     *Resolver
	owner *ssa.Function
}

func (m paramBaseModel) CallCost(c *ssa.CallCommon) bound.Bound {
	if _, ok := funcParamIndex(m.owner, c.Value); ok {
		return bound.Constant()
	}
	if callee := c.StaticCallee(); callee != nil && passesOwnerFuncParam(m.owner, c) {
		if ps, ok := m.r.paramSummaryOf(callee); ok {
			return substArgs(ps.Base, paramNamesOf(callee), c.Args)
		}
		return bound.Top() // handing our param to a non-parametric callee
	}
	return m.r.CallCost(c)
}

func passesOwnerFuncParam(owner *ssa.Function, c *ssa.CallCommon) bool {
	for _, a := range c.Args {
		if _, ok := funcParamIndex(owner, a); ok {
			return true
		}
	}
	return false
}

func paramNamesOf(fn *ssa.Function) []string {
	names := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		names[i] = p.Name()
	}
	return names
}
