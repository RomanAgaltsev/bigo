package tripcount

import "testing"

// TestRuleIncreasingNonNegOffset pins C5: a guard offset that is not a
// constant but is provably >= 0. The soundness argument is directional — once
// phi > e, phi+b >= phi > e for any b >= 0, so the guard fails — which is why
// no loop-invariance requirement appears here.
func TestRuleIncreasingNonNegOffset(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			// The NaiveSearch shape. len(pat) is non-negative by Go's
			// semantics, provable since v1.31.0.
			"len offset",
			`package input
func f(text, pat string) int {
	s := 0
	for i := 0; i+len(pat) <= len(text); i++ { s++ }
	return s
}`,
			"O(len(text))",
		},
		{
			// THE CONTROL THAT MATTERS. The offset varies inside the loop and
			// is never negative, so the loop genuinely runs <= len(a)+1 times.
			// A rule that additionally demanded loop-invariance would refuse
			// this for no reason — that is exactly the over-tightening this
			// case exists to catch.
			"offset varies but stays non-negative",
			`package input
func f(a []int, c bool) int {
	s, k := 0, 0
	for i := 0; i+k <= len(a); i++ {
		if c { k = 5 } else { k = 0 }
		s++
	}
	return s
}`,
			"O(len(a))",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)).String(); got != tt.want {
				t.Errorf("Of = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRuleIncreasingOffsetRejects holds finding S1's gate shut. A negative
// offset genuinely shifts the trip count by an unbounded amount, so only the
// SIGN may be relaxed — never the requirement itself.
func TestRuleIncreasingOffsetRejects(t *testing.T) {
	tests := []struct{ name, src string }{
		{
			// S1 verbatim: m = -1000000 runs len(a)+1000000 times.
			"bare parameter offset",
			`package input
func f(a []int, m int) int {
	s := 0
	for i := 0; i+m <= len(a); i++ { s++ }
	return s
}`,
		},
		{
			"offset may go negative",
			`package input
func f(a []int, c bool) int {
	s, k := 0, 0
	for i := 0; i+k <= len(a); i++ {
		if c { k = -1000000 } else { k = 0 }
		s++
	}
	return s
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)); !got.IsTop() {
				t.Errorf("Of = %q, want Top — a negative-capable offset is finding S1", got.String())
			}
		})
	}
}

// TestRuleGeometricNonNegOffset covers the OTHER call site the threading
// widens. R3's claim survives the same way R1's does: once phi > e,
// phi+b >= phi > e, and phi doubles, so it exceeds e within log2 steps.
func TestRuleGeometricNonNegOffset(t *testing.T) {
	src := `package input
func f(a []int, pat string) int {
	s := 0
	for i := 1; i+len(pat) <= len(a); i *= 2 { s++ }
	return s
}`
	if got := Of(innerLoop(t, src)).String(); got != "O(log(len(a)))" {
		t.Errorf("Of = %q, want O(log(len(a)))", got)
	}
}

// TestRuleIncreasingOffsetOnTheLeft is the regression test for the v1.33.0
// review's F2. The ADD arm tries bo.Y as the offset first; widening that test
// from "is a constant" to "is provably >= 0" made it match a non-negative
// induction PHI, so `3 + i` fired the first branch on i, returned
// mulOfPhi(3) = false, and never tried the symmetric arm that works.
//
// `3+i <= len(a)` bounded at O(len(a)) in v1.31.0 and v1.32.0 and went ⊤ in
// v1.33.0 — a silent capability loss shipped by a widening.
//
// The lesson these cases pin: widening a predicate inside an ORDERED CHAIN
// changes which branch fires, not just how often. Both operand orders must be
// covered, for every offset kind.
func TestRuleIncreasingOffsetOnTheLeft(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"constant offset on the left (F2 regression)",
			`package input
func f(a []int) int { s := 0; for i := 0; 3+i <= len(a); i++ { s++ }; return s }`,
			"O(len(a))",
		},
		{
			"constant offset on the right",
			`package input
func f(a []int) int { s := 0; for i := 0; i+3 <= len(a); i++ { s++ }; return s }`,
			"O(len(a))",
		},
		{
			"len offset on the left",
			`package input
func f(text, pat string) int { s := 0; for i := 0; len(pat)+i <= len(text); i++ { s++ }; return s }`,
			"O(len(text))",
		},
		{
			"len offset on the right",
			`package input
func f(text, pat string) int { s := 0; for i := 0; i+len(pat) <= len(text); i++ { s++ }; return s }`,
			"O(len(text))",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)).String(); got != tt.want {
				t.Errorf("Of = %q, want %q", got, tt.want)
			}
		})
	}
}
