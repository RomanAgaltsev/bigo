// Package smell implements advisory complexity smell rules (SM1–SM8). Each rule
// fires only on a provable SSA pattern — the smell analogue of ⊤: when a
// detector cannot prove the pattern, it stays silent.
//
// Smells are firewalled from verdicts: a smell never reads from or writes to a
// budget, a bound.Check result, or any verdict state. The analyzer runs the
// smell pass after and independent of the budget pass. Diagnostics carry the
// fixed prefix "smell(SMn):" so they form a class golangci-lint users filter on.
//
// Discovery notes (Task 0 of the implementation plan):
//   - Constant-trip loops: tripcount.Of returns ⊤ for `for i := 0; i < 10; i++`
//     (UpperExtent rejects a *ssa.Const bound), so constantTrip is detected
//     structurally here, not via tripcount. ⊤ from tripcount is ambiguous
//     (constant-trip AND data-dependent both read ⊤), which is why the loop
//     family helpers below inspect the exit comparison directly.
//   - Ignore representation: //bigo:ignore reaches the analyzer as a verb in
//     directive.FuncDirectives.Dirs, retrievable via directive.Verb(fd.Dirs,
//     annotation.Ignore). The analyzer skips ignored decls before calling Detect.
package smell

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/sizefacts"
	"github.com/RomanAgaltsev/bigo/internal/tripcount"
)

// AllRules is every smell rule, in canonical ID order.
var AllRules = []string{"SM1", "SM2", "SM3", "SM4", "SM5", "SM6", "SM7", "SM8"}

// Finding is one smell diagnostic. Rule is the canonical ID (SM1..SM8);
// Message excludes the "smell(SMn):" prefix (the analyzer adds it).
type Finding struct {
	Pos     token.Pos
	Rule    string
	Message string
}

// detector analyzes one function and returns its findings.
type detector func(fn *ssa.Function, ctx *fnContext) []Finding

// registry maps rule IDs to detectors.
var registry = map[string]detector{}

// register adds a detector for a rule id. Called from init in each detector file.
func register(id string, d detector) {
	registry[id] = d
}

// fnContext carries the per-function analysis built once and shared across all
// detectors in one Detect call.
type fnContext struct {
	forest *loopnest.Forest
	stab   *fieldpath.Stability
	// dataDep maps a natural loop to true when its trip count is NOT a
	// compile-time constant (⊤ counts as data-dependent for the purposes of
	// SM1/SM5, which only exclude constant-trip loops).
	dataDep map[*loopnest.Loop]bool
	// resolvable maps a natural loop to its trip-count bound when that bound is
	// resolvable AND non-constant (SM3/SM6 need a nameable bound).
	resolvable map[*loopnest.Loop]bound.Bound
}

// Detect runs every enabled rule's detector over fn and returns the combined
// findings. A nil/empty enabled map runs nothing.
func Detect(fn *ssa.Function, enabled map[string]bool) []Finding {
	if fn == nil || len(enabled) == 0 {
		return nil
	}
	ctx := &fnContext{
		forest:     loopnest.Build(fn),
		stab:       fieldpath.Analyze(fn),
		dataDep:    map[*loopnest.Loop]bool{},
		resolvable: map[*loopnest.Loop]bound.Bound{},
	}
	for _, root := range ctx.forest.Roots {
		classifyLoops(root, ctx)
	}
	var out []Finding
	for _, id := range AllRules {
		if !enabled[id] {
			continue
		}
		d, ok := registry[id]
		if !ok {
			continue
		}
		out = append(out, d(fn, ctx)...)
	}
	return out
}

// classifyLoops walks the loop forest, marking each loop data-dependent and
// recording any resolvable non-constant trip bound.
func classifyLoops(lp *loopnest.Loop, ctx *fnContext) {
	tc := tripcount.Of(lp, ctx.stab)
	constTrip := constantTrip(lp)
	ctx.dataDep[lp] = !constTrip // ⊤ and named bounds both count as data-dependent
	if !tc.IsTop() && !constTrip {
		// Non-⊤ trip count that is NOT a constant — the loop has a nameable bound.
		// (tripcount never returns Constant() for `i < 10`, so the constTrip check
		// is belt-and-suspenders; the main gate is !tc.IsTop().)
		ctx.resolvable[lp] = tc
	}
	for _, c := range lp.Children {
		classifyLoops(c, ctx)
	}
}

