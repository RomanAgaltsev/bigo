// Package mutual is the corpus for two-function recursion cycles. This PR pins
// the full spec-§6 regression set at ⊤; PR2 graduates the solvable pairs.
package mutual

//bigo:max O(n)
func EvenStep(n int) bool { // graduates: composed Sub(2), guarded -> O(n)
	if n <= 0 {
		return true
	}
	return OddStep(n - 1)
}

func OddStep(n int) bool {
	if n <= 0 {
		return false
	}
	return EvenStep(n - 1)
}

//bigo:max O(n)
func SumViaHelper(xs []int) int { // graduates: Sub(1) via helper -> O(len(xs))
	if len(xs) == 0 {
		return 0
	}
	return xs[0] + helperStep(xs)
}

func helperStep(xs []int) int { return SumViaHelper(xs[1:]) }

//bigo:max O(n log n)
func WalkSum(xs []int) int { // graduates: 2·T(n/2)+O(n) across the pair -> Master case 2
	if len(xs) < 2 {
		return len(xs)
	}
	return walkParts(xs)
}

func walkParts(xs []int) int {
	s := 0
	for _, v := range xs { // O(len(xs)) level work in the helper
		s += v
	}
	m := len(xs) / 2
	return s + WalkSum(xs[:m]) + WalkSum(xs[m:])
}

//bigo:max O(n)
func GrowA(n int) int { // want `cannot verify budget O\(n\)`
	if n <= 0 {
		return 0
	}
	return GrowB(n + 1) // growing edge -> ⊤ forever
}

func GrowB(n int) int { return GrowA(n - 2) }

//bigo:max O(n)
func MultiA(n int) int { // want `cannot verify budget O\(n\)`
	if n <= 0 {
		return 0
	}
	return MultiA(n-1) + MultiB(n-1) // member self-recurses: multi-cycle -> ⊤
}

func MultiB(n int) int {
	if n <= 0 {
		return 0
	}
	return MultiA(n - 1)
}

//bigo:max O(log n)
func DivGEZeroA(n int) int { // want `cannot verify budget O\(log\(n\)\)`
	if n >= 0 {
		return DivGEZeroB(n) // F1 class through the mutual path: >=0 is no floor
	}
	return 0
}

func DivGEZeroB(n int) int { return DivGEZeroA(n / 2) }

//bigo:max O(log n)
func NoBaseSliceA(xs []int) int { // want `cannot verify budget O\(log\(len\(xs\)\)\)`
	return NoBaseSliceB(xs)
}

func NoBaseSliceB(xs []int) int { return NoBaseSliceA(xs[:len(xs)/2]) }

//bigo:max O(n)
func ParseExpr(n int) int { // want `cannot verify budget O\(n\)`
	if n <= 0 {
		return 0
	}
	return ParseTerm(n - 1) // 3-cycle (parser shape): out of scope, ⊤ forever in v1
}

func ParseTerm(n int) int {
	if n <= 0 {
		return 0
	}
	return ParseFactor(n - 1)
}

func ParseFactor(n int) int {
	if n <= 0 {
		return 0
	}
	return ParseExpr(n - 1)
}

// funcValImpl is assigned in init (not a package-var initializer) to avoid a
// types init cycle while keeping the FuncValA→FuncValB edge a dynamic call: a
// local `f := FuncValB` is constant-folded by SSA into a static call, which
// would (soundly) be detected as a real 2-cycle. Routing through a func-typed
// variable keeps StaticCallee nil, so it stays ⊤ as the func-value non-goal
// intends.
var funcValImpl func(int) int

func init() { funcValImpl = FuncValB }

//bigo:max O(n)
func FuncValA(n int) int { // want `cannot verify budget O\(n\)`
	if n <= 0 {
		return 0
	}
	return funcValImpl(n - 1) // dynamic edge: not a static 2-cycle -> ⊤
}

func FuncValB(n int) int { return FuncValA(n - 1) }
