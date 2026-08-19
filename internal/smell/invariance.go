package smell

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/loopnest"
)

// loopInvariant reports whether v provably holds the same value on every
// iteration of lp.
//
// For an IMMUTABLE type this is decidable from the definition site alone. SSA
// is single-assignment, so a value whose defining instruction lies outside the
// loop body cannot be recomputed inside it, and Go strings are immutable, so no
// aliasing route can change it either. Values with no defining instruction —
// *ssa.Const, *ssa.Parameter, *ssa.FreeVar, *ssa.Global — are invariant by
// construction.
//
// It is NOT sufficient for a mutable operand: a slice defined outside the loop
// can still have its contents rewritten inside it. Callers handling mutable
// types must pair this with unmutatedIn.
func loopInvariant(v ssa.Value, lp *loopnest.Loop) bool {
	if instr, ok := v.(ssa.Instruction); ok {
		// Loop phis are covered here rather than special-cased: a header phi's
		// block is the header, which is always a member of lp.Blocks.
		return !lp.Blocks[instr.Block()]
	}
	return true
}
