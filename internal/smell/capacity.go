package smell

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/sizefacts"
)

func init() {
	register("SM3", smAppendNoPrealloc)
	register("SM6", smMapNoSizeHint)
}

// smAppendNoPrealloc fires when a slice-typed phi in a resolvable data-dependent
// loop starts at zero capacity (var s []T or make([]T, 0)) and is grown by
// append inside the loop. The fix names the bound: make(…, 0, <tc>).
func smAppendNoPrealloc(_ *ssa.Function, ctx *fnContext) []Finding {
	var out []Finding
	for lp, tc := range ctx.resolvable {
		for _, instr := range lp.Header.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok || phi.Block() != lp.Header {
				continue
			}
			if !isSlice(phi.Type()) {
				continue
			}
			// Entry edge (index 0) must be a proven zero-capacity origin.
			if !zeroCapOrigin(phi.Edges[0]) {
				continue
			}
			// A back-edge must be an append call with the phi as the first arg.
			pos, ok := appendsSelf(phi, lp)
			if !ok {
				continue
			}
			out = append(out, Finding{
				Pos:     pos,
				Rule:    "SM3",
				Message: "append in a loop bounded by " + tc.String() + " on a zero-capacity slice; preallocate with make(…, 0, " + tc.String() + ")",
			})
			break
		}
	}
	return out
}

// smMapNoSizeHint fires when a make(map[K]V) without a size hint (Reserve is
// nil/const-0) has a MapUpdate on its exact SSA value inside a resolvable
// data-dependent loop.
func smMapNoSizeHint(fn *ssa.Function, ctx *fnContext) []Finding {
	var out []Finding
	// Collect all MakeMap without a reserve, keyed by their SSA value.
	type mk struct {
		pos token.Pos
		v   ssa.Value
	}
	var maps []mk
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			mm, ok := instr.(*ssa.MakeMap)
			if !ok {
				continue
			}
			if hasReserve(mm) {
				continue
			}
			maps = append(maps, mk{pos: mm.Pos(), v: mm})
		}
	}
	for _, m := range maps {
		tc, ok := mapUpdateResolvableLoop(m.v, ctx)
		if !ok {
			continue
		}
		out = append(out, Finding{
			Pos:     m.pos,
			Rule:    "SM6",
			Message: "map grown in a loop bounded by " + tc.String() + " without a size hint; preallocate with make(map[K]V, " + tc.String() + ")",
		})
	}
	return out
}

// zeroCapOrigin reports whether v is a slice origin provable as zero-capacity.
//
// Three spellings, and the third is the one learner code actually uses:
//
//   - a const nil — `var s []T`;
//   - a MakeSlice with a const-0 Cap — `make([]T, 0, 0)` when ssa keeps it;
//   - a SLICE of a zero-length array — which is what `make([]T, 0)` compiles to.
//
// The third arm was missing until 2026-08-29 although this comment had always
// claimed `make([]T, 0)` was covered. go/ssa lowers a constant-zero-length make
// to an allocation of a zero-length ARRAY and a slice of it, never to a
// MakeSlice:
//
//	t0 = new [0]int (makeslice)     *[0]int
//	t1 = slice t0[:0]               []int
//
// so the MakeSlice arm never saw it. Measured on 144 real ya_algo learner files,
// `make([]T, 0)` is the dominant spelling and SM3's yield there was ZERO.
//
// SOUNDNESS. cap(x[lo:hi:max]) is max-lo when max is given, and cap(x)-lo
// otherwise. Both arms below prove cap == 0 without needing lo:
//
//   - an explicit const-0 Max gives cap = 0 - lo <= 0, since lo >= 0 always;
//   - a pointer-to-[0]T operand has cap(x) == 0 BY ITS TYPE, so cap = -lo <= 0.
//
// A slice of a NON-empty array is therefore correctly refused: `arr[:0]` on a
// [16]int has capacity 16, and calling that a zero-capacity origin would turn
// correctly-preallocated code into a finding. Pinned in TestSM3IgnoresPreallocated.
func zeroCapOrigin(v ssa.Value) bool {
	switch v := v.(type) {
	case *ssa.Const:
		return v.Value == nil // nil literal
	case *ssa.MakeSlice:
		return constZero(v.Cap)
	case *ssa.Slice:
		if v.Max != nil {
			return constZero(v.Max)
		}
		return zeroLenArrayPtr(v.X.Type())
	}
	return false
}

