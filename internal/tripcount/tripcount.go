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
	ruleIncreasing,
}
