// Package recursion is the corpus for recurrence solving. The soundness
// regression set (unguarded, growing, exponential, self-in-loop) stays pinned
// at ⊤; the subtractive linear recurrences and the divide-and-conquer families
// (Master theorem) graduate.
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
func Fib(n int) int { // want `cannot verify budget O\(1\)` `smell\(SM8\): provably exponential recursion`
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

//bigo:max O(log n)
func DivGEZero(n int) int { // want `cannot verify budget O\(log\(n\)\)`
	if n >= 0 {
		return DivGEZero(n / 2) // 0/2==0 is a fixed point: infinite for all n>=0 -> ⊤
	}
	return 0
}

//bigo:max O(log n)
func DivNegFloor(n int) int { // want `cannot verify budget O\(log\(n\)\)`
	if n > -5 {
		return DivNegFloor(n / 2) // negative floor: still infinite at n=0 -> ⊤
	}
	return 0
}

//bigo:max O(log n)
func DivGuardedPositive(n int) int { // n>0 => n>=1: the n<=0 base is reached -> O(log n)
	if n > 0 {
		return DivGuardedPositive(n / 2)
	}
	return 0
}

//bigo:max O(log n)
func FastPow(n int) int { // n==0 base: magnitude halves to 0 -> O(log n) (power-by-squaring)
	if n == 0 {
		return 1
	}
	return FastPow(n / 2)
}

//bigo:max O(log n)
func DivSliceNoBase(xs []int) int { // want `cannot verify budget O\(log\(len\(xs\)\)\)`
	m := len(xs) / 2
	return DivSliceNoBase(xs[:m]) // xs[:0] stays empty, no base guard -> ⊤
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

//bigo:max O(log n)
func BinarySearchRec(xs []int, t int) int { // T(n)=T(n/2)+O(1) -> O(log(len(xs)))
	if len(xs) == 0 {
		return -1
	}
	m := len(xs) / 2
	switch {
	case xs[m] == t:
		return m
	case xs[m] < t:
		return BinarySearchRec(xs[m+1:], t)
	default:
		return BinarySearchRec(xs[:m], t)
	}
}

//bigo:max O(n)
func TreeSum(xs []int) int { // Master case 1: 2T(n/2)+O(1) -> O(len(xs))
	if len(xs) < 2 {
		if len(xs) == 0 {
			return 0
		}
		return xs[0]
	}
	m := len(xs) / 2
	return TreeSum(xs[:m]) + TreeSum(xs[m:])
}

//bigo:max O(n log n)
func ScanHalve(xs []int) int { // Master case 2: 2T(n/2)+O(n) -> O(len(xs) log(len(xs)))
	s := 0
	for _, v := range xs { // O(len(xs)) per-level scan of the parameter itself
		s += v
	}
	if len(xs) < 2 {
		return s
	}
	m := len(xs) / 2
	return s + ScanHalve(xs[:m]) + ScanHalve(xs[m:])
}

// Merge sort (2T(n/2)+O(n) via merge(l, r)) stays ⊤: the O(len(xs)) per-level
// work is the merge of the two recursion RESULTS, so it is O(len(l)+len(r)),
// and tying len(l)+len(r) back to len(xs) needs relational length tracking
// (result length = input length; reslice partition) the engine does not model.
// The solver itself handles 2T(n/2)+O(n) — see TestSolveMaster — so ScanHalve,
// whose per-level work scans the parameter directly, graduates to the same
// O(n log n) bound.

//bigo:max O(n log n)
func Unbalanced(xs []int) int { // Akra–Bazzi: T(n/2)+2T(n/4)+O(n), p=1 -> O(len(xs) log(len(xs)))
	s := 0
	for _, v := range xs { // O(len(xs)) per-level scan of the parameter
		s += v
	}
	if len(xs) < 4 {
		return s
	}
	half := len(xs) / 2
	q := len(xs) / 4
	return s + Unbalanced(xs[:half]) + Unbalanced(xs[:q]) + Unbalanced(xs[:q])
}
