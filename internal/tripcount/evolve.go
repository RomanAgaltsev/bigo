package tripcount

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// ruleIncreasing — R1, the generalized counted loop.
//
// Shape: `ind ⋖ e` (LSS/LEQ, or GTR/GEQ with sides swapped) where ind is an
// affine image (constant coefficients — the S1 constraint) of a header phi
// whose every non-step edge has a provable CONSTANT lower bound (the B1
// constraint, generalized from "is a constant") and whose steps add positive
// constants; e resolves to a dominating extent.
//
// Claim: linear, O(upper(e)). Argument: the phi grows by >= 1 per iteration
// from >= some constant c, the affine comparand preserves monotonicity, and
// the guard fails once the comparand reaches e <= upper(e): trips are at most
// upper(e) - c + O(1).
func ruleIncreasing(sh *shape) (bound.Bound, bool) {
	if sh.cmp == nil {
		return bound.Bound{}, false
	}
	var indV, boundV ssa.Value
	switch sh.cmp.Op {
	case token.LSS, token.LEQ:
		indV, boundV = sh.cmp.X, sh.cmp.Y
	case token.GTR, token.GEQ:
		indV, boundV = sh.cmp.Y, sh.cmp.X
	default:
		return bound.Bound{}, false
	}
	phi, ok := affineOfPhi(indV)
	if !ok || phi.Block() != sh.loop.Header || !isIncreasingInductionPhi(sh, phi) {
		return bound.Bound{}, false
	}
	v, ok := sh.f.upperExtent(boundV, 0)
	if !ok {
		return bound.Bound{}, false
	}
	return bound.Of(bound.Term(v)), true
}

// affineOfPhi unwraps v = phi, phi+b, b+phi, a*phi, a*phi+b for constant
// a >= 1, b >= 0. Constant coefficients only: a variable offset shifts the
// trip count by an unbounded amount (finding S1).
func affineOfPhi(v ssa.Value) (*ssa.Phi, bool) {
	if p, ok := v.(*ssa.Phi); ok {
		return p, true
	}
	bo, ok := v.(*ssa.BinOp)
	if !ok {
		return nil, false
	}
	switch bo.Op {
	case token.ADD:
		if c, ok := constIntV(bo.Y); ok && c >= 0 {
			return mulOfPhi(bo.X)
		}
		if c, ok := constIntV(bo.X); ok && c >= 0 {
			return mulOfPhi(bo.Y)
		}
	case token.MUL:
		return mulOfPhi(bo)
	}
	return nil, false
}

// mulOfPhi unwraps v = phi or a*phi (const a >= 1).
func mulOfPhi(v ssa.Value) (*ssa.Phi, bool) {
	if p, ok := v.(*ssa.Phi); ok {
		return p, true
	}
	bo, ok := v.(*ssa.BinOp)
	if !ok || bo.Op != token.MUL {
		return nil, false
	}
	if c, ok := constIntV(bo.X); ok && c >= 1 {
		p, ok := bo.Y.(*ssa.Phi)
		return p, ok
	}
	if c, ok := constIntV(bo.Y); ok && c >= 1 {
		p, ok := bo.X.(*ssa.Phi)
		return p, ok
	}
	return nil, false
}

// isIncreasingInductionPhi: every edge is a positive-constant step or has a
// provable constant lower bound; at least one of each. A parameter init has
// no constant lower bound, so B1's `for i := m; i < n` stays rejected.
func isIncreasingInductionPhi(sh *shape, phi *ssa.Phi) bool {
	hasStep, hasInit := false, false
	for _, e := range phi.Edges {
		if isPositiveStep(phi, e) {
			hasStep = true
			continue
		}
		if _, ok := sh.f.lowerBoundConst(e, 0); !ok {
			return false
		}
		hasInit = true
	}
	return hasStep && hasInit
}

// ruleDecreasing — R2, the decreasing counted loop.
//
// Shape: `phi ⋗ c` (GTR/GEQ, or LSS/LEQ with sides swapped) for a CONSTANT c;
// every non-init edge subtracts a positive constant (phi-c or phi+negc);
// every init edge resolves to a dominating extent. A non-constant lower
// bound stays ⊤ — the mirror image of B1's parameter start.
//
// Claim: linear, O(upper(init)). Argument: the value starts <= upper(init),
// drops by >= 1 per iteration, and the guard fails at the constant.
func ruleDecreasing(sh *shape) (bound.Bound, bool) {
	if sh.cmp == nil {
		return bound.Bound{}, false
	}
	var indV, lowV ssa.Value
	switch sh.cmp.Op {
	case token.GTR, token.GEQ:
		indV, lowV = sh.cmp.X, sh.cmp.Y
	case token.LSS, token.LEQ:
		indV, lowV = sh.cmp.Y, sh.cmp.X
	default:
		return bound.Bound{}, false
	}
	if _, ok := constIntV(lowV); !ok {
		return bound.Bound{}, false
	}
	phi, ok := indV.(*ssa.Phi)
	if !ok || phi.Block() != sh.loop.Header {
		return bound.Bound{}, false
	}
	var extent bound.Var
	hasStep, hasInit := false, false
	for _, e := range phi.Edges {
		if isNegStep(phi, e) {
			hasStep = true
			continue
		}
		v, ok := sh.f.upperExtent(e, 0)
		if !ok {
			return bound.Bound{}, false
		}
		if hasInit && v != extent {
			return bound.Bound{}, false
		}
		extent, hasInit = v, true
	}
	if !hasStep || !hasInit {
		return bound.Bound{}, false
	}
	return bound.Of(bound.Term(extent)), true
}

