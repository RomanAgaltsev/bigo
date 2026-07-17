// Package goodpkg is a known-outcome fixture for the oracle collector:
// LinearSum is exactly inferable (counted loop), Opaque is honestly ⊤
// (call through a function value).
package goodpkg

// LinearSum sums a slice: exactly O(n) time, O(1) space.
//
//oracle:time O(n) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source fixture — counted loop, engine R1 territory
func LinearSum(s []int) int {
	total := 0
	for i := 0; i < len(s); i++ {
		total += s[i]
	}
	return total
}

// Opaque calls a function value the engine cannot price: honestly ⊤.
//
//oracle:time O(1)
//oracle:source fixture — funcvalue call, expected top
func Opaque(f func()) {
	f()
}
