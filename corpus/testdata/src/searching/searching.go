// Package searching is the canonical-corpus searching family.
package searching

// LinearSearch returns the index of x in s, or -1.
//
//oracle:time O(n) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source en.wikipedia.org/wiki/Linear_search
func LinearSearch(s []int, x int) int {
	for i := 0; i < len(s); i++ {
		if s[i] == x {
			return i
		}
	}
	return -1
}

// BinarySearch returns an index of x in sorted s, or -1. Iterative.
//
//oracle:time O(log n) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source CLRS ex. 2.3-5; en.wikipedia.org/wiki/Binary_search_algorithm
func BinarySearch(s []int, x int) int {
	lo, hi := 0, len(s)-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		switch {
		case s[mid] == x:
			return mid
		case s[mid] < x:
			lo = mid + 1
		default:
			hi = mid - 1
		}
	}
	return -1
}

// BinarySearchRec reports whether x is in sorted s. Recursive on halves; the
// recursion depth is the log-factor stack pin.
//
//oracle:time O(log n) where n=len(s)
//oracle:space O(log n) where n=len(s)
//oracle:source CLRS ex. 2.3-5 (recursive form)
func BinarySearchRec(s []int, x int) bool {
	if len(s) == 0 {
		return false
	}
	mid := len(s) / 2
	switch {
	case s[mid] == x:
		return true
	case s[mid] < x:
		return BinarySearchRec(s[mid+1:], x)
	default:
		return BinarySearchRec(s[:mid], x)
	}
}

// FirstOccurrence returns the smallest index of x in sorted s, or -1.
//
//oracle:time O(log n) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source en.wikipedia.org/wiki/Binary_search_algorithm (leftmost variant)
func FirstOccurrence(s []int, x int) int {
	lo, hi, ans := 0, len(s)-1, -1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if s[mid] >= x {
			if s[mid] == x {
				ans = mid
			}
			hi = mid - 1
		} else {
			lo = mid + 1
		}
	}
	return ans
}

// SearchRotated finds x in a sorted array rotated at an unknown pivot, or -1.
//
//oracle:time O(log n) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source www.geeksforgeeks.org/search-an-element-in-a-sorted-and-pivoted-array/ (bound reference)
func SearchRotated(s []int, x int) int {
	lo, hi := 0, len(s)-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if s[mid] == x {
			return mid
		}
		if s[lo] <= s[mid] {
			if s[lo] <= x && x < s[mid] {
				hi = mid - 1
			} else {
				lo = mid + 1
			}
		} else {
			if s[mid] < x && x <= s[hi] {
				lo = mid + 1
			} else {
				hi = mid - 1
			}
		}
	}
	return -1
}
