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
			if !appendsSelf(phi, lp) {
				continue
			}
			out = append(out, Finding{
				Pos:     phi.Pos(),
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
			Message: "map built without a size hint in a loop bounded by " + tc.String() + "; preallocate with make(map[K]V, " + tc.String() + ")",
		})
	}
	return out
}

// zeroCapOrigin reports whether v is a slice origin provable as zero-capacity:
// a const nil (var s []T), or a MakeSlice with a const-0 Cap.
func zeroCapOrigin(v ssa.Value) bool {
	switch v := v.(type) {
	case *ssa.Const:
		return v.Value == nil // nil literal
	case *ssa.MakeSlice:
		return constZero(v.Cap)
	}
	return false
}

// appendsSelf reports whether any back-edge of phi is an append(phi, ...) call.
func appendsSelf(phi *ssa.Phi, lp *loopnest.Loop) bool {
	for i, edge := range phi.Edges {
		if i == 0 {
			continue
		}
		if !lp.Blocks[phi.Block().Preds[i]] {
			continue
		}
		call, ok := edge.(*ssa.Call)
		if !ok {
			continue
		}
		bi, ok := call.Call.Value.(*ssa.Builtin)
		if !ok || bi.Name() != "append" {
			continue
		}
		if len(call.Call.Args) > 0 && call.Call.Args[0] == phi {
			return true
		}
	}
	return false
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
