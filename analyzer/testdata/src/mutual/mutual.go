// Package mutual is the corpus for two-function recursion cycles. This PR pins
// the full spec-§6 regression set at ⊤; PR2 graduates the solvable pairs.
package mutual

//bigo:max O(n)
func EvenStep(n int) bool { // want `cannot verify budget O\(n\)`
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

//bigo:max O(n)
func FuncValA(n int) int { // want `cannot verify budget O\(n\)`
	if n <= 0 {
		return 0
	}
	f := FuncValB
	return f(n - 1) // dynamic edge: not a static 2-cycle -> ⊤
}

func FuncValB(n int) int { return FuncValA(n - 1) }
