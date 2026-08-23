package engine

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/tripcount"
)

// loopFactor computes per-loop trip counts and records at most one CauseLoop
// per loop.
//
// Both the time walk and the space walk visit every block and ask for its
// enclosing loops, so a loop whose body spans many blocks is asked about many
// times. Recording a cause on each ask produced one identical entry per block —
// 104 of them on a single goldmark function. The seen set makes the diagnostic
// per-loop, which is what it always claimed to be.
//
// Shared by both walks so the position and dedup rules cannot drift between
// them; the previous split had space.go delegating to a helper while engine.go
// kept its own copy.
type loopFactor struct {
	fn   *ssa.Function
	stab *fieldpath.Stability
	tr   *Trace
	seen map[*loopnest.Loop]bool
	// traced is separate from seen ON PURPOSE: seen is populated only for ⊤
	// loops, so reusing it would record bounded loops once per enclosing block
	// and unbounded ones exactly once.
	traced map[*loopnest.Loop]bool
}

func newLoopFactor(fn *ssa.Function, stab *fieldpath.Stability, tr *Trace) *loopFactor {
	return &loopFactor{
		fn: fn, stab: stab, tr: tr,
		seen:   map[*loopnest.Loop]bool{},
		traced: map[*loopnest.Loop]bool{},
	}
}

// of returns lp's trip count, appending a CauseLoop the first time lp is seen
// ⊤, and recording one LoopStep per loop when tracing.
func (lf *loopFactor) of(lp *loopnest.Loop, causes *[]Cause) bound.Bound {
	tc, rule := tripcount.OfExplained(lp, lf.stab)
	if lf.tr != nil && !lf.traced[lp] {
		lf.traced[lp] = true
		lf.tr.Loops = append(lf.tr.Loops, LoopStep{
			Pos:   loopPos(lf.fn, lp),
			Rule:  rule,
			Count: tc,
		})
	}
	if tc.IsTop() && !lf.seen[lp] {
		lf.seen[lp] = true
		*causes = append(*causes, Cause{
			Pos:  loopPos(lf.fn, lp),
			Kind: CauseLoop,
			What: "loop with unrecognized trip count",
		})
	}
	return tc
}

// loopPos names where a loop is.
//
// The obvious candidate — the header's terminating instruction — is an
// *ssa.If, and (*ssa.If).Pos() is NEVER valid, so positioning the cause there
// produced a blocker no consumer could locate. Since collect.go writes
// file/line only for a valid position, every loop cause shipped with an empty
// file and a zero line, and `loop` is the top-ranked blocker class by
// graduation count.
//
// The scan order below was chosen by measurement across grpc-go and
// prometheus, not by guess — an earlier version of this function was ordered by
// reasoning and put the header phi second, which measured at only ~35% of loop
// causes actually naming their loop:
//
//  1. If.Cond — the guard expression. Lands on the loop's condition: `for n > 0`
//     and the `i < len(xs)` of a three-clause for. For a bare `for {}` it lands
//     on the break test, inside the body but still identifying the loop.
//     Measured at 34.9% of all loop causes.
//
//  2. The loop's BODY block — the header's in-loop successor — and its first
//     positioned instruction. A range loop's guard is a tuple extract with no
//     position, so tier 1 misses it entirely; the body block is inside the loop
//     BY CONSTRUCTION, so unlike the header it cannot name a declaration living
//     outside. SSA attributes the loop-setup instruction there to the `for`
//     line, so this is exact rather than approximate. Supplies a position for
//     96.7% (grpc-go) and 97.5% (prometheus) of loops whose guard has none.
//
//  3. The header's first positioned instruction. This is the induction phi, and
//     A PHI'S POSITION IS ITS VARIABLE'S DECLARATION — it coincides with the
//     `for` line only when the variable is declared in the for clause, and for
//     a loop over a parameter it points at the signature. It is deliberately
//     BELOW the body block, and kept only for the ~3% of loops whose body block
//     has no positioned instruction. It is not deleted because it is earlier
//     than the body block in 2 loops out of ~23,600 measured.
//
//  4. The function's own declaration, which beats nothing.
func loopPos(fn *ssa.Function, lp *loopnest.Loop) token.Pos {
	instrs := lp.Header.Instrs
	if len(instrs) > 0 {
		if ifi, ok := instrs[len(instrs)-1].(*ssa.If); ok {
			if p := ifi.Cond.Pos(); p.IsValid() {
				return p
			}
			if p := bodyBlockPos(ifi, lp); p.IsValid() {
				return p
			}
		}
	}
	for _, in := range instrs {
		if p := in.Pos(); p.IsValid() {
			return p
		}
	}
	return fn.Pos()
}

// bodyBlockPos returns the first positioned instruction of the loop's body
// block — the header's in-loop successor — or NoPos when there is none.
func bodyBlockPos(ifi *ssa.If, lp *loopnest.Loop) token.Pos {
	if len(ifi.Block().Succs) != 2 {
		return token.NoPos
	}
	body := ifi.Block().Succs[0]
	if !lp.Blocks[body] {
		return token.NoPos
	}
	for _, in := range body.Instrs {
		if p := in.Pos(); p.IsValid() {
			return p
		}
	}
	return token.NoPos
}
