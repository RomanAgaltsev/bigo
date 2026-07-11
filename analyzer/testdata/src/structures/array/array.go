// Package array is the canonical-structures corpus for contiguous sequences.
// Every function's doc comment states its TRUE complexity; //bigo:max pins the
// engine's verdict on it. Unverifiable entries name their graduation path.
package array

// LinearScan is O(n). Bounded today: counted loop over a parameter length.
//
//bigo:max O(n)
func LinearScan(xs []int, target int) int {
	for i := 0; i < len(xs); i++ {
		if xs[i] == target {
			return i
		}
	}
	return -1
}

// MinMax is O(n). Bounded today.
//
//bigo:max O(n)
func MinMax(xs []int) (int, int) {
	lo, hi := 0, 0
	for i := 0; i < len(xs); i++ {
		if xs[i] < lo {
			lo = xs[i]
		}
		if xs[i] > hi {
			hi = xs[i]
		}
	}
	return lo, hi
}

// PairSums is O(n^2). Bounded today: rectangular counted nest.
//
//bigo:max O(n^2)
func PairSums(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs); j++ {
			s += xs[i] + xs[j]
		}
	}
	return s
}

// Reverse is O(n). Bounded since the loop-algebra slice: the two-pointer gap
// shrinks every iteration and j never exceeds its initial extent.
//
//bigo:max O(n)
func Reverse(xs []int) {
	for i, j := 0, len(xs)-1; i < j; i, j = i+1, j-1 {
		xs[i], xs[j] = xs[j], xs[i]
	}
}

// BinarySearch is O(log n). Unverifiable today: halving loops are not a
// recognized induction. Graduates with: geometric/halving-loop recognition
// (Phase 2 tripcount).
//
//bigo:max O(log n)
func BinarySearch(xs []int, target int) int { // want `cannot verify budget O\(log\(len\(xs\)\)\)`
	lo, hi := 0, len(xs)
	for lo < hi {
		mid := (lo + hi) / 2
		if xs[mid] < target {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(xs) && xs[lo] == target {
		return lo
	}
	return -1
}

// InsertionSort is O(n^2). Bounded since the loop-algebra slice: the inner
// countdown j > 0 starts at the guard-bounded i.
//
//bigo:max O(n^2)
func InsertionSort(xs []int) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j-1] > xs[j]; j-- {
			xs[j-1], xs[j] = xs[j], xs[j-1]
		}
	}
}

// SelectionSort is O(n^2). Bounded since the loop-algebra slice: the inner
// start i+1 has a provable constant lower bound.
//
//bigo:max O(n^2)
func SelectionSort(xs []int) {
	for i := 0; i < len(xs); i++ {
		m := i
		for j := i + 1; j < len(xs); j++ {
			if xs[j] < xs[m] {
				m = j
			}
		}
		xs[i], xs[m] = xs[m], xs[i]
	}
}
