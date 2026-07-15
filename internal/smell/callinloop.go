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
// natural loop — recompiling even a constant number of times is the bug.
func smCompileInLoop(_ *ssa.Function, ctx *fnContext) []Finding {
	return callsInLoops(nil, ctx, sm4Names, false, "SM4",
		"regexp compiled inside a loop; hoist the pattern")
}

// smSortInLoop fires when a sorting function is called inside a data-dependent
// loop — composed O(n·m log m).
func smSortInLoop(_ *ssa.Function, ctx *fnContext) []Finding {
	return callsInLoops(nil, ctx, sm5Names, true, "SM5",
		"sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure")
}

// callsInLoops walks every instruction in every loop body and fires once per
// call whose callee origin matches names. When needDataDep is true, only
// data-dependent loops qualify.
func callsInLoops(_ *ssa.Function, ctx *fnContext, names map[string]bool, needDataDep bool, rule, msg string) []Finding {
	var out []Finding
	seen := map[*ssa.Call]bool{} // fire at most once per call site
	for _, root := range ctx.forest.Roots {
		walkLoopCalls(root, ctx, names, needDataDep, rule, msg, &out, seen)
	}
	return out
}

func walkLoopCalls(lp *loopnest.Loop, ctx *fnContext, names map[string]bool, needDataDep bool, rule, msg string, out *[]Finding, seen map[*ssa.Call]bool) {
	if !needDataDep || ctx.dataDep[lp] {
		for b := range lp.Blocks {
			for _, instr := range b.Instrs {
				call, ok := instr.(*ssa.Call)
				if !ok {
					continue
				}
				if seen[call] {
					continue
				}
				origin, ok := calleeOrigin(&call.Call)
				if !ok || !names[origin] {
					continue
				}
				seen[call] = true
				*out = append(*out, Finding{Pos: call.Pos(), Rule: rule, Message: msg})
			}
		}
	}
	for _, c := range lp.Children {
		walkLoopCalls(c, ctx, names, needDataDep, rule, msg, out, seen)
	}
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
