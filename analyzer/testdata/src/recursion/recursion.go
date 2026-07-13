// Package recursion is the corpus for recurrence solving. The soundness
// regression set (unguarded, growing, exponential, self-in-loop) stays pinned
// at ⊤; this PR graduates the subtractive linear recurrences. Later PRs
// graduate the divide-and-conquer families.
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

//bigo:max O(n)
func SumSlice(xs []int) int { // graduates: T(n)=T(n-1)+O(1) -> O(len(xs))
	if len(xs) == 0 {
		return 0
	}
	return xs[0] + SumSlice(xs[1:])
}

//bigo:max O(n)
func LinearSearchRec(xs []int, t int) int { // O(len(xs))
	if len(xs) == 0 {
		return -1
	}
	if xs[0] == t {
		return 0
	}
	return LinearSearchRec(xs[1:], t)
}

//bigo:max O(n)
func CountdownWork(n int) int { // guarded integer, T(n)=T(n-1)+O(1) -> O(n)
	if n <= 0 {
		return 0
	}
	return 1 + CountdownWork(n-1)
}
