// Package tripcount infers the iteration-count of a natural loop.
package tripcount

import (
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/size"
)

// shape is one loop's exit test, extracted once per Of call and handed to
// every evolution rule.
type shape struct {
	loop *loopnest.Loop
	ifi  *ssa.If
	cmp  *ssa.BinOp // nil when the exit test is not a comparison
	f    *facts
}

// Of returns the loop's iteration-count bound in canonical size variables or
// Top() when no evolution rule recognizes the shape. Rules are tried
// most-specific-first; order never affects soundness (each rule is
// independently sound), only which equivalent answer is produced.
func Of(loop *loopnest.Loop, stab *fieldpath.Stability) bound.Bound {
	h := loop.Header
	if len(h.Instrs) == 0 {
		return bound.Top()
	}
	ifi, ok := h.Instrs[len(h.Instrs)-1].(*ssa.If)
	if !ok {
		return bound.Top()
	}
	// The true branch must stay in the loop; otherwise the condition is an
	// exit test and the bound side would be misread.
	if len(ifi.Block().Succs) != 2 || !loop.Blocks[ifi.Block().Succs[0]] {
		return bound.Top()
	}
	sh := &shape{loop: loop, ifi: ifi, f: &facts{stab: stab}}
	sh.cmp, _ = ifi.Cond.(*ssa.BinOp)

	for _, rule := range rules {
		if b, ok := rule(sh); ok {
			return b
		}
	}
	return bound.Top()
}

// rules in most-specific-first order.
var rules = []func(*shape) (bound.Bound, bool){
	ruleDecreasing,
	ruleIncreasing,
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
	k, exact := constIntV(v)
	return exact && k > 0
}

// constIntV returns the exact int64 value of a constant.
func constIntV(v ssa.Value) (int64, bool) {
	c, ok := v.(*ssa.Const)
	if !ok || c.Value == nil {
		return 0, false
	}
	k, exact := constant.Int64Val(constant.ToInt(c.Value))
	return k, exact
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
