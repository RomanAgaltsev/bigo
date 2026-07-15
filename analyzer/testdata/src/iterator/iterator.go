// Package iterator is the corpus for range-over-func (iterator) costs.
//
// SSA shape (Go 1.26, recorded per plan Task 7): `for v := range seq` lowers to
//
//	t1 = <seq producer>(args)   // the iter.Seq value
//	t3 = make closure <body>    // the loop body, as a yield closure
//	t4 = t1(t3)                 // dynamic call of the seq with the body
//
// The producer call is O(1) (lazy construction); the iteration cost is charged
// at t4 by rangeFuncCost, which resolves t1 against the curated producer table.
package iterator

import (
	"maps"
	"slices"
)

//bigo:max O(n)
func SumValues(s []int) int { // graduates: O(len(s))
	total := 0
	for v := range slices.Values(s) {
		total += v
	}
	return total
}

//bigo:max O(n)
func SumBreak(s []int) int { // graduates: O(len(s)) upper bound even with early break
	total := 0
	for v := range slices.Values(s) {
		total += v
		if total > 100 {
			break
		}
	}
	return total
}

//bigo:max O(n)
func CountKeys(m map[int]int) int { // graduates: O(len(m))
	c := 0
	for range maps.Keys(m) {
		c++
	}
	return c
}

func recSeq(n int) func(yield func(int) bool) { // recursive user iterator producer
	return func(yield func(int) bool) {
		if n <= 0 {
			return
		}
		if !yield(n) {
			return
		}
		for v := range recSeq(n - 1) {
			if !yield(v) {
				return
			}
		}
	}
}

//bigo:max O(n)
func SumRecursive(n int) int { // want `cannot verify budget`
	total := 0
	for v := range recSeq(n) { // pin 5: recursive/user iterator producer -> ⊤
		total += v
	}
	return total
}

//bigo:max O(n)
func FromChannel(ch chan func(yield func(int) bool), s []int) int { // want `cannot verify budget`
	_ = s
	total := 0
	for v := range <-ch { // seq from a channel is not a curated producer -> ⊤
		total += v
	}
	return total
}
