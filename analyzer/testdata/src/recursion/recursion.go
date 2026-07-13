// Package recursion is the corpus for recurrence solving. This PR pins the
// full soundness regression set at ⊤; later PRs graduate the solvable ones.
package recursion

//bigo:max O(n)
func Unguarded(n int) int { // want `cannot verify budget O\(n\)`
	return Unguarded(n - 1) // no base: may not terminate -> must stay ⊤
}

//bigo:max O(n)
func Growing(n int) int { // want `cannot verify budget O\(n\)`
	if n > 1000 {
		return 0
	}
	return Growing(n + 1) // argument grows -> ⊤
}

//bigo:max O(1)
func Fib(n int) int { // want `cannot verify budget O\(1\)`
	if n < 2 {
		return n
	}
	return Fib(n-1) + Fib(n-2) // a=2 subtractive -> exponential -> ⊤
}

//bigo:max O(n)
func SelfInLoop(xs []int) int { // want `cannot verify budget O\(len\(xs\)\)`
	s := 0
	for range xs {
		s += SelfInLoop(xs[1:]) // self-call under a size loop -> ⊤
	}
	return s
}
