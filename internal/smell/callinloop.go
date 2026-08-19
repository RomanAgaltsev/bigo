package smell

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/loopnest"
)

// SM4 fires on regexp compilation inside ANY natural loop.
var sm4Names = map[string]bool{
	"regexp.Compile":     true,
	"regexp.MustCompile": true,
}

// SM5 fires on sorting inside a DATA-DEPENDENT natural loop.
var sm5Names = map[string]bool{
	"sort.Slice":            true,
	"sort.SliceStable":      true,
	"sort.Sort":             true,
	"sort.Stable":           true,
	"sort.Strings":          true,
	"sort.Ints":             true,
	"sort.Float64s":         true,
	"slices.Sort":           true,
	"slices.SortFunc":       true,
	"slices.SortStable":     true,
	"slices.SortStableFunc": true,
}

// SM2 names: the linear-scan helpers over a slice. Contains/Index are the
// common redundant-scan shape; the *Func variants scan with a predicate.
var sm2Names = map[string]bool{
	"slices.Contains":     true,
	"slices.Index":        true,
	"slices.ContainsFunc": true,
	"slices.IndexFunc":    true,
}

func init() {
	register("SM4", smCompileInLoop)
	register("SM5", smSortInLoop)
	register("SM2", smLinearScan)
}

// smCompileInLoop fires when regexp.Compile/MustCompile is called inside any
// natural loop FROM A LOOP-INVARIANT PATTERN — recompiling the same pattern
// even a constant number of times is the bug, and the pattern being invariant
// is what makes "hoist" true advice. A pattern built per iteration has nothing
// to hoist, so the rule stays silent rather than emit an impossible fix.
func smCompileInLoop(fn *ssa.Function, ctx *fnContext) []Finding {
	return callsInLoops(fn, ctx, sm4Names, false, "SM4",
		"regexp compiled inside a loop; hoist the pattern",
		func(c *ssa.Call, lp *loopnest.Loop) bool {
			args := c.Call.Args
			return len(args) > 0 && loopInvariant(args[0], lp)
		})
}

// smSortInLoop fires when a sorting function is called inside a data-dependent
// loop ON A SLICE THAT IS BOTH STABLE ACROSS ITERATIONS AND UNTOUCHED BY THE
// LOOP — the only condition under which the 2nd..nth sorts are redundant and
// "hoist or restructure" is true advice. A slice rebuilt, appended to, or
// written through inside the loop must be re-sorted, so the rule stays silent.
//
// This is the same test smLinearScan (SM2) applies below, extended with the
// non-mutation half that a mutable operand needs and an immutable one does not.
func smSortInLoop(fn *ssa.Function, ctx *fnContext) []Finding {
	return callsInLoops(fn, ctx, sm5Names, true, "SM5",
		"sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure",
		func(c *ssa.Call, lp *loopnest.Loop) bool {
			s, ok := sortedSliceOperand(&c.Call)
			if !ok {
				return false
			}
			return loopInvariant(s, lp) && unmutatedIn(s, lp)
		})
}

// operandOK decides whether a matched call is actionable with respect to lp,
// the innermost loop enclosing it. Returning false keeps the rule silent — the
// smell analogue of ⊤.
type operandOK func(c *ssa.Call, lp *loopnest.Loop) bool

// callsInLoops scans every call in fn whose callee origin matches names and is
// enclosed by at least one natural loop, and fires when ok accepts it.
//
// The structure mirrors smLinearScan (SM2) rather than walking the loop forest:
// scanning blocks visits each call exactly once, so no dedup set is needed, and
// it makes the enclosing-loop chain available per call.
//
// The gate and the predicate read DIFFERENT loops on purpose. needDataDep asks
// whether ANY enclosing loop is data-dependent — a sort in a constant-trip loop
// nested inside a data-dependent one is still repeated a data-dependent number
// of times. The predicate is evaluated against the INNERMOST enclosing loop
// (by Depth — EnclosingLoops does not guarantee an order),
// because hoisting out of that loop is already a win and requiring invariance
// across every enclosing loop would discard those cases.
func callsInLoops(fn *ssa.Function, ctx *fnContext, names map[string]bool, needDataDep bool, rule, msg string, ok operandOK) []Finding {
	var out []Finding
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			call, isCall := instr.(*ssa.Call)
			if !isCall {
				continue
			}
			origin, resolved := calleeOrigin(&call.Call)
			if !resolved || !names[origin] {
				continue
			}
			encl := ctx.forest.EnclosingLoops(b)
			if len(encl) == 0 {
				continue
			}
			if needDataDep && !anyDataDep(encl, ctx) {
				continue
			}
			if !ok(call, innermost(encl)) {
				continue
			}
			out = append(out, Finding{Pos: call.Pos(), Rule: rule, Message: msg})
		}
	}
	return out
}

