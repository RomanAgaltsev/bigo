// Package brokensearch is the kata corpus's sprint-3 first final: binary search
// over a sorted array that has been rotated at an unknown pivot.
//
// Reduced from the submitted solution. The author's claim scores only the
// search — "в качестве решения на Контесте принимается только реализация
// поиска" — so the I/O scaffolding it excludes is not in this repository.
package brokensearch

// BrokenSearch returns the index of k in a rotated sorted array, or -1.
//
// Each iteration discards half the range: whichever side of mid is in sorted
// order is testable directly, and the other half is where the rotation lives.
//
//oracle:time O(log n) where n=len(arr)
//oracle:space O(1) where n=len(arr)
//oracle:source ya_algo sprint 3 final 1; author's claim "В решении используется только бинарный поиск, который работает за O(log n)", auxiliary space "O(1) - можно сказать, что почти не используется"
func BrokenSearch(arr []int, k int) int {
	left, right := 0, len(arr)-1
	for left <= right {
		mid := left + (right-left)/2
		if arr[mid] == k {
			return mid
		}
		if arr[left] <= arr[mid] {
			if k < arr[mid] && k >= arr[left] {
				right = mid - 1
			} else {
				left = mid + 1
			}
		} else {
			if k <= arr[right] && k > arr[mid] {
				left = mid + 1
			} else {
				right = mid - 1
			}
		}
	}
	return -1
}
