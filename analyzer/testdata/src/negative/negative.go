package negative

type Doer interface{ Do(int) int }

//bigo:max O(n)
func InterfaceCall(xs []int, d Doer) int { // want `cannot verify budget O\(len\(xs\)\)`
	s := 0
	for i := 0; i < len(xs); i++ {
		s += d.Do(xs[i])
	}
	return s
}

//bigo:max O(n)
func ClosureCall(xs []int, f func(int) int) int { // want `cannot verify budget O\(len\(xs\)\)`
	s := 0
	for i := 0; i < len(xs); i++ {
		s += f(xs[i])
	}
	return s
}

// A goroutine-bearing function must not crash the analyzer. No budget, so no
// diagnostic is expected — this proves robustness, not a bound.
func Concurrent(xs []int) {
	ch := make(chan int, len(xs))
	for i := 0; i < len(xs); i++ {
		go func(v int) { ch <- v }(xs[i])
	}
}