// innermost returns the deepest loop in the chain. EnclosingLoops explicitly
// does NOT guarantee an order — its doc directs callers that need one to sort
// by Depth — and the slice is built from a map, so indexing it positionally
// picks a different loop from run to run.
func innermost(loops []*loopnest.Loop) *loopnest.Loop {
	best := loops[0]
	for _, lp := range loops[1:] {
		if lp.Depth > best.Depth {
			best = lp
		}
	}
	return best
}

// anyDataDep reports whether any loop in the chain is data-dependent.
func anyDataDep(loops []*loopnest.Loop, ctx *fnContext) bool {
	for _, lp := range loops {
		if ctx.dataDep[lp] {
			return true
		}
	}
	return false
}

// smLinearScan fires (SM2) on a linear-scan helper (slices.Contains/Index and
// their *Func forms) called inside a data-dependent loop when:
//   - the scanned slice is an *ssa.Parameter (entry-stable by construction — the
//     same slice every iteration), and
//   - the needle is loop-varying (defined inside the loop or a loop phi).
//
// v1 is restricted to *ssa.Parameter scan targets (severability valve): a slice
// rebuilt or appended inside the loop is not provably the same slice across
// iterations, so the detector stays silent rather than guess.
func smLinearScan(fn *ssa.Function, ctx *fnContext) []Finding {
	var out []Finding
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			origin, ok := calleeOrigin(&call.Call)
			if !ok || !sm2Names[origin] {
				continue
			}
			if len(call.Call.Args) < 2 {
				continue
			}
			sliceArg, needle := call.Call.Args[0], call.Call.Args[1]
			// (a) the call is in a data-dependent loop.
			lp := enclosingDataDepLoop(b, ctx)
			if lp == nil {
				continue
			}
			// (b) the scan target is entry-stable: an *ssa.Parameter. (A field-read
			// stability check could widen this; v1 keeps the provable core.)
			if _, ok := sliceArg.(*ssa.Parameter); !ok {
				continue
			}
			// (c) the needle is loop-varying: its defining instruction is inside the
			// loop, or it is a loop phi.
			if !needleLoopVarying(needle, lp) {
				continue
			}
			out = append(out, Finding{
				Pos:     call.Pos(),
				Rule:    "SM2",
				Message: "repeated linear scan over the same slice (quadratic); build a map/set once before the loop",
			})
		}
	}
	return out
}

// needleLoopVarying reports whether v is defined inside lp or is one of lp's
// header phis — i.e. it changes across iterations, making the scan not hoistable.
func needleLoopVarying(v ssa.Value, lp *loopnest.Loop) bool {
	if phi, ok := v.(*ssa.Phi); ok && phi.Block() == lp.Header {
		return true
	}
	switch v := v.(type) {
	case ssa.Instruction:
		return lp.Blocks[v.Block()]
	case ssa.Value: // *ssa.Parameter etc. have no defining instruction
		return false
	}
	return false
}

// enclosingDataDepLoop returns the nearest data-dependent natural loop containing
// b, or nil.
func enclosingDataDepLoop(b *ssa.BasicBlock, ctx *fnContext) *loopnest.Loop {
	for _, lp := range ctx.forest.EnclosingLoops(b) {
		if ctx.dataDep[lp] {
			return lp
		}
	}
	return nil
}
