// Package sizefacts provides the shared size-decrease primitives — constant
// resolution, canonical size variables, and provable lower-bound/upper-extent
// reasoning over SSA values — used by both tripcount and recurrence.
package sizefacts

import (
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/size"
)

// ConstIntV returns the exact int64 value of a constant.
func ConstIntV(v ssa.Value) (int64, bool) {
	c, ok := v.(*ssa.Const)
	if !ok || c.Value == nil {
		return 0, false
	}
	k, exact := constant.Int64Val(constant.ToInt(c.Value))
	return k, exact
}

// SizeVar maps a loop-bound value to a canonical size variable, or "".
func SizeVar(v ssa.Value) bound.Var {
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
		if IsInteger(t.Type()) {
			return size.Num(t.Name())
		}
	}
	return ""
}

// IsInteger reports whether t's underlying type is an integer basic type.
func IsInteger(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsInteger != 0
}

// IsPositiveStep reports whether e is phi + c for a constant c > 0.
func IsPositiveStep(phi *ssa.Phi, e ssa.Value) bool {
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
	k, exact := ConstIntV(v)
	return exact && k > 0
}

// maxFactsDepth bounds recursion through phi/arithmetic chains. Too shallow
// costs coverage, never correctness — rejection is the fallback.
const maxFactsDepth = 8

// Facts answers the two extent questions the evolution rules share. It holds
// no cross-loop state; a fresh instance is built per Of call.
type Facts struct {
	Stab *fieldpath.Stability
}

// LowerBoundConst returns a provable constant lower bound on v's value at
// every evaluation. ANY constant suffices: asymptotically a constant offset
// vanishes (the engine already accepts `for i := -5; i < n`).
func (f *Facts) LowerBoundConst(v ssa.Value, depth int) (int64, bool) {
	if depth > maxFactsDepth {
		return 0, false
	}
	switch t := v.(type) {
	case *ssa.Const:
		return ConstIntV(t)
	case *ssa.Phi:
		return f.phiLowerBound(t, depth)
	case *ssa.BinOp:
		if t.Op != token.ADD {
			return 0, false
		}
		if c, ok := ConstIntV(t.Y); ok {
			if lo, ok := f.LowerBoundConst(t.X, depth+1); ok {
				return lo + c, true
			}
			return 0, false
		}
		if c, ok := ConstIntV(t.X); ok {
			if lo, ok := f.LowerBoundConst(t.Y, depth+1); ok {
				return lo + c, true
			}
		}
	}
	return 0, false
}

// phiLowerBound: a phi whose every non-lower-bounded edge adds a NON-NEGATIVE
// constant to the phi never dips below its smallest bounded edge.
func (f *Facts) phiLowerBound(phi *ssa.Phi, depth int) (int64, bool) {
	low, hasInit := int64(0), false
	for _, e := range phi.Edges {
		if isNonNegStep(phi, e) {
			continue
		}
		lo, ok := f.LowerBoundConst(e, depth+1)
		if !ok {
			return 0, false
		}
		if !hasInit || lo < low {
			low = lo
		}
		hasInit = true
	}
	return low, hasInit
}

// isNonNegStep reports whether e is phi + c for a constant c >= 0.
func isNonNegStep(phi *ssa.Phi, e ssa.Value) bool {
	bo, ok := e.(*ssa.BinOp)
	if !ok || bo.Op != token.ADD {
		return false
	}
	switch {
	case bo.X == phi:
		c, ok := ConstIntV(bo.Y)
		return ok && c >= 0
	case bo.Y == phi:
		c, ok := ConstIntV(bo.X)
		return ok && c >= 0
	}
	return false
}

