// Package tripcount infers the iteration-count of a natural loop.
package tripcount

import (
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/size"
)

// Of returns the loop's iteration-count bound in canonical size variables or
// Top() when the loop shape is not recognized. Acceptance is deliberately
// strict — every relaxation here must be argued for soundness:
//   - the header's *ssa.If true-branch must stay inside the loop,
//   - the comparison must test a strictly increasing induction variable
//     against an upper bound (ind < b, ind <= b, b > ind, b >= ind),
//   - the induction phi must start at a constant and advance by a positive
//     constant step on every other edge.
func Of(loop *loopnest.Loop) bound.Bound {
	h := loop.Header
	if len(h.Instrs) == 0 {
		return bound.Top()
	}
	ifi, ok := h.Instrs[len(h.Instrs)-1].(*ssa.If)
	if !ok {
		return bound.Top()
	}
	// The true branch must be the edge that stays in the loop; otherwise the
	// condition is an exit test and "bound" would be misread.
	if len(ifi.Block().Succs) != 2 || !loop.Blocks[ifi.Block().Succs[0]] {
		return bound.Top()
	}
	cmp, ok := ifi.Cond.(*ssa.BinOp)
	if !ok {
		return bound.Top()
	}
	boundVal, ok := classify(loop, cmp)
	if !ok {
		return bound.Top()
	}
	if v := sizeVar(boundVal); v != "" {
		return bound.Of(bound.Term(v))
	}
	return bound.Top()
}

// classify returns the bound operand of the loop condition when the condition
// tests an increasing induction variable against an upper bound. Direction is
// load-bearing: `i >= n; i++` never terminates for i0 >= n, so only
// upper-bound comparisons are trip counts.
func classify(loop *loopnest.Loop, cmp *ssa.BinOp) (ssa.Value, bool) {
	switch cmp.Op {
	case token.LSS, token.LEQ: // induction < bound
		if isIncreasingInduction(loop, cmp.X) {
			return cmp.Y, true
		}
	case token.GTR, token.GEQ: // bound > induction
		if isIncreasingInduction(loop, cmp.Y) {
			return cmp.X, true
		}
	}
	return nil, false
}

// isIncreasingInduction reports whether v advances by a positive constant once
// per iteration: either the header phi itself (`for i := 0; i < N; i++`) or
// that phi plus a constant (the `for range` shape compares phi+1).
//
// The offset must be constant, and that is soundness rather than precision:
// `phi + j` for a variable j makes the condition `phi < bound - j`, so the
// trip count is bound-offset, not O(bound) — for j < 0 the loop runs more
// than `bound` times. It is the same defect as a non-constant start value,
// entering through the comparison operand instead of the phi's init edge.
func isIncreasingInduction(loop *loopnest.Loop, v ssa.Value) bool {
	if isInductionPhi(loop, v) {
		return true
	}
	bo, ok := v.(*ssa.BinOp)
	if !ok || bo.Op != token.ADD {
		return false
	}
	return (isInductionPhi(loop, bo.X) && isConstant(bo.Y)) ||
		(isInductionPhi(loop, bo.Y) && isConstant(bo.X))
}

// isInductionPhi reports whether v is a header phi that starts at a constant
// and advances by a positive constant step. Every edge must be one or the
// other; any third kind of edge (e.g. a reset to a parameter) disqualifies.
// Both constraints are soundness, not style: a zero or negative step means
// the loop need not terminate, and a non-constant start `i := m` makes the
// trip count bound-start, which is not O(bound).
func isInductionPhi(loop *loopnest.Loop, v ssa.Value) bool {
	phi, ok := v.(*ssa.Phi)
	if !ok || phi.Block() != loop.Header {
		return false
	}
	hasStep, hasInit := false, false
	for _, e := range phi.Edges {
		switch {
		case isPositiveStep(phi, e):
			hasStep = true
		case isConstant(e):
			hasInit = true
		default:
			return false
		}
	}
	return hasStep && hasInit
}

// isPositiveStep reports whether e is phi + c for a constant c > 0.
func isPositiveStep(phi *ssa.Phi, e ssa.Value) bool {
	bo, ok := e.(*ssa.BinOp)
	if !ok || bo.Op != token.ADD {
		return false
	}
	switch {
	case bo.X == phi:
		return isPositiveConst(bo.Y)
	case bo.Y == phi:
		return isPositiveConst(bo.X)
	}
	return false
}

func isConstant(v ssa.Value) bool {
	_, ok := v.(*ssa.Const)
	return ok
}

func isPositiveConst(v ssa.Value) bool {
	c, ok := v.(*ssa.Const)
	if !ok || c.Value == nil {
		return false
	}
	k, exact := constant.Int64Val(constant.ToInt(c.Value))
	return exact && k > 0
}

// sizeVar maps a loop-bound value to a canonical size variable, or "".
func sizeVar(v ssa.Value) bound.Var {
	switch t := v.(type) {
	case *ssa.Call:
		if b, ok := t.Call.Value.(*ssa.Builtin); ok && len(t.Call.Args) == 1 {
			if p, ok := t.Call.Args[0].(*ssa.Parameter); ok {
				switch b.Name() {
				case "len":
					return size.Len(p.Name())
				case "cap":
					return size.Cap(p.Name())
				}
			}
		}
	case *ssa.Parameter:
		if isInteger(t.Type()) {
			return size.Num(t.Name())
		}
	}
	return ""
}

func isInteger(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsInteger != 0
}
