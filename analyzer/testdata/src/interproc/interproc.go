package interproc

import "sort"

func scan(ys []int, t int) int {
	for i := 0; i < len(ys); i++ {
		if ys[i] == t {
			return i
		}
	}
	return -1
}

//bigo:max O(n)
func CallsLinearOK(xs []int, t int) int { return scan(xs, t) }

//bigo:max O(n)
func LinearOverLinearBad(xs []int) int { // want `complexity O\(len\(xs\)\^2\) exceeds budget O\(len\(xs\)\)`
	s := 0
	for i := 0; i < len(xs); i++ {
		s += scan(xs, xs[i])
	}
	return s
}

//bigo:max O(n log n)
func SortOK(xs []int) { sort.Ints(xs) }

//bigo:max O(n)
func SortTooSlow(xs []int) { // want `complexity O\(len\(xs\) log\(len\(xs\)\)\) exceeds budget O\(len\(xs\)\)`
	sort.Ints(xs)
}
