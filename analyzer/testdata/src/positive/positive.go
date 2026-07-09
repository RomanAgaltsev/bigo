package positive

//bigo:max O(1)
func Constant(x int) int { return x * 2 }

//bigo:max O(n)
func LinearScan(xs []int, t int) int {
	for i := 0; i < len(xs); i++ {
		if xs[i] == t {
			return i
		}
	}
	return -1
}

//bigo:max O(n^2)
func BubbleSort(xs []int) {
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs); j++ {
			if xs[i] < xs[j] {
				xs[i], xs[j] = xs[j], xs[i]
			}
		}
	}
}

//bigo:max O(n) where n=k
func CountUp(k int) int {
	s := 0
	for i := 0; i < k; i++ {
		s += i
	}
	return s
}
