package smell

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

func init() { register("SM1", smConcatInLoop) }

// smConcatInLoop fires when a string-typed phi in a data-dependent loop header
// has a back-edge value that combines the phi with new data each iteration —
// through a string + (BinOp ADD) with the phi as an operand, or a fmt.Sprintf
// whose varargs include the phi. That is the SSA shape of `s += x` /
// `s = s + x` / `s = fmt.Sprintf("%s", s, ...)` in a loop, the quadratic
// string-building pattern strings.Builder replaces.
//
// Crucially, a value that merely passes the phi through UNCHANGED (a continue
// path that leaves the accumulator alone) is NOT accumulation — the detector
// must prove the phi is combined with fresh data each iteration, not just that
// the phi is loop-carried. This keeps SM1 off plain loop-carried string
// variables that happen to survive a continue (paramSpill, capturedSpillRoot).
func smConcatInLoop(fn *ssa.Function, ctx *fnContext) []Finding {
	var out []Finding
	for _, lp := range dataDependentLoops(fn, ctx) {
		for _, instr := range lp.Header.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok || !isString(phi.Type()) {
				continue
			}
			if phi.Block() != lp.Header {
				continue
			}
			// Check each back-edge (from inside the loop) for a self-referential
			// string accumulation.
			for i, edge := range phi.Edges {
				if i == 0 {
					continue // entry edge
				}
				if lp.Blocks[phi.Block().Preds[i]] && stringAccumulates(edge, phi, map[ssa.Value]bool{}) {
					out = append(out, Finding{
						Pos:     phi.Pos(),
						Rule:    "SM1",
						Message: "string built by repeated concatenation in a loop (quadratic); use strings.Builder",
					})
					break
				}
			}
		}
	}
	return out
}

// stringAccumulates reports whether v represents the accumulator phi being
// combined with new data: a string + (BinOp ADD) with the phi as an operand, or
// a fmt.Sprintf with the phi among its varargs. Control-flow merge phis (a
// continue path leaving the phi unchanged) are walked, but a bare pass-through
// of the phi is NOT accumulation.
func stringAccumulates(v ssa.Value, target *ssa.Phi, visited map[ssa.Value]bool) bool {
	if visited[v] {
		return false
	}
	visited[v] = true
	switch v := v.(type) {
	case *ssa.BinOp:
		if v.Op != token.ADD || !isString(v.Type()) {
			return false
		}
		// The accumulation: is target a direct operand, or nested in a + chain?
		return addOperandHas(v.X, target) || addOperandHas(v.Y, target)
	case *ssa.Call:
		// fmt.Sprintf("%s", s, ...) builds a new string from its varargs; the
		// phi is boxed in a MakeInterface and stored in the varargs aggregate.
		if name, ok := calleeOrigin(&v.Call); ok && name == "fmt.Sprintf" {
			for _, arg := range v.Call.Args {
				if sprintfArgHas(arg, target) {
					return true
				}
			}
		}
		return false
	case *ssa.Phi:
		if v == target {
			return false // unchanged pass-through, not accumulation
		}
		// Control-flow merge (e.g. a continue path): chase both edges.
		for _, e := range v.Edges {
			if stringAccumulates(e, target, visited) {
				return true
			}
		}
		return false
	}
	return false
}

// addOperandHas reports whether v's subtree contains target, chasing only
// through nested string + (the chain a + b + c). Boxing and control-flow merges
// are not part of a plain concat operand.
func addOperandHas(v ssa.Value, target *ssa.Phi) bool {
	if v == target {
		return true
	}
	if bo, ok := v.(*ssa.BinOp); ok && bo.Op == token.ADD && isString(bo.Type()) {
		return addOperandHas(bo.X, target) || addOperandHas(bo.Y, target)
	}
	return false
}

// sprintfArgHas reports whether target appears in a Sprintf varargs argument,
// chasing through the boxing (Slice over the varargs Alloc, the Alloc's stores
// — including those through &alloc[i] — and MakeInterface).
func sprintfArgHas(v ssa.Value, target *ssa.Phi) bool {
	if v == target {
		return true
	}
	switch v := v.(type) {
	case *ssa.MakeInterface:
		return sprintfArgHas(v.X, target)
	case *ssa.Slice:
		return sprintfArgHas(v.X, target)
	case *ssa.Alloc:
		for _, ref := range *v.Referrers() {
			switch r := ref.(type) {
			case *ssa.Store:
				if sprintfArgHas(r.Val, target) {
					return true
				}
			case *ssa.IndexAddr:
				for _, s := range *r.Referrers() {
					if st, ok := s.(*ssa.Store); ok && sprintfArgHas(st.Val, target) {
						return true
					}
				}
			}
		}
	}
	return false
}
