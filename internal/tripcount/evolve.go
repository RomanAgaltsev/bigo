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
