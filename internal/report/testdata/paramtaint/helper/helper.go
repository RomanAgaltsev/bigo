// Package helper holds the parametric function in a DIFFERENT package from its
// callers, so the callers' resolver never runs InferTop on it — which is what
// makes the memo-hit guard insufficient on its own (2026-08-11 review F2).
package helper

func slow(m map[int]int, k int) int {
	for k != 0 {
		k = m[k]
	}
	return k
}

// Run takes a func parameter, so callers price it through paramSummaryOf.
func Run(f func(), m map[int]int) {
	slow(m, 1)
	f()
}
