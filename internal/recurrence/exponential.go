package recurrence

import (
	"go/token"

	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"golang.org/x/tools/go/ssa"
)

// ProvablyExponential reports whether fn is a directly self-recursive function
// whose recurrence is provably exponential: Θ(aⁿ) for branching factor a ≥ 2.
// This is the positive smell claim SM8 makes on unannotated code — the exact
// family the solver *rejects for bounding* in solveSubtractive (a ≥ 2
// subtractive). It must be at least as strict as the solver's rejection:
// everything the extractor cannot positively classify returns false.
//
// ok only when all of:
//   - fn is directly self-recursive (selfCalls non-empty);
//   - no self-call sits inside an enclosing size loop (constant multiplicity);
//   - a measure parameter exists whose every self-call steps subtractively
//     (stepsFor returns all stepSub, at least one strict step);
//   - the termination guard holds (terminates) — a proved base, so the claim is
//     on terminating code, not a runaway;
//   - the branching factor (selfCallMult) is ≥ 2 (a=1 is linear, not exponential).
//
// a is the branching factor (e.g. naive Fibonacci → a=2). Everything else —
// divisive steps (binary search solves), mutual recursion, unguarded recursion,
// a=1 countdowns — returns false.
func ProvablyExponential(fn *ssa.Function) (a int, ok bool) {
	calls := selfCalls(fn)
	if len(calls) == 0 {
		return 0, false
	}
	// Constant multiplicity: no self-call may sit inside an enclosing loop.
	forest := loopnest.Build(fn)
	for _, c := range calls {
		if underLoop(forest, callBlock(fn, c)) {
			return 0, false
		}
	}
	for pi, p := range fn.Params {
		terms, ok := stepsFor(p, pi, calls)
		if !ok {
			continue
		}
		if !terminates(fn, p, terms, calls) {
			continue
		}
		// All steps must be subtractive — a divisive step solves (binary search),
		// and a mixed recurrence is out of scope for a positive claim.
		if !allSubtractive(terms) {
			continue
		}
		mult := selfCallMult(fn, calls)
		if mult >= 2 {
			if memoized(fn, calls) {
				return 0, false // a cache turns the recurrence linear: not provably exponential
			}
			return mult, true
		}
		return 0, false // mult < 2: linear or constant, not exponential
	}
	return 0, false
}

// allSubtractive reports whether every step is a subtractive (n−c) step. A
// divisive step (n/b) would solve via the Master theorem and is not exponential.
func allSubtractive(terms []sizeStep) bool {
	for _, t := range terms {
		if t.kind != stepSub {
			return false
		}
	}
	return true
}

// memoized reports whether fn caches its recursive results: a comma-ok map
// lookup whose block dominates every self-call, paired with a map update on the
// same map. Such a cache computes each argument once, turning the syntactic
// a≥2 recurrence linear, so SM8 must not claim it is exponential. The check is
// deliberately conservative — an advisory false negative on a genuine
// exponential that merely touches a map is safe; a false "provably exponential"
// on memoized code is not (it is the smell analogue of a wrong bound).
func memoized(fn *ssa.Function, calls []*ssa.CallCommon) bool {
	updated := map[ssa.Value]bool{}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			if upd, ok := instr.(*ssa.MapUpdate); ok {
				updated[mapRoot(upd.Map)] = true
			}
		}
	}
	if len(updated) == 0 {
		return false // no cache write: a read-only lookup table is not memoization
	}
	var callBlocks []*ssa.BasicBlock
	for _, c := range calls {
		if b := callBlock(fn, c); b != nil {
			callBlocks = append(callBlocks, b)
		}
	}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			lu, ok := instr.(*ssa.Lookup)
			if !ok || !lu.CommaOk {
				continue // a plain lookup does not short-circuit the recursion
			}
			if updated[mapRoot(lu.X)] && dominatesAll(lu.Block(), callBlocks) {
				return true
			}
		}
	}
	return false
}

// mapRoot normalizes a map value to a comparable root, peeling a single load
// (*x) so a global or captured map — read as `*ssa.UnOp{MUL, X}` — compares
// equal to the same map read elsewhere. A by-value map parameter is its own root.
func mapRoot(v ssa.Value) ssa.Value {
	if un, ok := v.(*ssa.UnOp); ok && un.Op == token.MUL {
		return un.X
	}
	return v
}

// dominatesAll reports whether b dominates every block in bs (bs non-empty).
func dominatesAll(b *ssa.BasicBlock, bs []*ssa.BasicBlock) bool {
	if len(bs) == 0 {
		return false
	}
	for _, x := range bs {
		if !ssaDominates(b, x) {
			return false
		}
	}
	return true
}

// ssaDominates reports whether a is on every path from entry to b (a dominates
// itself).
func ssaDominates(a, b *ssa.BasicBlock) bool {
	for x := b; x != nil; x = x.Idom() {
		if x == a {
			return true
		}
	}
	return false
}