// isNegStep reports whether e is phi - c (c > 0) or phi + c (c < 0).
func isNegStep(phi *ssa.Phi, e ssa.Value) bool {
	bo, ok := e.(*ssa.BinOp)
	if !ok {
		return false
	}
	switch bo.Op {
	case token.SUB:
		if bo.X == phi {
			c, ok := constIntV(bo.Y)
			return ok && c > 0
		}
	case token.ADD:
		switch {
		case bo.X == phi:
			c, ok := constIntV(bo.Y)
			return ok && c < 0
		case bo.Y == phi:
			c, ok := constIntV(bo.X)
			return ok && c < 0
		}
	}
	return false
}

// ruleGeometricUp — R3.
//
// Shape: upper-bound comparison of an affine image of phi (as R1); every
// non-init edge multiplies phi by a constant k >= 2, optionally adding a
// constant d >= 0; init edges have constant lower bounds, >= 1, or >= 0 when
// EVERY step has d >= 1 (a start of 0 under pure multiplication never grows —
// the classic infinite loop, pinned in TestLoopAlgebraStaysTop).
//
// Claim: logarithmic, O(log(upper(e))). Argument: from >= 1 each step at
// least doubles the value (or maps 0 -> >= 1 first), so the affine comparand
// exceeds upper(e) within log2(upper(e)) + O(1) iterations.
func ruleGeometricUp(sh *shape) (bound.Bound, bool) {
	if sh.cmp == nil {
		return bound.Bound{}, false
	}
	var indV, boundV ssa.Value
	switch sh.cmp.Op {
	case token.LSS, token.LEQ:
		indV, boundV = sh.cmp.X, sh.cmp.Y
	case token.GTR, token.GEQ:
		indV, boundV = sh.cmp.Y, sh.cmp.X
	default:
		return bound.Bound{}, false
	}
	phi, ok := affineOfPhi(indV)
	if !ok || phi.Block() != sh.loop.Header {
		return bound.Bound{}, false
	}
	minInit, allD1 := int64(0), true
	hasStep, hasInit := false, false
	for _, e := range phi.Edges {
		if _, d, ok := mulStep(phi, e); ok {
			hasStep = true
			if d < 1 {
				allD1 = false
			}
			continue
		}
		lo, ok := sh.f.lowerBoundConst(e, 0)
		if !ok {
			return bound.Bound{}, false
		}
		if !hasInit || lo < minInit {
			minInit = lo
		}
		hasInit = true
	}
	if !hasStep || !hasInit {
		return bound.Bound{}, false
	}
	if minInit < 1 && (minInit < 0 || !allD1) {
		return bound.Bound{}, false
	}
	v, ok := sh.f.upperExtent(boundV, 0)
	if !ok {
		return bound.Bound{}, false
	}
	return bound.Of(bound.Mono(v, 0, 1)), true
}

// mulStep reports whether e is a geometric step of phi — e = k*phi + d for
// consts k >= 2, d >= 0 — possibly SELECTED by an intermediate phi whose every
// edge is itself such a step (the sift-down `c = 2i+1 or 2i+2` merge: both
// alternatives multiply, so the >=-doubling growth argument still holds). The
// returned d is the minimum across a merge's arms — the worst case for a value
// escaping a zero start, which is all ruleGeometricUp consults.
func mulStep(phi *ssa.Phi, e ssa.Value) (k, d int64, ok bool) {
	k, d, ok = mulStepRaw(phi, e, 0)
	if !ok || d < 0 {
		return 0, 0, false
	}
	return k, d, true
}

// mulStepRaw returns the exact (k, d) of e = k*phi + d (sign of d checked by
// mulStep), descending through intermediate phis. depth guards against cyclic
// SSA; a header-phi self-reference (an identity, non-growing edge) is rejected.
func mulStepRaw(phi *ssa.Phi, e ssa.Value, depth int) (k, d int64, ok bool) {
	if depth > maxFactsDepth {
		return 0, 0, false
	}
	if p, isPhi := e.(*ssa.Phi); isPhi {
		if p == phi {
			return 0, 0, false
		}
		var minK, minD int64
		has := false
		for _, edge := range p.Edges {
			ek, ed, eok := mulStepRaw(phi, edge, depth+1)
			if !eok {
				return 0, 0, false
			}
			if !has || ek < minK {
				minK = ek
			}
			if !has || ed < minD {
				minD = ed
			}
			has = true
		}
		if !has {
			return 0, 0, false
		}
		return minK, minD, true
	}
	return affineMul(phi, e)
}

