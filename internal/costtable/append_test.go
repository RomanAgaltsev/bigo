package costtable

import "testing"

// Bulk append copies its spread argument, so it costs len(src) — the same work
// appendSpace already charges on the space axis.
//
// The space side of this was fixed after an oracle-confirmed WRONG space bound
// (#87 probe, B3); the time side kept charging amortized O(1) regardless of the
// spread, which under-approximates. An under-approximation is the direction the
// prime directive exists to prevent: it can turn into a false Within against a
// budget rather than a safe top.
func TestAppendChargesTheSpread(t *testing.T) {
	tests := []struct {
		name, src, want string
		ok              bool
	}{
		{"scalar append stays amortized O(1)", `package input
func f(xs []int) []int { return append(xs, 1) }`, "O(1)", true},
		{"spread of a sized slice costs its length", `package input
func f(dst, src []int) []int { return append(dst, src...) }`, "O(len(src))", true},
		{"spread into a fresh slice costs the source length", `package input
func f(src []int) []int { return append([]int{0}, src...) }`, "O(len(src))", true},
		{"spread of a string into bytes costs its length", `package input
func f(dst []byte, s string) []byte { return append(dst, s...) }`, "O(len(s))", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := costOf(t, tt.src)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("cost = %q, want %q", got, tt.want)
			}
		})
	}
}
