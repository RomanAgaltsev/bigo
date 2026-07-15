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

func init() {
	register("SM4", smCompileInLoop)
	register("SM5", smSortInLoop)
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
