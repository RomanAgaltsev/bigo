// Package dandc is the canonical-corpus divide-and-conquer family.
package dandc

// MaxSubarrayDC returns the maximum subarray sum by divide and conquer —
// deliberately the O(n log n) textbook form, not Kadane.
//
//oracle:time O(n log n) where n=len(s)
//oracle:space O(log n) where n=len(s)
//oracle:source CLRS §4.1; en.wikipedia.org/wiki/Maximum_subarray_problem
func MaxSubarrayDC(s []int) int {
	if len(s) == 0 {
		return 0
	}
	if len(s) == 1 {
		return s[0]
	}
	mid := len(s) / 2
	left := MaxSubarrayDC(s[:mid])
	right := MaxSubarrayDC(s[mid:])
	sum, bestL := 0, s[mid-1]
	for i := mid - 1; i >= 0; i-- {
		sum += s[i]
		if sum > bestL {
			bestL = sum
		}
	}
	sum, bestR := 0, s[mid]
	for i := mid; i < len(s); i++ {
		sum += s[i]
		if sum > bestR {
			bestR = sum
		}
	}
	best := bestL + bestR
	if left > best {
		best = left
	}
	if right > best {
		best = right
	}
	return best
}

// CountInversions counts pairs i<j with s[i]>s[j], mergesort-based. Literature
// peak auxiliary space is O(n); bigo's Heap channel counts total allocation,
// so loose space is expected and sound.
//
//oracle:time O(n log n) where n=len(s)
//oracle:space O(n) where n=len(s)
//oracle:source CLRS problem 2-4; en.wikipedia.org/wiki/Counting_inversions
func CountInversions(s []int) int {
	if len(s) <= 1 {
		return 0
	}
	mid := len(s) / 2
	left := append([]int(nil), s[:mid]...)
	right := append([]int(nil), s[mid:]...)
	inv := CountInversions(left) + CountInversions(right)
	i, j, k := 0, 0, 0
	for i < len(left) && j < len(right) {
		if left[i] <= right[j] {
			s[k] = left[i]
			i++
		} else {
			s[k] = right[j]
			j++
			inv += len(left) - i
		}
		k++
	}
	for i < len(left) {
		s[k] = left[i]
		i++
		k++
	}
	for j < len(right) {
		s[k] = right[j]
		j++
		k++
	}
	return inv
}

// PeakElement returns an index whose element is not smaller than its
// neighbors, by iterative bisection.
//
//oracle:time O(log n) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source jeffe.cs.illinois.edu/teaching/algorithms/ (recursion notes); www.geeksforgeeks.org/find-a-peak-in-a-given-array/ (bound reference)
func PeakElement(s []int) int {
	lo, hi := 0, len(s)-1
	for lo < hi {
		mid := lo + (hi-lo)/2
		if s[mid] < s[mid+1] {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// PowerDC computes x^b (b ≥ 0) by recursive squaring.
//
//oracle:time O(log b) where b=b
//oracle:space O(log b) where b=b
//oracle:source CLRS §31.6 (repeated squaring); en.wikipedia.org/wiki/Exponentiation_by_squaring
func PowerDC(x, b int) int {
	if b == 0 {
		return 1
	}
	half := PowerDC(x, b/2)
	if b%2 == 0 {
		return half * half
	}
	return half * half * x
}

// MaxMinDC returns the minimum and maximum by divide and conquer.
//
//oracle:time O(n) where n=len(s)
//oracle:space O(log n) where n=len(s)
//oracle:source www.geeksforgeeks.org/maximum-and-minimum-in-an-array/ (tournament method, bound reference)
func MaxMinDC(s []int) (int, int) {
	if len(s) == 1 {
		return s[0], s[0]
	}
	if len(s) == 2 {
		if s[0] < s[1] {
			return s[0], s[1]
		}
		return s[1], s[0]
	}
	mid := len(s) / 2
	lmin, lmax := MaxMinDC(s[:mid])
	rmin, rmax := MaxMinDC(s[mid:])
	if rmin < lmin {
		lmin = rmin
	}
	if rmax > lmax {
		lmax = rmax
	}
	return lmin, lmax
}

// MajorityDC returns a majority-candidate element by divide and conquer with
// linear-time counting per level.
//
//oracle:time O(n log n) where n=len(s)
//oracle:space O(log n) where n=len(s)
//oracle:source CLRS-style D&C; en.wikipedia.org/wiki/Boyer%E2%80%93Moore_majority_vote_algorithm (D&C alternative, bound reference)
func MajorityDC(s []int) int {
	if len(s) == 1 {
		return s[0]
	}
	mid := len(s) / 2
	left := MajorityDC(s[:mid])
	right := MajorityDC(s[mid:])
	if left == right {
		return left
	}
	countL, countR := 0, 0
	for _, v := range s {
		if v == left {
			countL++
		} else if v == right {
			countR++
		}
	}
	if countL > countR {
		return left
	}
	return right
}
