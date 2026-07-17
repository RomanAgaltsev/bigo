// Package matrix is the canonical-corpus matrix family. All matrices are
// square n×n, n = len(m) — the mapping every pin's where-clause uses.
package matrix

// Mul returns a×b for square n×n matrices.
//
//oracle:time O(n^3) where n=len(a)
//oracle:space O(n^2) where n=len(a)
//oracle:source CLRS §4.2 (naive); en.wikipedia.org/wiki/Matrix_multiplication_algorithm
func Mul(a, b [][]int) [][]int {
	n := len(a)
	out := make([][]int, n)
	for i := 0; i < n; i++ {
		out[i] = make([]int, n)
		for j := 0; j < n; j++ {
			sum := 0
			for k := 0; k < n; k++ {
				sum += a[i][k] * b[k][j]
			}
			out[i][j] = sum
		}
	}
	return out
}

// TransposeInPlace transposes a square matrix in place.
//
//oracle:time O(n^2) where n=len(m)
//oracle:space O(1) where n=len(m)
//oracle:source en.wikipedia.org/wiki/Transpose
func TransposeInPlace(m [][]int) {
	for i := 0; i < len(m); i++ {
		for j := i + 1; j < len(m); j++ {
			m[i][j], m[j][i] = m[j][i], m[i][j]
		}
	}
}

// SpiralOrder returns the clockwise spiral traversal of a square matrix.
//
//oracle:time O(n^2) where n=len(m)
//oracle:space O(n^2) where n=len(m)
//oracle:source www.geeksforgeeks.org/print-a-given-matrix-in-spiral-form/ (bound reference)
func SpiralOrder(m [][]int) []int {
	n := len(m)
	out := make([]int, 0, n*n)
	top, bottom, left, right := 0, n-1, 0, n-1
	for top <= bottom && left <= right {
		for j := left; j <= right; j++ {
			out = append(out, m[top][j])
		}
		top++
		for i := top; i <= bottom; i++ {
			out = append(out, m[i][right])
		}
		right--
		if top <= bottom {
			for j := right; j >= left; j-- {
				out = append(out, m[bottom][j])
			}
			bottom--
		}
		if left <= right {
			for i := bottom; i >= top; i-- {
				out = append(out, m[i][left])
			}
			left++
		}
	}
	return out
}

// SearchSorted reports whether x is in a row-and-column-sorted square matrix
// (staircase search from the top-right corner).
//
//oracle:time O(n) where n=len(m)
//oracle:space O(1) where n=len(m)
//oracle:source www.geeksforgeeks.org/search-in-row-wise-and-column-wise-sorted-matrix/ (bound reference)
func SearchSorted(m [][]int, x int) bool {
	i, j := 0, len(m)-1
	for i < len(m) && j >= 0 {
		switch {
		case m[i][j] == x:
			return true
		case m[i][j] > x:
			j--
		default:
			i++
		}
	}
	return false
}

// Rotate90 rotates a square matrix 90° clockwise in place
// (transpose + row reversal).
//
//oracle:time O(n^2) where n=len(m)
//oracle:space O(1) where n=len(m)
//oracle:source www.geeksforgeeks.org/rotate-a-matrix-by-90-degree-in-clockwise-direction/ (bound reference)
func Rotate90(m [][]int) {
	n := len(m)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			m[i][j], m[j][i] = m[j][i], m[i][j]
		}
	}
	for i := 0; i < n; i++ {
		for l, r := 0, n-1; l < r; l, r = l+1, r-1 {
			m[i][l], m[i][r] = m[i][r], m[i][l]
		}
	}
}
