package tripcount

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/size"
	"github.com/RomanAgaltsev/bigo/internal/sizefacts"
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
	v, ok := sh.f.UpperExtent(boundV, 0)
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
		if c, ok := sizefacts.ConstIntV(bo.Y); ok && c >= 0 {
			return mulOfPhi(bo.X)
		}
		if c, ok := sizefacts.ConstIntV(bo.X); ok && c >= 0 {
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
	if c, ok := sizefacts.ConstIntV(bo.X); ok && c >= 1 {
		p, ok := bo.Y.(*ssa.Phi)
		return p, ok
	}
	if c, ok := sizefacts.ConstIntV(bo.Y); ok && c >= 1 {
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
		if sizefacts.IsPositiveStep(phi, e) {
			hasStep = true
			continue
		}
		if _, ok := sh.f.LowerBoundConst(e, 0); !ok {
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
	if _, ok := sizefacts.ConstIntV(lowV); !ok {
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
		v, ok := sh.f.UpperExtent(e, 0)
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
			c, ok := sizefacts.ConstIntV(bo.Y)
			return ok && c > 0
		}
	case token.ADD:
		switch {
		case bo.X == phi:
			c, ok := sizefacts.ConstIntV(bo.Y)
			return ok && c < 0
		case bo.Y == phi:
			c, ok := sizefacts.ConstIntV(bo.X)
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
		lo, ok := sh.f.LowerBoundConst(e, 0)
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
	v, ok := sh.f.UpperExtent(boundV, 0)
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

// maxStepDepth bounds recursion through phi/arithmetic chains in mulStepRaw.
// Too shallow costs coverage, never correctness — rejection is the fallback.
const maxStepDepth = 8

// mulStepRaw returns the exact (k, d) of e = k*phi + d (sign of d checked by
// mulStep), descending through intermediate phis. depth guards against cyclic
// SSA; a header-phi self-reference (an identity, non-growing edge) is rejected.
func mulStepRaw(phi *ssa.Phi, e ssa.Value, depth int) (k, d int64, ok bool) {
	if depth > maxStepDepth {
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
		if c, isC := sizefacts.ConstIntV(bo.Y); isC {
			if kk, dd, ok := affineMul(phi, bo.X); ok {
				return kk, dd + c, true
			}
		}
		if c, isC := sizefacts.ConstIntV(bo.X); isC {
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
		c, ok := sizefacts.ConstIntV(bo.Y)
		return c, ok && c >= 2
	case bo.Y == phi:
		c, ok := sizefacts.ConstIntV(bo.X)
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
	c, ok := sizefacts.ConstIntV(lowV)
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
		v, ok := sh.f.UpperExtent(e, 0)
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
	k, ok := sizefacts.ConstIntV(bo.Y)
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
	d, ok := sizefacts.ConstIntV(sub.Y)
	return ok && d >= 0
}

// ruleRangeNext — R5, `range` over a map or string.
//
// Shape: the loop's exit If tests the ok-Extract of a Next in this header,
// whose Iter is Range(x), where x names a size (parameter or stable field
// path). Next yields each element at most once, so trips <= the element
// count. Strings are immutable — no further check. Maps may be mutated
// during range with unspecified visitation of new keys (Go spec), so the
// loop's blocks must contain no MapUpdate and no call-shaped or
// channel-synchronizing instruction; plain stores are fine (they cannot
// change a map's length — the reason this check is local, not fieldpath's).
func ruleRangeNext(sh *shape) (bound.Bound, bool) {
	ext, ok := sh.ifi.Cond.(*ssa.Extract)
	if !ok {
		return bound.Bound{}, false
	}
	next, ok := ext.Tuple.(*ssa.Next)
	if !ok || next.Block() != sh.loop.Header {
		return bound.Bound{}, false
	}
	rng, ok := next.Iter.(*ssa.Range)
	if !ok {
		return bound.Bound{}, false
	}
	if !next.IsString && mapRangeDirty(sh.loop) {
		return bound.Bound{}, false
	}
	x := rng.X
	if p, ok := x.(*ssa.Parameter); ok {
		return bound.Of(bound.Term(size.Len(p.Name()))), true
	}
	if path, ok := sh.f.Stab.PathFor(x); ok {
		return bound.Of(bound.Term(size.Len(path))), true
	}
	return bound.Bound{}, false
}

// mapRangeDirty reports whether the loop body could change the ranged map's
// size: any map write, or any instruction that hands control to code that
// could (calls, defers, goroutines, channel synchronization).
func mapRangeDirty(loop *loopnest.Loop) bool {
	for b := range loop.Blocks {
		for _, instr := range b.Instrs {
			switch v := instr.(type) {
			case *ssa.MapUpdate, *ssa.Defer, *ssa.Go, *ssa.Select, *ssa.Send:
				return true
			case *ssa.UnOp:
				if v.Op == token.ARROW {
					return true
				}
			case *ssa.Call:
				if bi, ok := v.Call.Value.(*ssa.Builtin); ok {
					switch bi.Name() {
					case "len", "cap":
						continue
					}
				}
				return true
			}
		}
	}
	return false
}

// ruleBisection — R6, the two-phi shrinking interval (binary search).
//
// Shape: guard `lo < hi` (strict; or `hi > lo`), both header phis; every
// in-loop edge pair updates EXACTLY one of them — lo' = mid + c (c >= 1) or
// hi' = mid - c (c >= 0, including hi' = mid) — where mid is (lo+hi)/2 or
// lo + (hi-lo)/2 computed in this loop; lowerBoundConst(lo0) >= 0;
// upperExtent(hi0) resolves.
//
// Claim: logarithmic, O(log(upper(hi0))). Argument: with lo >= 0 throughout
// (lo0 >= 0, lo only moves up to mid+c) and floor division, lo <= mid < hi
// whenever lo < hi — for (lo+hi)/2 under the documented no-overflow
// assumption (a length above 2^62), for lo+(hi-lo)/2 unconditionally. Both
// updates shrink hi-lo to <= ceil((hi-lo)/2), strictly, so the guard fails
// within log2(upper(hi0)) + 2 iterations. `lo' = mid` is rejected: when
// hi == lo+1, mid == lo and the loop need not terminate.
func ruleBisection(sh *shape) (bound.Bound, bool) {
	if sh.cmp == nil || (sh.cmp.Op != token.LSS && sh.cmp.Op != token.GTR) {
		return bound.Bound{}, false
	}
	loV, hiV := sh.cmp.X, sh.cmp.Y
	if sh.cmp.Op == token.GTR {
		loV, hiV = hiV, loV
	}
	lo, ok := loV.(*ssa.Phi)
	if !ok || lo.Block() != sh.loop.Header {
		return bound.Bound{}, false
	}
	hi, ok := hiV.(*ssa.Phi)
	if !ok || hi.Block() != sh.loop.Header {
		return bound.Bound{}, false
	}
	var extent bound.Var
	hasExtent, hasBack := false, false
	for i, pred := range lo.Block().Preds {
		le, he := lo.Edges[i], hi.Edges[i]
		if !sh.loop.Blocks[pred] { // init edge pair
			c, ok := sh.f.LowerBoundConst(le, 0)
			if !ok || c < 0 {
				return bound.Bound{}, false
			}
			v, ok := sh.f.UpperExtent(he, 0)
			if !ok {
				return bound.Bound{}, false
			}
			if hasExtent && v != extent {
				return bound.Bound{}, false
			}
			extent, hasExtent = v, true
			continue
		}
		hasBack = true
		switch { // exactly one end moves
		case he == hi && isLoUpdate(sh, le, lo, hi):
		case le == lo && isHiUpdate(sh, he, lo, hi):
		default:
			return bound.Bound{}, false
		}
	}
	if !hasExtent || !hasBack {
		return bound.Bound{}, false
	}
	return bound.Of(bound.Mono(extent, 0, 1)), true
}

// isLoUpdate matches lo' = mid + c for const c >= 1.
func isLoUpdate(sh *shape, v ssa.Value, lo, hi *ssa.Phi) bool {
	bo, ok := v.(*ssa.BinOp)
	if !ok || bo.Op != token.ADD {
		return false
	}
	if c, okC := sizefacts.ConstIntV(bo.Y); okC && c >= 1 {
		return isMid(sh, bo.X, lo, hi)
	}
	if c, okC := sizefacts.ConstIntV(bo.X); okC && c >= 1 {
		return isMid(sh, bo.Y, lo, hi)
	}
	return false
}

// isHiUpdate matches hi' = mid or hi' = mid - c for const c >= 0.
func isHiUpdate(sh *shape, v ssa.Value, lo, hi *ssa.Phi) bool {
	if isMid(sh, v, lo, hi) {
		return true
	}
	bo, ok := v.(*ssa.BinOp)
	if !ok || bo.Op != token.SUB {
		return false
	}
	c, okC := sizefacts.ConstIntV(bo.Y)
	return okC && c >= 0 && isMid(sh, bo.X, lo, hi)
}

// isMid matches (lo+hi)/2 and lo + (hi-lo)/2, computed inside this loop.
func isMid(sh *shape, v ssa.Value, lo, hi *ssa.Phi) bool {
	in, ok := v.(ssa.Instruction)
	if !ok || !sh.loop.Blocks[in.Block()] {
		return false
	}
	bo, ok := v.(*ssa.BinOp)
	if !ok {
		return false
	}
	// (lo+hi)/2
	if bo.Op == token.QUO {
		if c, okC := sizefacts.ConstIntV(bo.Y); okC && c == 2 {
			if add, okA := bo.X.(*ssa.BinOp); okA && add.Op == token.ADD {
				return (add.X == lo && add.Y == hi) || (add.X == hi && add.Y == lo)
			}
		}
		return false
	}
	// lo + (hi-lo)/2
	if bo.Op == token.ADD {
		half, x := bo.Y, bo.X
		if x != lo {
			half, x = bo.X, bo.Y
		}
		if x != lo {
			return false
		}
		q, ok := half.(*ssa.BinOp)
		if !ok || q.Op != token.QUO {
			return false
		}
		if c, okC := sizefacts.ConstIntV(q.Y); !okC || c != 2 {
			return false
		}
		sub, ok := q.X.(*ssa.BinOp)
		return ok && sub.Op == token.SUB && sub.X == hi && sub.Y == lo
	}
	return false
}
