package concurrent

func scan(ys []int) int {
	s := 0
	for i := 0; i < len(ys); i++ {
		s += ys[i]
	}
	return s
}

// Concurrency-dependent bounds are unverifiable in v1, even when the
// spawned callee is resolvable.

//bigo:max O(n)
func SpawnsWork(xs []int) { // want `cannot verify budget O\(len\(xs\)\)`
	for i := 0; i < len(xs); i++ {
		go scan(xs)
	}
}

// defer is sequential semantics: n deferred linear scans are quadratic work,
// and the engine must see it (this was the review's confirmed wrong bound).

//bigo:max O(n)
func DeferInLoop(xs []int) { // want `complexity O\(len\(xs\)\^2\) exceeds budget O\(len\(xs\)\)`
	for i := 0; i < len(xs); i++ {
		defer scan(xs)
	}
}

// A single defer outside any loop is plain O(callee) — must still verify.

//bigo:max O(n)
func SingleDefer(xs []int) int {
	defer scan(xs)
	return len(xs)
}

// Channel ops without goroutines stay bounded (send/recv are O(1) work).

//bigo:max O(n)
func ChannelFill(xs []int, ch chan int) {
	for i := 0; i < len(xs); i++ {
		ch <- xs[i]
	}
}
