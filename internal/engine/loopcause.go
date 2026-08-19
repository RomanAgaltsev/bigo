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
	seen map[*loopnest.Loop]bool
}

func newLoopFactor(fn *ssa.Function, stab *fieldpath.Stability) *loopFactor {
	return &loopFactor{fn: fn, stab: stab, seen: map[*loopnest.Loop]bool{}}
}

// of returns lp's trip count, appending a CauseLoop the first time lp is seen ⊤.
func (lf *loopFactor) of(lp *loopnest.Loop, causes *[]Cause) bound.Bound {
	tc := tripcount.Of(lp, lf.stab)
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
// The guard EXPRESSION is the right anchor, and the scan order below was
// chosen by measurement rather than by guess:
//
//   - If.Cond lands on the loop's condition — `for n > 0` and the `i < len(xs)`
//     of a three-clause for. For a bare `for {}` it lands on the break test,
//     which is inside the body but still identifies the loop.
//   - The header's first positioned instruction is NOT a good substitute: it is
//     the induction phi, and a phi's position is its VARIABLE'S DECLARATION. It
//     coincides with the `for` line only when the variable is declared in the
//     for clause; for a loop over a parameter it points at the signature. It is
//     kept only as a fallback, where something beats nothing.
//   - A header with nothing positioned falls back to the function's own
//     declaration.
func loopPos(fn *ssa.Function, lp *loopnest.Loop) token.Pos {
	instrs := lp.Header.Instrs
	if len(instrs) > 0 {
		if ifi, ok := instrs[len(instrs)-1].(*ssa.If); ok {
			if p := ifi.Cond.Pos(); p.IsValid() {
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
