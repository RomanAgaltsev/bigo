package recognize

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/loopnest"
)

// linkedTraversal matches `for p != nil { … p = p.<field> }` where <field> has
// p's own type: a walk along a single self-referential link.
//
// The bound names the number of nodes reachable from the entry pointer, which
// the bound algebra cannot express — no parameter has that length. Reporting it
// is only sound because a recognition carries no verdict power, and it is the
// first thing bigo can say about this class at all.
func linkedTraversal(fn *ssa.Function) []Recognition {
	forest := loopnest.Build(fn)
	var out []Recognition
	for _, root := range forest.Roots {
		if r, ok := matchLinked(root); ok {
			out = append(out, r)
		}
	}
	return out
}

func matchLinked(loop *loopnest.Loop) (Recognition, bool) {
	if len(loop.Header.Instrs) == 0 {
		return Recognition{}, false
	}
	ifi, ok := loop.Header.Instrs[len(loop.Header.Instrs)-1].(*ssa.If)
	if !ok {
		return Recognition{}, false
	}
	cmp, ok := ifi.Cond.(*ssa.BinOp)
	if !ok || cmp.Op != token.NEQ {
		return Recognition{}, false
	}
	phi, ok := cmp.X.(*ssa.Phi)
	if !ok {
		return Recognition{}, false
	}
	if c, isConst := cmp.Y.(*ssa.Const); !isConst || !c.IsNil() {
		return Recognition{}, false
	}
	// Every in-loop edge must load the SAME field, of the phi's own type,
	// from the phi itself. An edge that is anything else — a second link, a
	// conditional advance's merge, a value from elsewhere — means "visits each
	// node at most once" is not established, so the whole shape is refused.
	var field *types.Var
	for i, e := range phi.Edges {
		preds := phi.Block().Preds
		if i >= len(preds) {
			return Recognition{}, false
		}
		if !loop.Blocks[preds[i]] {
			continue // the entry value
		}
		f, ok := selfLinkField(e, phi)
		if !ok {
			return Recognition{}, false
		}
		if field != nil && field != f {
			return Recognition{}, false // two different links: refuse
		}
		field = f
	}
	if field == nil {
		return Recognition{}, false
	}
	return Recognition{
		// The guard's own position: control-flow instructions carry
		// token.NoPos, and a recognition with no position cannot be reported
		// against the loop it describes.
		Pos:     cmp.Pos(),
		Pattern: "linked-structure traversal",
		Kind:    KindWorst,
		Bound:   "O(n)",
		Assumption: "the traversal follows the single self-referential link ." +
			field.Name() + " and visits each node at most once; n is the number " +
			"of nodes reachable from the entry pointer",
	}, true
}

// selfLinkField reports the struct field v loads from phi, when v is exactly
// `*(&phi.field)` and the field has phi's own type.
func selfLinkField(v ssa.Value, phi *ssa.Phi) (*types.Var, bool) {
	load, ok := v.(*ssa.UnOp)
	if !ok || load.Op != token.MUL {
		return nil, false
	}
	fa, ok := load.X.(*ssa.FieldAddr)
	if !ok || fa.X != ssa.Value(phi) {
		return nil, false
	}
	st, ok := fa.X.Type().Underlying().(*types.Pointer)
	if !ok {
		return nil, false
	}
	strct, ok := st.Elem().Underlying().(*types.Struct)
	if !ok {
		return nil, false
	}
	if fa.Field < 0 || fa.Field >= strct.NumFields() {
		return nil, false
	}
	f := strct.Field(fa.Field)
	if !types.Identical(f.Type(), phi.Type()) {
		return nil, false
	}
	return f, true
}
