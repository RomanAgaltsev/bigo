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

// uniformDiv returns the shared divisor of a purely divisive recurrence when
// every term divides by the same b, else ok=false — mixed ratios are left to
// Akra–Bazzi.
func uniformDiv(terms []sizeStep) (int, bool) {
	if len(terms) == 0 {
		return 0, false
	}
	b := terms[0].div
	for _, t := range terms[1:] {
		if t.div != b {
			return 0, false
		}
	}
	return int(b), true
}

// logBase returns k with b^k == a for a >= 1, b >= 2, else ok=false (a is not
// an integer power of b — the exponent log_b a would be non-integer, which
// poly-log monomials cannot represent).
func logBase(a, b int) (int, bool) {
	if a < 1 || b < 2 {
		return 0, false
	}
	k, v := 0, 1
	for v < a {
		v *= b
		k++
	}
	if v == a {
		return k, true
	}
	return 0, false
}

// degree returns the dominant (pow, log) of b's single monomial in variable n.
// After extract's varsSubset check every term is in {n}, so the antichain is a
// single monomial; take its max. O(1) work gives (0, 0). ⊤ or empty -> ok=false.
func degree(b bound.Bound, n bound.Var) (pow, log int, ok bool) {
	if b.IsTop() {
		return 0, 0, false
	}
	bestPow, bestLog, has := 0, 0, false
	for _, m := range b.Terms() {
		p, lg := m.FactorOf(n)
		if !has || p > bestPow || (p == bestPow && lg > bestLog) {
			bestPow, bestLog, has = p, lg, true
		}
	}
	return bestPow, bestLog, has
}

// solveMaster applies the Master theorem for T(n) = a·T(n/b) + f(n), emitting
// only when the critical exponent c* = log_b a is a non-negative integer.
//   - Case 1 (f grows slower than n^c*):  Θ(n^c*).
//   - Case 2 (f matches, f = Θ(n^c* log^l n)):  Θ(n^c* log^(l+1) n).
//   - Case 3 (f dominates):  Θ(f(n)); regularity holds for every poly-log f
//     with degree > c*.
func solveMaster(a, b int, work bound.Bound, n bound.Var) (bound.Bound, bool) {
	cstar, ok := logBase(a, b)
	if !ok {
		return bound.Top(), false
	}
	d, l, ok := degree(work, n)
	if !ok {
		return bound.Top(), false
	}
	switch {
	case d < cstar:
		return bound.Of(bound.Mono(n, cstar, 0)), true
	case d == cstar:
		return bound.Of(bound.Mono(n, cstar, l+1)), true
	default:
		return work, true
	}
}
