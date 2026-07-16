// Package reportfix is the report-schema golden fixture. Every function's
// expected document record is stated in its comment; the golden test pins the
// exact document. It deliberately contains an exceeds and an unverifiable
// function — this package is never analyzed by bigo-on-bigo CI (testdata is
// invisible to go list).
package reportfix

// Store exists to give Unverifiable an unannotated interface call — the one
// ⊤ shape that stays ⊤ until the far-future interface-resolution slice, so
// the golden does not churn when function-value costs (Now-lane 3.3) land.
type Store interface {
	Get(k string) int
}

// Linear: no directives; expected record: proven linear time, no budget.
func Linear(xs []int) int {
	s := 0
	for _, x := range xs {
		s += x
	}
	return s
}

// WithinBudget: expected verdict "within", raw budget preserved with where.
//
//bigo:max O(n) where n=len(xs)
func WithinBudget(xs []int) int {
	s := 0
	for _, x := range xs {
		s += x
	}
	return s
}

// ExceedsBudget: quadratic body under a linear budget; expected "exceeds".
//
//bigo:max O(n) where n=len(xs)
func ExceedsBudget(xs []int) int {
	s := 0
	for i := range xs {
		for j := range xs {
			s += xs[i] * xs[j]
		}
	}
	return s
}

// Unverifiable: unannotated interface call; expected time top, verdict
// "unverifiable", one cause of kind "call".
//
//bigo:max O(1)
func Unverifiable(s Store) int {
	return s.Get("k")
}

// Tree carries the method fixture.
type Tree struct {
	items []int
}

// Sum: method with pointer receiver; expected record has receiver "*Tree"
// and a proven bound over the field size.
func (t *Tree) Sum() int {
	s := 0
	for _, x := range t.items {
		s += x
	}
	return s
}

// telemetry: expected in the trusted table and in its record's trust list.
//
//bigo:ignore
func telemetry() {}

// extLookup: linear body asserted O(log n) for callers; expected in the
// trusted table. (The assertion is deliberately tighter than the body — the
// point of //bigo:cost is that bigo trusts it, and the report must expose it.)
//
//bigo:cost O(log n) where n=len(xs)
func extLookup(xs []int, k int) int {
	for i := range xs {
		if xs[i] == k {
			return i
		}
	}
	return -1
}

// UsesTrust: expected "within" — provable only via extLookup's cost assertion
// and telemetry's ignore, both visible in the document's trust surface.
//
//bigo:max O(log n) where n=len(xs)
func UsesTrust(xs []int, k int) int {
	telemetry()
	return extLookup(xs, k)
}

// Doubled: expected space record — heap O(n), stack O(1), verdict "within".
//
//bigo:space O(n) where n=len(xs)
func Doubled(xs []int) []int {
	out := make([]int, 0, 2*len(xs))
	out = append(out, xs...)
	out = append(out, xs...)
	return out
}

// InvalidBudget: the budget names variable m with no where-binding, which
// fails normalize.Budget ("unbound size variable") — stable forever, since it
// is a user error, not an engine limitation. Expected budget record: verdict
// "invalid", raw preserved, bound absent; the time bound is still inferred.
//
//bigo:max O(m)
func InvalidBudget(xs []int) int {
	return len(xs)
}

// ConcatInLoop is a deliberate SM1: string concatenation in a data-dependent
// loop. Pinned by the report goldens.
func ConcatInLoop(xs []string) string {
	var s string
	for _, x := range xs {
		s += x
	}
	return s
}

// IgnoredSmell concatenates in a loop exactly as ConcatInLoop does, but is
// ignored — the document must stay silent about it, as the analyzer does.
//
//bigo:ignore
func IgnoredSmell(xs []string) string {
	var s string
	for _, x := range xs {
		s += x
	}
	return s
}