// zeroLenArrayPtr reports whether t is a pointer to a zero-length array, the
// operand shape `make([]T, 0)` produces. Anything else — a pointer to a sized
// array, a slice, a string — has a capacity this cannot prove is zero.
func zeroLenArrayPtr(t types.Type) bool {
	p, ok := t.Underlying().(*types.Pointer)
	if !ok {
		return false
	}
	arr, ok := p.Elem().Underlying().(*types.Array)
	return ok && arr.Len() == 0
}

// appendsSelf reports whether any back-edge of phi is an append(phi, ...) call,
// returning that call's position. The call site — not phi.Pos(), which for a
// named local is the variable's declaration — is where the finding anchors, so
// that two accumulators declared on one line stay distinct (issue #73).
// A back edge is followed THROUGH intermediate phis, because an append guarded
// by an `if` does not reach the header directly:
//
//	for i := 2; i < n; i++ {
//	    if pred(i) { out = append(out, i) }   // the latch merges append and not-append
//	}
//
// There the header phi's back edge is the if-join phi, and the append is one of
// THAT phi's edges. Before 2026-08-29 only the direct form was recognised, which
// is why SM3 was silent on the ya_algo learner corpus: the guarded spelling is
// the common one.
//
// Following the merge does not over-claim. The finding says the slice is grown
// by append inside the loop and should be preallocated; that is true when the
// append happens on SOME path, which is what a merge edge witnesses. It is
// advice, not a bound.
func appendsSelf(phi *ssa.Phi, lp *loopnest.Loop) (token.Pos, bool) {
	for i, edge := range phi.Edges {
		if i == 0 {
			continue
		}
		if !lp.Blocks[phi.Block().Preds[i]] {
			continue
		}
		if pos, ok := appendOf(edge, phi, lp, map[ssa.Value]bool{}); ok {
			return pos, true
		}
	}
	return token.NoPos, false
}

// appendOf reports whether v is append(target, …), or a phi inside the loop
// that merges such a call on one of its edges. The seen set terminates on the
// phi cycles a nested loop produces.
func appendOf(v ssa.Value, target *ssa.Phi, lp *loopnest.Loop, seen map[ssa.Value]bool) (token.Pos, bool) {
	if v == nil || seen[v] {
		return token.NoPos, false
	}
	seen[v] = true
	switch t := v.(type) {
	case *ssa.Call:
		bi, ok := t.Call.Value.(*ssa.Builtin)
		if !ok || bi.Name() != "append" {
			return token.NoPos, false
		}
		if len(t.Call.Args) > 0 && t.Call.Args[0] == target {
			return t.Pos(), true
		}
	case *ssa.Phi:
		// Only merges INSIDE the loop: a phi outside it is a different value's
		// history, not this iteration's growth.
		if t.Block() == nil || !lp.Blocks[t.Block()] {
			return token.NoPos, false
		}
		for _, e := range t.Edges {
			if pos, ok := appendOf(e, target, lp, seen); ok {
				return pos, true
			}
		}
	}
	return token.NoPos, false
}

// hasReserve reports whether a MakeMap has a non-zero reserve (size hint).
func hasReserve(mm *ssa.MakeMap) bool {
	if mm.Reserve == nil {
		return false
	}
	// A const-0 reserve is still "no hint" in effect.
	if c, ok := sizefacts.ConstIntV(mm.Reserve); ok && c == 0 {
		return false
	}
	return true
}

// mapUpdateResolvableLoop returns the trip-count bound of a resolvable
// data-dependent loop that contains a MapUpdate on mapVal.
func mapUpdateResolvableLoop(mapVal ssa.Value, ctx *fnContext) (bound.Bound, bool) {
	refs := mapVal.Referrers()
	if refs == nil {
		return bound.Top(), false
	}
	for _, ref := range *refs {
		upd, ok := ref.(*ssa.MapUpdate)
		if !ok {
			continue
		}
		for _, lp := range ctx.forest.EnclosingLoops(upd.Block()) {
			if tc, ok := ctx.resolvable[lp]; ok {
				return tc, true
			}
		}
	}
	return bound.Top(), false
}

func constZero(v ssa.Value) bool {
	c, ok := sizefacts.ConstIntV(v)
	return ok && c == 0
}

// isSlice reports whether t's underlying type is a slice.
func isSlice(t types.Type) bool {
	_, ok := t.Underlying().(*types.Slice)
	return ok
}
