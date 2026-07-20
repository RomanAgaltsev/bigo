package tripcount

import "testing"

// TestRuleGeometricShiftStep pins C10: R4 reads `phi >> k` for a constant
// k >= 1 as division by 2^k. R4's guard already forces phi >= 1 inside the
// loop, so the sign question that makes arithmetic SHR differ from truncating
// QUO cannot arise.
func TestRuleGeometricShiftStep(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"binary exponentiation: b >>= 1",
			`package input
func f(b int) int { s := 0; for b > 0 { b >>= 1; s++ }; return s }`,
			"O(log(b))",
		},
		{
			"wider constant shift: b >>= 3",
			`package input
func f(b int) int { s := 0; for b > 0 { b >>= 3; s++ }; return s }`,
			"O(log(b))",
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

// TestRuleGeometricShiftRejects holds the shift gate shut. Both shapes are
// non-terminating or non-decreasing, and each would get a finite bound if the
// gate were dropped.
func TestRuleGeometricShiftRejects(t *testing.T) {
	tests := []struct{ name, src string }{
		{
			// k = 0 makes the step the identity: b never moves.
			"zero shift is a fixed point",
			`package input
func f(b int) int { s := 0; for b > 0 { b >>= 0; s++ }; return s }`,
		},
		{
			// A variable amount may be 0 at runtime, so it proves nothing.
			"variable shift amount",
			`package input
func f(b, k int) int { s := 0; for b > 0 { b >>= k; s++ }; return s }`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)); !got.IsTop() {
				t.Errorf("Of = %q, want Top — the shift does not prove a decrease", got.String())
			}
		})
	}
}

// TestRuleIncreasingSquareGuard pins C6: R1 reads `phi*phi <= n` as bounding
// phi by n, since phi <= phi*phi for phi >= 1.
func TestRuleIncreasingSquareGuard(t *testing.T) {
	// Trial division. The bound is the conservative O(n), not O(sqrt(n)) —
	// sqrt is inexpressible in the poly-log algebra.
	src := `package input
func f(n int) int { s := 0; for d := 2; d*d <= n; d++ { s++ }; return s }`
	if got := Of(innerLoop(t, src)).String(); got != "O(n)" {
		t.Errorf("Of = %q, want O(n)", got)
	}
}

// TestRuleIncreasingSquareGuardRejects holds the square gate shut. `a*b <= n`
// with DISTINCT operands proves nothing about either factor — a may be huge
// while b is tiny — so only the same-value square may match.
func TestRuleIncreasingSquareGuardRejects(t *testing.T) {
	src := `package input
func f(n, m int) int { s := 0; for d := 2; d*m <= n; d++ { s++ }; return s }`
	if got := Of(innerLoop(t, src)); !got.IsTop() {
		t.Errorf("Of = %q, want Top — a distinct-operand product bounds neither factor", got.String())
	}
}
