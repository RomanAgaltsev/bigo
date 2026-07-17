// Package wrongpkg holds the deliberate WRONG fixture. The pin declares the
// "literature" bound O(n^2); the body is a single counted loop the engine
// bounds exactly at O(n). Emitted O(n) does not dominate the O(n^2) pin —
// from the oracle's view, inference just claimed a bound below ground truth,
// which is precisely a wrong bound. Proves the alarm fires.
package wrongpkg

// TooGood is mispinned on purpose. DO NOT FIX THE PIN.
//
//oracle:time O(n^2) where n=len(s)
//oracle:source deliberate WRONG fixture — proves the alarm fires (spec §6)
func TooGood(s []int) int {
	total := 0
	for i := 0; i < len(s); i++ {
		total += s[i]
	}
	return total
}
