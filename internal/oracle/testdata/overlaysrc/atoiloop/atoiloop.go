// Package atoiloop is the CollectWith overlay probe: bounded under the kata
// cost model, top without it. It is not part of any shipped corpus.
package atoiloop

import "strconv"

// SumInts parses each element and sums it.
//
//oracle:time O(n) where n=len(ss)
//oracle:source kata cost model: one conversion of one token is one element operation
func SumInts(ss []string) int {
	total := 0
	for i := 0; i < len(ss); i++ {
		v, _ := strconv.Atoi(ss[i])
		total += v
	}
	return total
}
