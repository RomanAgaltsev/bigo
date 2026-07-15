package recurrence

import (
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
