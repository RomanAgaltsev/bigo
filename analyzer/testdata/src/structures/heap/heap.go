// Package heap is the canonical-structures corpus for binary heaps.
package heap

// PeekMin is O(1). Bounded today: no loops.
//
//bigo:max O(1)
func PeekMin(h []int) (int, bool) {
	if len(h) == 0 {
		return 0, false
	}
	return h[0], true
}

// SiftDown is O(log n) — under the precondition 0 <= i < len(h). bigo
// CORRECTLY refuses: for i = -1 the loop never terminates (2i+1 stays
// negative), so no bound exists without the precondition. Graduates with:
// value-range preconditions on parameters (a future annotation feature).
//
//bigo:max O(log n)
func SiftDown(h []int, i int) { // want `cannot verify budget O\(log\(len\(h\)\)\)`
	for 2*i+1 < len(h) {
		c := 2*i + 1
		if c+1 < len(h) && h[c+1] < h[c] {
			c++
		}
		if h[i] <= h[c] {
			return
		}
		h[i], h[c] = h[c], h[i]
		i = c
	}
}

// SiftUp is O(log n) in the heap size — and O(log i) in its start index,
// which is the bound bigo can prove (i <= len(h) is a precondition it cannot
// see). Bounded since the loop-algebra slice.
//
//bigo:max O(log k) where k=i
func SiftUp(h []int, i int) {
	for i > 0 {
		p := (i - 1) / 2
		if h[p] <= h[i] {
			return
		}
		h[p], h[i] = h[i], h[p]
		i = p
	}
}

// SiftRoot is O(log n): sift the root down. Bounded since the loop-algebra
// slice (geometric induction from the constant 0).
//
//bigo:max O(log n)
func SiftRoot(h []int) {
	i := 0
	for 2*i+1 < len(h) {
		c := 2*i + 1
		if c+1 < len(h) && h[c+1] < h[c] {
			c++
		}
		if h[i] <= h[c] {
			return
		}
		h[i], h[c] = h[c], h[i]
		i = c
	}
}
