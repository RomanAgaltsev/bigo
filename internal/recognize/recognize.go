// Package recognize matches functions against known algorithmic shapes and
// reports the conventional bound for that shape.
//
// A recognition is ADVISORY and carries zero verdict power: it is a rendered
// string plus metadata, never a bound.Bound, so it cannot reach bound.Check or
// a budget even by mistake. That firewall is the same one internal/smell has,
// and it is structural rather than a convention.
//
// Every recognizer refuses when the shape is not exact. A recognizer that
// fires on a near-miss is worse than one that never fires, because the whole
// value of the channel is that a named pattern actually is that pattern. No
// recognizer may ever be generated, inferred, or wildcarded.
package recognize

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// Kind says which case a recognized bound describes. Labelling an amortized
// worst-case bound "expected" would ship a false statement, so the kind is
// carried explicitly rather than implied.
type Kind string

// The cases a recognized bound can describe.
const (
	KindWorst     Kind = "worst"
	KindAmortized Kind = "amortized"
	KindExpected  Kind = "expected"
)

// Recognition is one matched shape. Bound is RENDERED TEXT, deliberately not a
// bound.Bound: it may name a quantity the bound algebra cannot express, which
// is only sound because a recognition never participates in a verdict.
//
// Assumption is not decoration. It states the preconditions the bound rests
// on, including any that bigo cannot itself verify, and it is what makes an
// advisory claim honest rather than merely unproved.
type Recognition struct {
	Pos        token.Pos
	Pattern    string
	Kind       Kind
	Bound      string
	Assumption string
}

// recognizer analyzes one function and returns its recognitions.
type recognizer func(fn *ssa.Function) []Recognition

var recognizers = []recognizer{twoPointer}

// Detect runs every recognizer over fn.
func Detect(fn *ssa.Function) []Recognition {
	if fn == nil || fn.Blocks == nil {
		return nil
	}
	var out []Recognition
	for _, r := range recognizers {
		out = append(out, r(fn)...)
	}
	return out
}
