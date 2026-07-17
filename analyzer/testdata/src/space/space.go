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

//bigo:space O(1)
func NoAllocLoop(n int) int { // data-dependent loop that allocates nothing: O(1) within
	s := 0
	for i := 0; i < n; i++ {
		s += i
	}
	return s
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

// HeapOverBudget's space is a fully known O(n) — no unresolved call — so the
// message must name the known space, not a nonexistent unresolved cost.

//bigo:space O(1)
func HeapOverBudget(n int) []int { // want `cannot verify space budget O\(1\): inferred space O\(n\) is a total-allocation upper bound`
	return make([]int, n)
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

// Map growth is charged per assign and scaled by the enclosing loop, so a map
// built to the size of its input is O(len) heap. Before issue #49 this inferred
// O(1) and passed an O(1) budget silently — the space-axis blind spot.

//bigo:space O(1)
func MapGrowth(modules []string) map[string]bool { // want `cannot verify space budget O\(1\): inferred space O\(len\(modules\)\) is a total-allocation upper bound`
	out := map[string]bool{} // want `smell\(SM6\): map built without a size hint in a loop bounded by`
	for _, m := range modules {
		out[m] = true
	}
	return out
}

// The honest budget verifies — the shape is bounded, not ⊤.

//bigo:space O(n) where n = len(modules)
func MapGrowthBudgeted(modules []string) map[string]bool {
	out := map[string]bool{} // want `smell\(SM6\): map built without a size hint in a loop bounded by`
	for _, m := range modules {
		out[m] = true
	}
	return out
}

// A map assign outside any loop stays O(1): the per-assign charge is scaled by
// enclosing loop trips, so a bounded number of assigns must not inflate.

//bigo:space O(1)
func MapAssignNoLoop(k string) map[string]bool {
	out := map[string]bool{}
	out[k] = true
	return out
}

// A function proved self-recursive whose depth the solver cannot solve is ⊤
// stack, not O(1). QuickSort partitions at a data-dependent index, so no
// constant-fraction or subtractive measure exists. Before issue #76 the stack
// kept InferSpace's O(1) default — a positive claim of constant stack for a
// function proved to recurse — and this budget verified silently.

//bigo:space O(1)
func UnsolvedRecursionStack(s []int) { // want `cannot verify space budget O\(1\): recursion depth is unverifiable`
	if len(s) <= 1 {
		return
	}
	pivot := s[len(s)-1]
	i := 0
	for j := 0; j < len(s)-1; j++ {
		if s[j] < pivot {
			s[i], s[j] = s[j], s[i]
			i++
		}
	}
	s[i], s[len(s)-1] = s[len(s)-1], s[i]
	UnsolvedRecursionStack(s[:i])
	UnsolvedRecursionStack(s[i+1:])
}

// The caller-side twin: a caller of an unsolvable-recursive helper inherits an
// unbounded transient stack, so it cannot claim a bounded space either.

//bigo:space O(1)
func CallsUnsolvedRecursion(s []int) { // want `cannot verify space budget O\(1\)`
	UnsolvedRecursionStack(s)
}
