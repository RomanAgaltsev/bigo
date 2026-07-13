package recurrence

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// stubModel is a trivial CostModel for the in-package detection tests. Using
// callsummary here would form an import cycle — callsummary imports recurrence
// (to intercept Solve), and these tests live in package recurrence. The test
// functions make no external (non-self, non-builtin) calls, so costing every
// call O(1) is neutral: self-calls are held constant by selfConst, and the
// builtins present (len) are O(1) anyway.
type stubModel struct{}

func (stubModel) CallCost(*ssa.CallCommon) bound.Bound { return bound.Constant() }
