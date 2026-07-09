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