// affineMul matches e = k*phi + d for a const k >= 2 and integer d (its sign is
// checked by mulStep), peeling constant ADD layers so a nested (2*phi+1)+1
// reads as k=2, d=2.
func affineMul(phi *ssa.Phi, e ssa.Value) (k, d int64, ok bool) {
	bo, isBin := e.(*ssa.BinOp)
	if !isBin {
		return 0, 0, false
	}
	switch bo.Op {
	case token.MUL:
		if kk, ok := mulOf(phi, bo); ok {
			return kk, 0, true
		}
	case token.ADD:
		if c, isC := constIntV(bo.Y); isC {
			if kk, dd, ok := affineMul(phi, bo.X); ok {
				return kk, dd + c, true
			}
		}
		if c, isC := constIntV(bo.X); isC {
			if kk, dd, ok := affineMul(phi, bo.Y); ok {
				return kk, dd + c, true
			}
		}
	}
	return 0, 0, false
}

// mulOf matches v = k*phi for const k >= 2.
func mulOf(phi *ssa.Phi, v ssa.Value) (int64, bool) {
	bo, ok := v.(*ssa.BinOp)
	if !ok || bo.Op != token.MUL {
		return 0, false
	}
	switch {
	case bo.X == phi:
		c, ok := constIntV(bo.Y)
		return c, ok && c >= 2
	case bo.Y == phi:
		c, ok := constIntV(bo.X)
		return c, ok && c >= 2
	}
	return 0, false
}

// ruleGeometricDown — R4.
//
// Shape: `phi > c` (c >= 0) or `phi >= c` (c >= 1) — the asymmetry matters:
// under >= 0, a value of 0 divides to 0 forever. Every non-init edge is
// phi/k or (phi-d)/k for consts k >= 2, d >= 0; inits resolve to extents
// (a parameter init is fine here: an integer parameter is its own extent,
// which is why SiftUp infers O(log(i)) rather than O(log(len(h)))).
//
// Claim: logarithmic, O(log(upper(init))). Argument: for values above the
// constant guard (>= 1 in both accepted forms), floor division by k >= 2
// (after subtracting a non-negative d) at least halves the value; truncation
// only accelerates the descent.
func ruleGeometricDown(sh *shape) (bound.Bound, bool) {
	if sh.cmp == nil {
		return bound.Bound{}, false
	}
	var indV, lowV ssa.Value
	var op token.Token
	switch sh.cmp.Op {
	case token.GTR, token.GEQ:
		indV, lowV, op = sh.cmp.X, sh.cmp.Y, sh.cmp.Op
	case token.LSS, token.LEQ:
		indV, lowV = sh.cmp.Y, sh.cmp.X
		op = map[token.Token]token.Token{token.LSS: token.GTR, token.LEQ: token.GEQ}[sh.cmp.Op]
	default:
		return bound.Bound{}, false
	}
	c, ok := constIntV(lowV)
	if !ok || (op == token.GTR && c < 0) || (op == token.GEQ && c < 1) {
		return bound.Bound{}, false
	}
	phi, ok := indV.(*ssa.Phi)
	if !ok || phi.Block() != sh.loop.Header {
		return bound.Bound{}, false
	}
	var extent bound.Var
	hasStep, hasInit := false, false
	for _, e := range phi.Edges {
		if divStep(phi, e) {
			hasStep = true
			continue
		}
		v, ok := sh.f.upperExtent(e, 0)
		if !ok {
			return bound.Bound{}, false
		}
		if hasInit && v != extent {
			return bound.Bound{}, false
		}
		extent, hasInit = v, true
	}
	if !hasStep || !hasInit {
		return bound.Bound{}, false
	}
	return bound.Of(bound.Mono(extent, 0, 1)), true
}

// divStep matches e = phi/k or (phi-d)/k for consts k >= 2, d >= 0.
func divStep(phi *ssa.Phi, e ssa.Value) bool {
	bo, ok := e.(*ssa.BinOp)
	if !ok || bo.Op != token.QUO {
		return false
	}
	k, ok := constIntV(bo.Y)
	if !ok || k < 2 {
		return false
	}
	if bo.X == phi {
		return true
	}
	sub, ok := bo.X.(*ssa.BinOp)
	if !ok || sub.Op != token.SUB || sub.X != phi {
		return false
	}
	d, ok := constIntV(sub.Y)
	return ok && d >= 0
}
