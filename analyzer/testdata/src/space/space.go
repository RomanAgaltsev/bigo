// Package space is the corpus for //bigo:space budgets. Heap allocation is
// modeled as TOTAL allocated — an upper bound on peak — so it proves Within
// only, never Exceeds. See LoopAllocGCd for the load-bearing soundness pin.
package space

//bigo:space O(n)
func MakeFill(n int) []int { // heap O(n): within
	out := make([]int, n)
	for i := 0; i < n; i++ {
		out[i] = i
	}
	return out
}

//bigo:space O(1)
func Constant(a, b int) int { // no allocation: O(1) within
	return a + b
}

// LoopAllocGCd allocates O(1) inside an n-loop; the objects are GC'd, so PEAK
// is O(1), but bigo bounds TOTAL allocation = O(n) as a safe over-approximation.
// O(n) must be `within`; O(1) must be `cannot verify` — and MUST NOT be
// `exceeds` (that would be a false positive, the space cardinal sin).

//bigo:space O(n)
func LoopAllocGCdOK(n int) { // within: total O(n) <= O(n)
	for i := 0; i < n; i++ {
		_ = make([]int, 4)
	}
}

//bigo:space O(1)
func LoopAllocGCdStrict(n int) { // want `cannot verify space budget O\(1\)`
	for i := 0; i < n; i++ {
		_ = make([]int, 4)
	}
}

//bigo:space O(1)
func UnknownMake(g func() int) []int { // want `cannot verify space budget O\(1\)`
	return make([]int, g())
}

// RecSum is all-stack: it allocates nothing, but recurses len(xs) deep, so its
// true peak space is the O(len(xs)) recursion stack (heap O(1) ∨ stack O(n)).
// Stack is a real peak, so O(n) verifies as `within` here.

//bigo:space O(n)
func RecSum(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	return xs[0] + RecSum(xs[1:])
}
