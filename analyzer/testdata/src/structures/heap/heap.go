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

// SiftDown is O(log n). Unverifiable today: the induction advances
// geometrically (i -> 2i+1). Graduates with: geometric-induction recognition
// (Phase 2 tripcount) — the same capability as array.BinarySearch.
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

// SiftUp is O(log n). Unverifiable today: geometric decrease (i -> (i-1)/2).
// Graduates with: geometric-induction recognition (Phase 2 tripcount).
//
//bigo:max O(log n)
func SiftUp(h []int, i int) { // want `cannot verify budget O\(log\(len\(h\)\)\)`
	for i > 0 {
		p := (i - 1) / 2
		if h[p] <= h[i] {
			return
		}
		h[p], h[i] = h[i], h[p]
		i = p
	}
}
