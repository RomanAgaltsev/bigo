// Package tripcount infers the iteration-count of a natural loop.
package tripcount

import (
	"go/token"
	"go/types"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/size"
	"golang.org/x/tools/go/ssa"
)

// Of returns the loop's iteration-count bound in canonical size variables or
// Top() when the loop shape is not recognized.
func Of(loop *loopnest.Loop) bound.Bound {
	h := loop.Header
	if len(h.Instrs) == 0 {
		return bound.Top()
	}
	ifi, ok := h.Instrs[len(h.Instrs)-1].(*ssa.If)
	if !ok {
		return bound.Top()
	}
	cmp, ok := ifi.Cond.(*ssa.BinOp)
	if !ok || !isComparison(cmp.Op) {
		return bound.Top()
	}
	_, boundVal := classify(loop, cmp)
	if boundVal == nil {
		return bound.Top()
	}
	if v := sizeVar(boundVal); v != "" {
		return bound.Of(bound.Term(v))
	}
	return bound.Top()
}

func isComparison(op token.Token) bool {
	switch op {
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}

// classify returns (induction, bound) operands of the loop condition or (nil,nil).
func classify(loop *loopnest.Loop, cmp *ssa.BinOp) (ssa.Value, ssa.Value) {
	if isInduction(loop, cmp.X) {
		return cmp.X, cmp.Y
	}
	if isInduction(loop, cmp.Y) {
		return cmp.Y, cmp.X
	}
	return nil, nil
}

// isInduction reports whether v adnvances once per iteration: either the header
// phi itself (the `for i := 0; i < N; i++` shape) or that phi's in-header
// increment (the `for range` shape, whose condition compares phi+1, not phi).
func isInduction(loop *loopnest.Loop, v ssa.Value) bool {
	if isInductionPhi(loop, v) {
		return true
	}
	bo, ok := v.(*ssa.BinOp)
	if !ok || bo.Op != token.ADD {
		return false
	}
	return isInductionPhi(loop, bo.X) || isInductionPhi(loop, bo.Y)
}

// isInductionPhi reports whether v is a header phi advanced by a constant step
// along one of its edges. The step must be an *ssa.Const: a runtime step k may
// be <= 0, in which case the loop need not terminate and no trip count exists.
func isInductionPhi(loop *loopnest.Loop, v ssa.Value) bool {
	phi, ok := v.(*ssa.Phi)
	if !ok || phi.Block() != loop.Header {
		return false
	}
	for _, e := range phi.Edges {
		bo, ok := e.(*ssa.BinOp)
		if !ok || bo.Op != token.ADD {
			continue
		}
		switch {
		case bo.X == phi:
			if _, ok := bo.Y.(*ssa.Const); ok {
				return true
			}
		case bo.Y == phi:
			if _, ok := bo.X.(*ssa.Const); ok {
				return true
			}
		}
	}
	return false
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