// UpperExtent resolves v to a size variable dominating max(v, 0) at every
// in-loop evaluation (trip counts are non-negative, so dominating the
// non-negative part suffices; this is what keeps e/c sound when e can be
// negative). Rules may only WEAKEN extents — never construct growing ones.
func (f *Facts) UpperExtent(v ssa.Value, depth int) (bound.Var, bool) {
	if depth > maxFactsDepth {
		return "", false
	}
	if s := SizeVar(v); s != "" {
		return s, true
	}
	if s, ok := f.Stab.VarFor(v); ok {
		return s, true
	}
	// len(x) for a locally-derived x — a make, a slice expression, or the
	// append(nil, y...) copy idiom. SizeVar above names len(param) only, so
	// without this every local slice is ⊤.
	if c, ok := v.(*ssa.Call); ok {
		if b, isBuiltin := c.Call.Value.(*ssa.Builtin); isBuiltin && b.Name() == "len" && len(c.Call.Args) == 1 {
			return f.lenExtent(c.Call.Args[0], depth+1)
		}
	}
	switch t := v.(type) {
	case *ssa.BinOp:
		switch t.Op {
		case token.SUB:
			// e - x <= e when x >= 0 (provably); e - negconst = e + |c|,
			// and constants vanish asymptotically.
			if lo, ok := f.LowerBoundConst(t.Y, depth+1); ok && lo >= 0 {
				return f.UpperExtent(t.X, depth+1)
			}
			if _, ok := ConstIntV(t.Y); ok { // negative const (>=0 handled above)
				return f.UpperExtent(t.X, depth+1)
			}
		case token.ADD:
			// e + c: constants vanish asymptotically (either sign).
			if _, ok := ConstIntV(t.Y); ok {
				return f.UpperExtent(t.X, depth+1)
			}
			if _, ok := ConstIntV(t.X); ok {
				return f.UpperExtent(t.Y, depth+1)
			}
		case token.QUO:
			// e/c <= max(e, 0) for const c >= 1 (floor toward zero).
			if c, ok := ConstIntV(t.Y); ok && c >= 1 {
				return f.UpperExtent(t.X, depth+1)
			}
		}
	case *ssa.Phi:
		return f.phiUpperExtent(t, depth)
	}
	return "", false
}

// phiUpperExtent covers two phi families:
//   - guard-bounded strict induction: value <= max(const inits, e + step)
//     everywhere = O(upper(e));
//   - non-increasing phi (every step edge adds c <= 0 or subtracts c >= 0):
//     value never exceeds its inits.
func (f *Facts) phiUpperExtent(phi *ssa.Phi, depth int) (bound.Var, bool) {
	if e, ok := guardBound(phi); ok {
		return f.UpperExtent(e, depth+1)
	}
	var up bound.Var
	hasInit := false
	for _, e := range phi.Edges {
		if isNonIncStep(phi, e) {
			continue
		}
		v, ok := f.UpperExtent(e, depth+1)
		if !ok {
			return "", false
		}
		if hasInit && v != up {
			return "", false // two different extents: keep v1 simple, reject
		}
		up, hasInit = v, true
	}
	return up, hasInit
}

// isNonIncStep reports whether e is phi + c (c <= 0) or phi - c (c >= 0).
// Division is deliberately NOT non-increasing: -5/2 = -2 > -5.
func isNonIncStep(phi *ssa.Phi, e ssa.Value) bool {
	bo, ok := e.(*ssa.BinOp)
	if !ok {
		return false
	}
	switch bo.Op {
	case token.ADD:
		switch {
		case bo.X == phi:
			c, ok := ConstIntV(bo.Y)
			return ok && c <= 0
		case bo.Y == phi:
			c, ok := ConstIntV(bo.X)
			return ok && c <= 0
		}
	case token.SUB:
		if bo.X == phi {
			c, ok := ConstIntV(bo.Y)
			return ok && c >= 0
		}
	}
	return false
}

