package smell

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/loopnest"
)

// loopInvariant reports whether v provably holds the same value on every
// iteration of lp.
//
// For an IMMUTABLE type this is decidable from the definition site alone. SSA
// is single-assignment, so a value whose defining instruction lies outside the
// loop body cannot be recomputed inside it, and Go strings are immutable, so no
// aliasing route can change it either. Values with no defining instruction —
// *ssa.Const, *ssa.Parameter, *ssa.FreeVar, *ssa.Global — are invariant by
// construction.
//
// It is NOT sufficient for a mutable operand: a slice defined outside the loop
// can still have its contents rewritten inside it. Callers handling mutable
// types must pair this with unmutatedIn.
func loopInvariant(v ssa.Value, lp *loopnest.Loop) bool {
	if instr, ok := v.(ssa.Instruction); ok {
		// Loop phis are covered here rather than special-cased: a header phi's
		// block is the header, which is always a member of lp.Blocks.
		return !lp.Blocks[instr.Block()]
	}
	return true
}

// sortedSliceOperand returns the slice a sort call operates on, unwrapping the
// interface boxing that sort.Slice/sort.SliceStable require.
//
// It reports false when no slice-typed value is reachable. That is not an
// oversight for sort.Sort/sort.Stable: their sort.Interface argument keeps the
// sortable state behind user-defined Len/Less/Swap, so there is nothing whose
// stability this package could check, and those two names decline permanently.
func sortedSliceOperand(c *ssa.CallCommon) (ssa.Value, bool) {
	if len(c.Args) == 0 {
		return nil, false
	}
	v := c.Args[0]
	for {
		switch u := v.(type) {
		case *ssa.MakeInterface:
			v = u.X
		case *ssa.ChangeType:
			v = u.X
		case *ssa.Convert:
			v = u.X
		default:
			if isSlice(v.Type()) {
				return v, true
			}
			return nil, false
		}
	}
}

// unmutatedIn reports whether nothing inside lp can change v's contents.
//
// Conservative by construction: a referrer inside the loop is accepted only
// when it provably cannot write. Everything else — an append, a store, a Send,
// a pass to any function outside sm5Names — is treated as a possible write,
// because if the contents change then the next sort is necessary work rather
// than a redundant one, and the rule's advice would be wrong.
//
// The walk carries a seen set rather than a depth cap: retyping chains are
// short but may share values, and a cap would truncate exactly the cases that
// need the answer.
func unmutatedIn(v ssa.Value, lp *loopnest.Loop) bool {
	return unmutatedRefs(v, lp, map[ssa.Value]bool{})
}

func unmutatedRefs(v ssa.Value, lp *loopnest.Loop, seen map[ssa.Value]bool) bool {
	if seen[v] {
		return true
	}
	seen[v] = true
	refs := v.Referrers()
	if refs == nil {
		return true
	}
	for _, ref := range *refs {
		if !lp.Blocks[ref.Block()] {
			continue
		}
		switch r := ref.(type) {
		case *ssa.MakeInterface:
			if !unmutatedRefs(r, lp, seen) {
				return false
			}
		case *ssa.ChangeType:
			if !unmutatedRefs(r, lp, seen) {
				return false
			}
		case *ssa.Convert:
			if !unmutatedRefs(r, lp, seen) {
				return false
			}
		case *ssa.IndexAddr:
			// An element address writes only if something stores through it.
			if !onlyLoaded(r) {
				return false
			}
		case *ssa.Call:
			if !nonMutatingCall(r) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// onlyLoaded reports whether every use of an element address is a load.
func onlyLoaded(v ssa.Value) bool {
	refs := v.Referrers()
	if refs == nil {
		return true
	}
	for _, ref := range *refs {
		un, ok := ref.(*ssa.UnOp)
		if !ok || un.Op != token.MUL {
			return false
		}
	}
	return true
}

// nonMutatingCall reports whether c provably cannot write through its slice
// argument: a sort from sm5Names (which permutes but does not change the
// multiset, so a following sort is still redundant), or the len/cap builtins.
func nonMutatingCall(c *ssa.Call) bool {
	if bi, ok := c.Call.Value.(*ssa.Builtin); ok {
		return bi.Name() == "len" || bi.Name() == "cap"
	}
	origin, resolved := calleeOrigin(&c.Call)
	return resolved && sm5Names[origin]
}
