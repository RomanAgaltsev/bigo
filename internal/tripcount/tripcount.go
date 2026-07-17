// Package tripcount infers the iteration-count of a natural loop.
package tripcount

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/sizefacts"
)

// shape is one loop's exit test, extracted once per Of call and handed to
// every evolution rule.
type shape struct {
	loop *loopnest.Loop
	ifi  *ssa.If
	cmp  *ssa.BinOp // nil when the exit test is not a comparison
	f    *sizefacts.Facts
}

// Of returns the loop's iteration-count bound in canonical size variables or
// Top() when no evolution rule recognizes the shape. Rules are tried
// most-specific-first; order never affects soundness (each rule is
// independently sound), only which equivalent answer is produced.
func Of(loop *loopnest.Loop, stab *fieldpath.Stability) bound.Bound {
	h := loop.Header
	if len(h.Instrs) == 0 {
		return bound.Top()
	}
	ifi, ok := h.Instrs[len(h.Instrs)-1].(*ssa.If)
	if !ok {
		return bound.Top()
	}
	// The true branch must stay in the loop; otherwise the condition is an
	// exit test and the bound side would be misread.
	if len(ifi.Block().Succs) != 2 || !loop.Blocks[ifi.Block().Succs[0]] {
		return bound.Top()
	}
	// The false branch must LEAVE the loop. Every rule below argues, in some
	// form, "the guard fails => the loop ends": R1 exits once the comparand
	// reaches e, R7 once i >= E_a or j >= E_b. That premise holds only if the
	// guard-false edge is an exit. When it re-enters the loop — the shape
	// `for { if cond { … } else { … } }`, where this If IS the header and both
	// its edges are in-loop — the guard failing ends nothing, the loop runs on,
	// and any bound derived from the guard is unjustified.
	//
	// Checking only the true edge (as this did through v1.28.0) produced wrong
	// bounds under R1, R2, R3, R4 and R7 — including on TERMINATING code, where
	// some unrelated exit bounds the loop and the guard's variable does not
	// (`for { if i < len(a) {…} else {…}; i++; if t >= limit { break } }` was
	// reported O(len(a)) while running limit/2 times). R6 escaped only by
	// accident: its exactly-one-end rule rejects the stalling path. Both
	// goldens are blind to this family — the pins live in edge/.
	if loop.Blocks[ifi.Block().Succs[1]] {
		return bound.Top()
	}
	sh := &shape{loop: loop, ifi: ifi, f: &sizefacts.Facts{Stab: stab}}
	sh.cmp, _ = ifi.Cond.(*ssa.BinOp)

	for _, rule := range rules {
		if b, ok := rule(sh); ok {
			return b
		}
	}
	return bound.Top()
}

// rules in most-specific-first order.
var rules = []func(*shape) (bound.Bound, bool){
	ruleRangeNext,
	ruleBisection,
	ruleGeometricUp,
	ruleGeometricDown,
	ruleDecreasing,
	ruleTwoPointer,
	ruleIncreasing,
}
