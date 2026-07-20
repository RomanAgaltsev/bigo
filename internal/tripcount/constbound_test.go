package tripcount

import "testing"

// TestConstantTripCounts: a loop bounded by a compile-time constant runs a
// constant number of times and contributes O(1). bigo reported ⊤ for all of
// these through v1.35.0 — including `for i := 0; i < 3; i++` — because each
// rule ends by asking UpperExtent for a size VARIABLE and a constant has no
// name to return.
func TestConstantTripCounts(t *testing.T) {
	tests := []struct{ name, src string }{
		{"R1 increasing to a literal", `package input
func f() int { s := 0; for i := 0; i < 256; i++ { s++ }; return s }`},

		{"R1 increasing to a named constant", `package input
const N = 8
func f() int { s := 0; for i := 0; i < N; i++ { s++ }; return s }`},

		{"R2 decreasing from a constant", `package input
func f() int { s := 0; for i := 256; i > 0; i-- { s++ }; return s }`},

		{"R3 geometric up to a constant", `package input
func f() int { s := 0; for i := 1; i < 256; i *= 2 { s++ }; return s }`},

		{"R4 geometric down from a constant", `package input
func f() int { s := 0; for i := 256; i > 1; i /= 2 { s++ }; return s }`},

		// len of a fixed-size ARRAY is a compile-time constant in Go, and SSA
		// folds it — so this arm covers the array idioms for free. Pinned so a
		// future SSA change that stops folding becomes visible rather than
		// silently costing capability.
		{"len of a fixed-size array", `package input
func f() int { var a [256]int; s := 0; for i := 0; i < len(a); i++ { s += a[i] }; return s }`},

		{"range over a fixed-size array", `package input
func f() int { var a [256]int; s := 0; for range a { s++ }; return s }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)).String(); got != "O(1)" {
				t.Errorf("Of = %q, want O(1)", got)
			}
		})
	}
}

// TestConstantTripCountRejects holds the init precondition shut. A constant
// GUARD is not licence to skip the init check: `for i := m; i < 256; i++` with
// a parameter m runs 256-m times, which is unbounded below (finding S1/B1).
func TestConstantTripCountRejects(t *testing.T) {
	tests := []struct{ name, src string }{
		{"parameter init under a constant guard", `package input
func f(m int) int { s := 0; for i := m; i < 256; i++ { s++ }; return s }`},

		{"mixed inits: one constant, one len(s)", `package input
func f(s []int, c bool) int {
	i := 256
	if c { i = len(s) }
	t := 0
	for ; i > 0; i-- { t++ }
	return t
}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)); !got.IsTop() {
				t.Errorf("Of = %q, want Top — a parameter init has no provable bound, whatever the guard is", got.String())
			}
		})
	}
}

// TestDecreasingFromParameterStaysLinear guards a capability the constant arm
// must NOT swallow. For the DECREASING rules a parameter init is legitimately
// bounded — an integer parameter is its own extent, so `for i := m; i > 0; i--`
// is O(m) and has been since R2 shipped. Only an ALL-CONSTANT init set may
// collapse to O(1); turning this into O(1) would be a wrong bound.
func TestDecreasingFromParameterStaysLinear(t *testing.T) {
	src := `package input
func f(m int) int { s := 0; for i := m; i > 0; i-- { s++ }; return s }`
	if got := Of(innerLoop(t, src)).String(); got != "O(m)" {
		t.Errorf("Of = %q, want O(m) — a parameter init is its own extent here", got)
	}
}
