package recurrence

// Package-internal two-function recursion cycles. A mutual pair is exactly:
// fn -> g and g -> fn by static calls, same package, neither self-recursive,
// and g unique. Anything else (3-cycles, multi-cycles, cross-package, dynamic
// edges) reads as "no partner" and stays ⊤ — see the mutual-recursion spec's
// §2 non-goals.
//
// Recorded SSA shapes (go/ssa dump, Task 3 Step 1 — the source of truth). A
// cross-member call's argument references the CALLER's *ssa.Parameter directly,
// so sizeStepOf(arg, callerParam) matches the same shapes as self-recursion:
//
//	IsEven->IsOdd  IsOdd(n-1)      arg: *ssa.BinOp{SUB, X:n(param), Y:const 1}   -> Sub 1
//	A->B           B(xs[:len/2])   arg: *ssa.Slice{X:xs(param), Low:nil, High:len/2} -> Div 2
//	B->A           A(xs[1:])       arg: *ssa.Slice{X:xs(param), Low:const 1, High:nil} -> Sub 1

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
)

// callsTo returns fn's static call sites whose callee is target.
func callsTo(fn, target *ssa.Function) []*ssa.CallCommon {
	var out []*ssa.CallCommon
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			if cc := callCommon(instr); cc != nil && cc.StaticCallee() == target {
				out = append(out, cc)
			}
		}
	}
	return out
}

// MutualPartner returns the unique two-cycle partner of fn, if any: the same-
// package function g such that fn statically calls g, g statically calls fn,
// neither is self-recursive, and no other function also forms a two-cycle with
// fn. A second distinct partner makes the SCC larger than two, which is out of
// scope, so the result is (nil, false).
func MutualPartner(fn *ssa.Function) (*ssa.Function, bool) {
	if fn == nil || len(fn.Blocks) == 0 || fn.Pkg == nil || IsSelfRecursive(fn) {
		return nil, false
	}
	var partner *ssa.Function
	seen := map[*ssa.Function]bool{}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			cc := callCommon(instr)
			if cc == nil {
				continue
			}
			g := cc.StaticCallee()
			if g == nil || g == fn || seen[g] {
				continue
			}
			seen[g] = true
			if g.Pkg != fn.Pkg || len(g.Blocks) == 0 || IsSelfRecursive(g) {
				continue
			}
			if len(callsTo(g, fn)) == 0 {
				continue // g does not call back: not a two-cycle member
			}
			if partner != nil {
				return nil, false // two distinct two-cycles through fn: ambiguous
			}
			partner = g
		}
	}
	return partner, partner != nil
}

// SolvePair solves the two-function cycle fn↔partner as a virtual
// self-recurrence in fn's measure vocabulary, routing the composed recurrence
// through the shipped solvers (solveWork/depthOf) — no new solver math. ok=false
// (⊤) when extraction fails a soundness precondition or the composed recurrence
// is out of the solvers' representable families (e.g. a≥2 subtractive).
func SolvePair(fn, partner *ssa.Function, model engine.CostModel) (bound.Bound, bound.Bound, bool) {
	r, ok := extractPair(fn, partner, model)
	if !ok {
		return bound.Top(), bound.Top(), false
	}
	// Branching factor per cycle traversal: calls fn→partner on a path × calls
	// partner→fn on a path. selfCallMult is a per-path MAX (an upper bound), so
	// the product over-approximates a at worst, which only inflates the exponent
	// — sound. Mirrors single-function extract, which sets rec.mult likewise;
	// solveWork reads mult for the divisive (Master) family and the composed
	// term count for the subtractive a≥2 rejection.
	r.mult = selfCallMult(fn, callsTo(fn, partner)) * selfCallMult(partner, callsTo(partner, fn))
	w, ok := solveWork(r)
	if !ok {
		return bound.Top(), bound.Top(), false
	}
	return w, depthOf(r), true
}

// pairEdge is one direction of the cycle: the callee position the measure
// lands in, the per-call steps, and the call sites.
type pairEdge struct {
	toIndex int        // parameter index the measure lands in at the callee
	steps   []sizeStep // one per call site (stepSame allowed)
	calls   []*ssa.CallCommon
}

// extractPair composes the two-cycle a→b→a into a virtual self-recurrence in
// a's measure vocabulary. ok=false on any failed soundness precondition; the
// caller falls back to ⊤. The composed-step termination argument is spec §4.3;
// divisive composition reuses the hardened ≥1 guard predicates (PR #44).
func extractPair(a, b *ssa.Function, model engine.CostModel) (rec, bool) {
	callsAB, callsBA := callsTo(a, b), callsTo(b, a)
	if len(callsAB) == 0 || len(callsBA) == 0 {
		return rec{}, false
	}
	// Constant multiplicity per member: no cycle call under a size loop.
	for _, pair := range []struct {
		fn    *ssa.Function
		calls []*ssa.CallCommon
	}{{a, callsAB}, {b, callsBA}} {
		forest := loopnest.Build(pair.fn)
		for _, c := range pair.calls {
			if underLoop(forest, callBlock(pair.fn, c)) {
				return rec{}, false
			}
		}
	}
	// Thread the measure: a param of A → a unique position in B → back to A.
	for pa, pA := range a.Params {
		edgeAB, ok := threadMeasure(pA, callsAB, b)
		if !ok {
			continue
		}
		pB := b.Params[edgeAB.toIndex]
		edgeBA, ok := threadMeasure(pB, callsBA, a)
		if !ok || edgeBA.toIndex != pa {
			continue // must land back in the SAME measure position of A
		}
		terms, ok := composeSteps(edgeAB.steps, edgeBA.steps)
		if !ok {
			continue
		}
		if !terminatesPair(a, pA, edgeAB.calls, b, pB, edgeBA.calls, terms) {
			continue
		}
		// Level work: both members, cycle calls held O(1), single-variable each.
		workA, ok := localWorkExcluding(a, model, a, b)
		if !ok || !varsSubset(workA, measureVar(pA)) {
			return rec{}, false
		}
		workB, ok := localWorkExcluding(b, model, a, b)
		if !ok || !varsSubset(workB, measureVar(pB)) {
			return rec{}, false
		}
		workB = workB.Subst(map[bound.Var]bound.Var{measureVar(pB): measureVar(pA)})
		return rec{
			measure: measureVar(pA),
			param:   pA,
			terms:   terms,
			work:    workA.Join(workB), // one traversal does A's level work + B's
		}, true
	}
	return rec{}, false
}

