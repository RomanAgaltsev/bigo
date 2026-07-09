package multivar

//bigo:max O(n*m) where n=len(a), m=len(b)
func Pairs(a, b []int) int {
	s := 0
	for i := 0; i < len(a); i++ {
		for j := 0; j < len(b); j++ {
			s += a[i] * b[j]
		}
	}
	return s
}

// O(len(a) len(b)) strictly dominates the sole budget term, so the violation is
// provable even though the bound is multi-variable.
//
//bigo:max O(n) where n=len(a), m=len(b)
func PairsOverBudget(a, b []int) int { // want `complexity O\(len\(a\) len\(b\)\) exceeds budget O\(len\(a\)\)`
	s := 0
	for i := 0; i < len(a); i++ {
		for j := 0; j < len(b); j++ {
			s += a[i] * b[j]
		}
	}
	return s
}

// Neither O(len(a) len(b)) nor O(len(a)^2) dominates the other, so the budget is
// neither provably met nor provably violated: incomparable -> unverifiable.
//
//bigo:max O(n^2) where n=len(a), m=len(b)
func PairsIncomparable(a, b []int) int { // want `cannot verify budget O\(len\(a\)\^2\): inferred O\(len\(a\) len\(b\)\) is not comparable`
	s := 0
	for i := 0; i < len(a); i++ {
		for j := 0; j < len(b); j++ {
			s += a[i] * b[j]
		}
	}
	return s
}
