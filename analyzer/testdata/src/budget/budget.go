package budget

//bigo:max O(n)
func LinearOK(xs []int, target int) int {
	for i := 0; i < len(xs); i++ {
		if xs[i] == target {
			return i
		}
	}
	return -1
}

//bigo:max O(n)
func QuadraticBad(xs []int) int { // want `complexity O\(len\(xs\)\^2\) exceeds budget O\(len\(xs\)\)`
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs); j++ {
			s += xs[i] * xs[j]
		}
	}
	return s
}

func other(int) int

//bigo:max O(n)
func Unresolved(xs []int) int { // want `cannot verify budget O\(len\(xs\)\)`
	s := 0
	for i := 0; i < len(xs); i++ {
		s += other(xs[i])
	}
	return s
}

//bigo:mx O(n)
func TypoVerb(xs []int) int { // want `invalid //bigo: directive`
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs); j++ {
			s += xs[i] * xs[j]
		}
	}
	return s
}

//bigo:max O(n^)
func BadExpr(xs []int) int { // want `invalid //bigo: directive`
	s := 0
	for i := 0; i < len(xs); i++ {
		s += xs[i]
	}
	return s
}

// Both verbs on one declaration: cost is what callers see, max still gates
// this body. Neither may silently vanish (finding S2).

//bigo:cost O(1)
//bigo:max O(n)
func CostAndMax(xs []int) int { // want `complexity O\(len\(xs\)\^2\) exceeds budget O\(len\(xs\)\)`
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs); j++ {
			s += xs[i] * xs[j]
		}
	}
	return s
}

// A repeated verb is a mistake, not a merge: report it and keep the first.

//bigo:max O(n)
//bigo:max O(n^2)
func DuplicateMax(xs []int) int { // want `duplicate //bigo:max directive`
	s := 0
	for i := 0; i < len(xs); i++ {
		s += xs[i]
	}
	return s
}

// cost and ignore both assert this function's summary to callers; asserting
// two different summaries is incoherent.

//bigo:cost O(1)
//bigo:ignore
func CostAndIgnore(x int) int { // want `//bigo:cost and //bigo:ignore are mutually exclusive`
	return x
}