// threadMeasure finds the unique callee parameter index that every call feeds
// with a recognized step (Same/Sub/Div) of p. Growing or unrecognized args at
// the winning index — or an inconsistent index across calls — reject.
func threadMeasure(p *ssa.Parameter, calls []*ssa.CallCommon, callee *ssa.Function) (pairEdge, bool) {
	edge := pairEdge{toIndex: -1, calls: calls}
	for _, c := range calls {
		found := -1
		var st sizeStep
		for j, arg := range c.Args {
			if j >= len(callee.Params) {
				break
			}
			s := sizeStepOf(arg, p)
			if s.kind == stepBad {
				continue
			}
			if found >= 0 {
				return pairEdge{}, false // measure feeds two positions: ambiguous
			}
			found, st = j, s
		}
		if found < 0 {
			return pairEdge{}, false // this call does not thread the measure
		}
		if edge.toIndex < 0 {
			edge.toIndex = found
		} else if edge.toIndex != found {
			return pairEdge{}, false // inconsistent position across calls
		}
		edge.steps = append(edge.steps, st)
	}
	return edge, true
}

// composeSteps builds one composed cycle step per (A→B, B→A) call pair.
// Same∘Same is not a decrease; Sub∘Sub adds; Div∘Div multiplies; Sub∘Div is
// out of scope (spec §4.2). At least one composed term must be strict.
func composeSteps(ab, ba []sizeStep) ([]sizeStep, bool) {
	var terms []sizeStep
	for _, s := range ab {
		for _, t := range ba {
			c, ok := composeStep(s, t)
			if !ok {
				return nil, false
			}
			if c.kind != stepSame {
				terms = append(terms, c)
			}
		}
	}
	if len(terms) == 0 {
		return nil, false // no strict edge anywhere: the cycle never decreases
	}
	return terms, true
}

func composeStep(s, t sizeStep) (sizeStep, bool) {
	switch {
	case s.kind == stepSame && t.kind == stepSame:
		return sizeStep{kind: stepSame}, true
	case s.kind == stepSame:
		return t, true
	case t.kind == stepSame:
		return s, true
	case s.kind == stepSub && t.kind == stepSub:
		return sizeStep{kind: stepSub, sub: s.sub + t.sub}, true
	case s.kind == stepDiv && t.kind == stepDiv:
		return sizeStep{kind: stepDiv, div: s.div * t.div}, true
	default:
		return sizeStep{}, false // mixed Sub∘Div: rejected in v1
	}
}

// terminatesPair proves the composed cycle is well-founded (spec §4.3): one
// member's guard suffices, because the guard is evaluated once per cycle
// traversal, the composed measure strictly decreases across each traversal,
// and — for divisive composition — the ≥1 floor excludes the 0 fixed point, so
// the guarded member's recursion condition must eventually fail. A guard on the
// false side or on a non-measure condition proves nothing and correctly fails
// these predicates. Divisive composition requires the ≥1 predicate (integer:
// strict floor; slice: empty/short base guard) — the F1 lesson applied
// per-cycle. Subtractive-only integer cycles accept any lower-bound guard (a
// strictly −c sequence crosses any floor); subtractive slice cycles are
// structurally well-founded (xs[c:] panics at len 0).
func terminatesPair(a *ssa.Function, pA *ssa.Parameter, callsAB []*ssa.CallCommon,
	b *ssa.Function, pB *ssa.Parameter, callsBA []*ssa.CallCommon, terms []sizeStep) bool {
	strictDiv := hasDivStep(terms)
	if isSliceLike(pA.Type()) {
		if !strictDiv {
			return true // subtractive slice: structural panic base
		}
		return allGuardedSlice(a, pA, callsAB) || allGuardedSlice(b, pB, callsBA)
	}
	return allGuardedInt(a, pA, callsAB, strictDiv) || allGuardedInt(b, pB, callsBA, strictDiv)
}

func allGuardedInt(fn *ssa.Function, p *ssa.Parameter, calls []*ssa.CallCommon, strictDiv bool) bool {
	for _, c := range calls {
		if !guardedByMeasure(callBlock(fn, c), p, strictDiv) {
			return false
		}
	}
	return len(calls) > 0
}

func allGuardedSlice(fn *ssa.Function, p *ssa.Parameter, calls []*ssa.CallCommon) bool {
	for _, c := range calls {
		if !guardedBySliceBase(callBlock(fn, c), p) {
			return false
		}
	}
	return len(calls) > 0
}
