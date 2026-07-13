package recurrence

import "github.com/RomanAgaltsev/bigo/internal/bound"

// termsKind classifies the step shape of a recurrence's self-calls, selecting
// the solver family in Solve.
type termsKind int

const (
	mixed  termsKind = iota // a blend of subtractive and divisive steps (or empty)
	allSub                  // every step is subtractive (n − c)
	allDiv                  // every step is divisive (n / b)
)

// kindOf returns allSub when every term is a subtractive step, allDiv when every
// term is divisive, and mixed otherwise (including the empty slice).
func kindOf(terms []sizeStep) termsKind {
	if len(terms) == 0 {
		return mixed
	}
	sub, div := true, true
	for _, t := range terms {
		switch t.kind {
		case stepSub:
			div = false
		case stepDiv:
			sub = false
		default:
			return mixed
		}
	}
	switch {
	case sub:
		return allSub
	case div:
		return allDiv
	default:
		return mixed
	}
}

// solveSubtractive solves T(n) = a·T(n−c) + f(n). a=1 gives O(n·f(n)) — at most
// n/c levels, each ≤ f(n) work. a≥2 is Θ(a^(n/c)), exponential and
// unrepresentable in poly-log monomials → ⊤.
func solveSubtractive(r rec) (bound.Bound, bool) {
	if len(r.terms) != 1 {
		return bound.Top(), false
	}
	// Each level does f(n) work over n/c levels: multiply f by the measure.
	return r.work.Mul(bound.Of(bound.Term(r.measure))), true
}
