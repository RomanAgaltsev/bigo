// Package sorting is the canonical-corpus sorting family. Literature-pinned
// worst-case bounds; no //bigo: directives — the engine infers unaided.
package sorting

// InsertionSort sorts in place by insertion.
//
//oracle:time O(n^2) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source CLRS §2.1; en.wikipedia.org/wiki/Insertion_sort (worst case)
func InsertionSort(s []int) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}

// SelectionSort sorts in place by repeated minimum selection.
//
//oracle:time O(n^2) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source CLRS ex. 2.2-2; en.wikipedia.org/wiki/Selection_sort
func SelectionSort(s []int) {
	for i := 0; i < len(s); i++ {
		min := i
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[min] {
				min = j
			}
		}
		s[i], s[min] = s[min], s[i]
	}
}

// BubbleSort sorts in place by adjacent swaps.
//
//oracle:time O(n^2) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source CLRS problem 2-2; en.wikipedia.org/wiki/Bubble_sort
func BubbleSort(s []int) {
	for i := 0; i < len(s); i++ {
		for j := 0; j+1 < len(s)-i; j++ {
			if s[j] > s[j+1] {
				s[j], s[j+1] = s[j+1], s[j]
			}
		}
	}
}

// MergeSort returns a sorted copy, top-down. Literature auxiliary space is
// O(n) peak live; bigo's Heap channel counts total allocation, so a loose
// space verdict here is expected and sound.
//
//oracle:time O(n log n) where n=len(s)
//oracle:space O(n) where n=len(s)
//oracle:source CLRS §2.3.1; en.wikipedia.org/wiki/Merge_sort
func MergeSort(s []int) []int {
	if len(s) <= 1 {
		out := make([]int, len(s))
		copy(out, s)
		return out
	}
	mid := len(s) / 2
	left, right := MergeSort(s[:mid]), MergeSort(s[mid:])
	out := make([]int, 0, len(s))
	i, j := 0, 0
	for i < len(left) && j < len(right) {
		if left[i] <= right[j] {
			out = append(out, left[i])
			i++
		} else {
			out = append(out, right[j])
			j++
		}
	}
	out = append(out, left[i:]...)
	return append(out, right[j:]...)
}

// QuickSort sorts in place, Lomuto partition, last-element pivot. Worst case
// (sorted input) is quadratic time and linear recursion depth.
//
//oracle:time O(n^2) where n=len(s)
//oracle:space O(n) where n=len(s)
//oracle:source CLRS §7; en.wikipedia.org/wiki/Quicksort (worst case)
func QuickSort(s []int) {
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
	QuickSort(s[:i])
	QuickSort(s[i+1:])
}

// HeapSort sorts in place: build a max-heap, then repeatedly extract.
//
//oracle:time O(n log n) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source CLRS §6.4; en.wikipedia.org/wiki/Heapsort
func HeapSort(s []int) {
	for i := len(s)/2 - 1; i >= 0; i-- {
		siftDown(s, i, len(s))
	}
	for end := len(s) - 1; end > 0; end-- {
		s[0], s[end] = s[end], s[0]
		siftDown(s, 0, end)
	}
}

func siftDown(s []int, root, end int) {
	for {
		child := 2*root + 1
		if child >= end {
			return
		}
		if child+1 < end && s[child+1] > s[child] {
			child++
		}
		if s[root] >= s[child] {
			return
		}
		s[root], s[child] = s[child], s[root]
		root = child
	}
}

// CountingSort returns a sorted copy of s whose values lie in [0, k).
//
//oracle:time O(n + k) where n=len(s), k=k
//oracle:space O(n + k) where n=len(s), k=k
//oracle:source CLRS §8.2; en.wikipedia.org/wiki/Counting_sort
func CountingSort(s []int, k int) []int {
	counts := make([]int, k)
	for _, v := range s {
		counts[v]++
	}
	out := make([]int, 0, len(s))
	for v := 0; v < k; v++ {
		for c := 0; c < counts[v]; c++ {
			out = append(out, v)
		}
	}
	return out
}

// RadixSortLSD sorts non-negative 32-bit ints, base 256, four passes. For
// fixed-width keys the digit count is a constant, so the literature bound
// O(d(n+k)) collapses to O(n): d=4, k=256.
//
//oracle:time O(n) where n=len(s)
//oracle:space O(n) where n=len(s)
//oracle:source CLRS §8.3 (fixed d, k); en.wikipedia.org/wiki/Radix_sort
func RadixSortLSD(s []uint32) {
	buf := make([]uint32, len(s))
	for shift := 0; shift < 32; shift += 8 {
		var counts [257]int
		for _, v := range s {
			counts[byte(v>>shift)+1]++
		}
		for i := 1; i < 257; i++ {
			counts[i] += counts[i-1]
		}
		for _, v := range s {
			buf[counts[byte(v>>shift)]] = v
			counts[byte(v>>shift)]++
		}
		copy(s, buf)
	}
}

// ShellSort sorts in place with Shell's original n/2^j gap sequence, whose
// worst case is quadratic.
//
//oracle:time O(n^2) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source en.wikipedia.org/wiki/Shellsort (Shell's sequence, worst case)
func ShellSort(s []int) {
	for gap := len(s) / 2; gap > 0; gap /= 2 {
		for i := gap; i < len(s); i++ {
			key := s[i]
			j := i - gap
			for j >= 0 && s[j] > key {
				s[j+gap] = s[j]
				j -= gap
			}
			s[j+gap] = key
		}
	}
}

// BucketSort sorts values in [0,1) into n buckets, insertion-sorting each.
// Worst case (all values in one bucket) is quadratic.
//
//oracle:time O(n^2) where n=len(s)
//oracle:space O(n) where n=len(s)
//oracle:source CLRS §8.4 (worst case); en.wikipedia.org/wiki/Bucket_sort
func BucketSort(s []float64) {
	n := len(s)
	if n == 0 {
		return
	}
	buckets := make([][]float64, n)
	for _, v := range s {
		i := int(v * float64(n))
		buckets[i] = append(buckets[i], v)
	}
	pos := 0
	for _, b := range buckets {
		for i := 1; i < len(b); i++ {
			key := b[i]
			j := i - 1
			for j >= 0 && b[j] > key {
				b[j+1] = b[j]
				j--
			}
			b[j+1] = key
		}
		for _, v := range b {
			s[pos] = v
			pos++
		}
	}
}