// guardBound returns the bound expression e when phi is a STRICT induction —
// every edge a constant init or a positive-constant step — of a loop whose
// exit test upper-bounds the phi itself with the true branch staying inside.
// Strictness is load-bearing: with constant inits the phi's value is bounded
// by max(inits, e + step) EVERYWHERE, including after a zero-iteration exit;
// a parameter init would make the never-entered exit value unbounded.
func guardBound(phi *ssa.Phi) (ssa.Value, bool) {
	for _, e := range phi.Edges {
		if !IsPositiveStep(phi, e) && !isConstant(e) {
			return nil, false
		}
	}
	h := phi.Block()
	if len(h.Instrs) == 0 || len(h.Succs) != 2 {
		return nil, false
	}
	ifi, ok := h.Instrs[len(h.Instrs)-1].(*ssa.If)
	if !ok {
		return nil, false
	}
	cmp, ok := ifi.Cond.(*ssa.BinOp)
	if !ok {
		return nil, false
	}
	// The true branch must stay in the loop (be able to return to the header);
	// otherwise `phi < e` is an EXIT test and e bounds phi from BELOW, not above
	// — the loop continues while phi >= e and the induction is unbounded. This
	// mirrors the check tripcount.Of makes for the loop it analyzes; here it is
	// self-contained because guardBound is reached without a *loopnest.Loop.
	if len(h.Succs) != 2 || !reachesBlock(h.Succs[0], h) {
		return nil, false
	}
	switch cmp.Op {
	case token.LSS, token.LEQ:
		if cmp.X == phi {
			return cmp.Y, true
		}
	case token.GTR, token.GEQ:
		if cmp.Y == phi {
			return cmp.X, true
		}
	}
	return nil, false
}

// reachesBlock reports whether target is reachable from start by following
// successor edges (start counts as reaching itself: a single-block loop).
func reachesBlock(start, target *ssa.BasicBlock) bool {
	seen := map[*ssa.BasicBlock]bool{}
	stack := []*ssa.BasicBlock{start}
	for len(stack) > 0 {
		b := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if b == target {
			return true
		}
		if seen[b] {
			continue
		}
		seen[b] = true
		stack = append(stack, b.Succs...)
	}
	return false
}

// lenExtent resolves len(v) for a value whose length is locally derivable:
// a make, a slice expression, or the append(const-length, x...) copy idiom.
// Parameters are already named by SizeVar; this is everything else.
//
// Every rule is an UPPER bound on len(v), matching UpperExtent's contract.
// Over-approximating is safe (a Within becomes Unknown); under-approximating is
// a wrong bound, which is why make reads Len and never Cap.
func (f *Facts) lenExtent(v ssa.Value, depth int) (bound.Var, bool) {
	if depth > maxFactsDepth {
		return "", false
	}
	switch t := v.(type) {
	case *ssa.MakeSlice:
		// len(make([]T, n, c)) == n. NOT c: make([]T, 0, len(s)) has length 0,
		// and reading Cap would claim len(s) for an empty slice.
		return f.UpperExtent(t.Len, depth+1)
	case *ssa.Slice:
		// len(s[i:j]) == j-i <= j. Bounding by the operand's length instead
		// would be unsound: s[0:cap(s)] can exceed len(s).
		if t.High != nil {
			return f.UpperExtent(t.High, depth+1)
		}
		// s[i:] — the high bound IS the operand's length, so that is sound here.
		return f.lenOf(t.X, depth+1)
	case *ssa.Call:
		if b, ok := t.Call.Value.(*ssa.Builtin); ok && b.Name() == "append" && len(t.Call.Args) == 2 {
			// len(append(a, b...)) == len(a) + len(b). Only expressible as a
			// single extent when len(a) is constant; UpperExtent returns one
			// variable, so the general case is ⊤.
			if !constLen(t.Call.Args[0]) {
				return "", false
			}
			return f.lenOf(t.Call.Args[1], depth+1)
		}
	}
	return "", false
}

// lenOf resolves len(v) whether v is a length-classed parameter, a field path,
// or a locally-derived value.
func (f *Facts) lenOf(v ssa.Value, depth int) (bound.Var, bool) {
	// A parameter, but only one whose size IS a length: size.Len on an integer
	// parameter would fabricate "len(n)".
	if av, cls, ok := size.ValueClass(v); ok && cls == size.Length {
		return av, true
	}
	if s, ok := f.Stab.VarFor(v); ok {
		return s, true
	}
	return f.lenExtent(v, depth)
}

// constLen reports whether len(v) is provably a compile-time constant — the
// gate for the append copy idiom. A nil slice and make([]T, 0) qualify;
// everything else is not proven and therefore does not.
func constLen(v ssa.Value) bool {
	switch t := v.(type) {
	case *ssa.Const:
		return t.Value == nil // nil slice literal: len 0
	case *ssa.MakeSlice:
		c, ok := ConstIntV(t.Len)
		return ok && c == 0
	}
	return false
}