// constantTrip reports whether lp has a compile-time constant trip count. It
// inspects the loop header's exit comparison: an induction `i` stepped by a
// constant, compared against a constant bound. This covers `for i := 0; i < N;
// i++` for constant N — the shape tripcount.Of returns ⊤ for (UpperExtent
// rejects a *ssa.Const), but which is NOT data-dependent.
func constantTrip(lp *loopnest.Loop) bool {
	h := lp.Header
	if len(h.Instrs) == 0 {
		return false
	}
	ifi, ok := h.Instrs[len(h.Instrs)-1].(*ssa.If)
	if !ok || len(ifi.Block().Succs) != 2 || !lp.Blocks[ifi.Block().Succs[0]] {
		return false
	}
	cmp, ok := ifi.Cond.(*ssa.BinOp)
	if !ok {
		return false
	}
	switch cmp.Op {
	case token.LSS, token.LEQ, token.GTR, token.GEQ, token.EQL, token.NEQ:
	default:
		return false
	}
	// The bound operand must be a compile-time constant.
	return isConstBound(cmp.X) && isConstInduction(cmp.Y, h) ||
		isConstBound(cmp.Y) && isConstInduction(cmp.X, h)
}

// isConstBound reports whether v is a compile-time integer constant.
func isConstBound(v ssa.Value) bool {
	if _, ok := sizefacts.ConstIntV(v); ok {
		return true
	}
	return false
}

// isConstInduction reports whether v is a phi in header h whose step edge is a
// constant increment/decrement and whose init edge is a constant. Loose
// (accepts any constant-init/constant-step phi) — combined with a constant
// bound operand this suffices to prove a constant trip count.
func isConstInduction(v ssa.Value, h *ssa.BasicBlock) bool {
	phi, ok := v.(*ssa.Phi)
	if !ok || phi.Block() != h {
		return false
	}
	for _, e := range phi.Edges {
		switch e := e.(type) {
		case *ssa.Const:
			// constant init
		case *ssa.BinOp:
			if e.Op != token.ADD && e.Op != token.SUB {
				return false
			}
			// the step must be phi +/- const: pick the non-phi operand
			other := e.Y
			if e.Y == phi {
				other = e.X
			}
			if e.X != phi && e.Y != phi {
				return false // neither operand is the phi
			}
			if _, ok := sizefacts.ConstIntV(other); !ok {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// ParseRules parses the -smells flag value into an enabled-set.
//   - "all" or "" (empty) → every rule enabled
//   - "none" → empty set (no rules)
//   - "SM1,SM4" → exactly those named rules
//
// An unknown rule ID is an error.
func ParseRules(flag string) (map[string]bool, error) {
	flag = strings.TrimSpace(flag)
	if flag == "" || flag == "all" {
		out := make(map[string]bool, len(AllRules))
		for _, r := range AllRules {
			out[r] = true
		}
		return out, nil
	}
	if flag == "none" {
		return map[string]bool{}, nil
	}
	out := map[string]bool{}
	for _, part := range strings.Split(flag, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !isKnownRule(part) {
			return nil, fmt.Errorf("unknown smell rule %q (valid: all, none, or SM1..SM8)", part)
		}
		out[part] = true
	}
	return out, nil
}

func isKnownRule(id string) bool {
	for _, r := range AllRules {
		if r == id {
			return true
		}
	}
	return false
}

// dataDependentLoops returns every natural loop in fn whose trip count is NOT a
// compile-time constant. ⊤ counts as data-dependent: an unknown trip count is
// still data-dependent for smell purposes (rules that must NAME the bound,
// SM3/SM6, additionally consult the resolvable map on fnContext).
func dataDependentLoops(_ *ssa.Function, ctx *fnContext) []*loopnest.Loop {
	var out []*loopnest.Loop
	for _, root := range ctx.forest.Roots {
		collectDataDep(root, ctx, &out)
	}
	return out
}

func collectDataDep(lp *loopnest.Loop, ctx *fnContext, out *[]*loopnest.Loop) {
	if ctx.dataDep[lp] {
		*out = append(*out, lp)
	}
	for _, c := range lp.Children {
		collectDataDep(c, ctx, out)
	}
}

// calleeOrigin returns the full path name of a static call's callee, resolving
// generic instantiations to their origin (slices.Sort, not slices.Sort[int]).
func calleeOrigin(c *ssa.CallCommon) (string, bool) {
	callee := c.StaticCallee()
	if callee == nil {
		return "", false
	}
	if o := callee.Origin(); o != nil {
		callee = o
	}
	return callee.String(), true
}

// isString reports whether t's underlying type is the string basic type.
func isString(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.String
}
