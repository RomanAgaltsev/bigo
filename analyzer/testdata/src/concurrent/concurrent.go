package concurrent

import "sync"

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

// Mutex operations are O(1) work on the same argument as the channel ops above:
// blocking under contention is wall-clock, not work. A mutex-guarded scan
// verifies (issue #46 — this shape caused 42 ⊤ verdicts in chaotic alone).

//bigo:max O(n) where n = len(xs)
func MutexGuarded(mu *sync.Mutex, xs []int) int {
	mu.Lock()
	defer mu.Unlock()
	return scan(xs)
}

//bigo:max O(n) where n = len(xs)
func RWMutexGuarded(mu *sync.RWMutex, xs []int) int {
	mu.RLock()
	defer mu.RUnlock()
	return scan(xs)
}

// sync.Once.Do(f) costs cost(f), not O(1). It is deliberately out of the cost
// table: an O(1) entry would under-approximate the call into a false Within.
// This pins that it stays honestly unverifiable.

//bigo:max O(n) where n = len(xs)
func OnceDo(once *sync.Once, xs []int) { // want `cannot verify budget O\(len\(xs\)\)`
	once.Do(func() { scan(xs) })
}
