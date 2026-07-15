package smell

import (
	"fmt"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/recurrence"
)

func init() { register("SM8", smExponential) }

// smExponential fires when recurrence.ProvablyExponential positively proves fn
// is Θ(aⁿ) — naive Fibonacci-shaped recursion. This is the differentiator: the
// diagnostic no other Go linter can make, because it needs the recurrence
// solver's termination proof and branching-factor analysis.
func smExponential(fn *ssa.Function, _ *fnContext) []Finding {
	a, ok := recurrence.ProvablyExponential(fn)
	if !ok {
		return nil
	}
	return []Finding{{
		Pos:     fn.Pos(),
		Rule:    "SM8",
		Message: fmt.Sprintf("provably exponential recursion (%d subtractive self-calls per level); consider memoization or an iterative DP", a),
	}}
}
