// Package funcvalue is the corpus for parametric (function-valued) costs.
package funcvalue

func apply(f func(int) int, x int) int { return f(x) } // parametric helper

func each(xs []int, f func(int)) { // PerParam[f] = O(len(xs))
	for _, v := range xs {
		f(v)
	}
}

func inc(x int) int { return x + 1 }

func noop(int) {}

//bigo:max O(n)
func EachConst(xs []int) { // graduates: O(len(xs)) × O(1)
	each(xs, noop)
}

//bigo:max O(1)
func ApplyOnce(x int) int { // graduates: one O(1) invocation
	return apply(inc, x)
}

var sink func(int)

//bigo:max O(n)
func StoredEscapes(xs []int, f func(int)) { // want `cannot verify budget O\(len\(xs\)\)`
	sink = f // pin 6: stored then called -> count ⊤
	for _, v := range xs {
		f(v)
	}
}

func opaque(f func(int)) // bodyless: pin 7's unknown callee

//bigo:max O(n)
func ForwardedToUnknown(xs []int, f func(int)) { // want `cannot verify budget O\(len\(xs\)\)`
	opaque(f)
	for _, v := range xs {
		f(v)
	}
}

//bigo:max O(n)
func GoInvoked(xs []int, f func(int)) { // want `cannot verify budget O\(len\(xs\)\)`
	go f(0) // pin 4: goroutine launch is ⊤ before counting matters
	for _, v := range xs {
		f(v)
	}
}

var fieldHeld struct{ cb func(int) }

//bigo:max O(n)
func FromStructField(xs []int) { // want `cannot verify budget O\(len\(xs\)\)`
	for _, v := range xs {
		fieldHeld.cb(v) // pin 1: func value from a struct field -> ⊤
	}
}

//bigo:max O(n)
func EachClosureCapturingConst(xs []int, base int) { // graduates: O(1) closure capturing an int
	each(xs, func(v int) { _ = v + base })
}

//bigo:max O(n)
func EachClosureSelfCapture(xs []int) { // graduates: O(1) closure capturing the iterated slice
	each(xs, func(i int) { _ = xs[0] + i }) // read-only capture -> spill size recovered
}

func makeCB() func(int) { return func(int) {} }

//bigo:max O(n)
func EachReturnedClosure(xs []int) { // want `cannot verify budget O\(len\(xs\)\)`
	each(xs, makeCB()) // pin 2: value from a call is not an in-scope MakeClosure -> ⊤
}

//bigo:max O(n)
func EachMutatedCapture(xs []int) { // want `cannot verify budget O\(len\(xs\)\)`
	f := func(i int) { _ = xs[0] + i }
	xs = append(xs, 1) // pin 3: capture reassigned -> second store -> spill refuses
	each(xs, f)
}

//bigo:max O(n)
func EachCaptureSizedDeferred(xs, ys []int) { // want `cannot verify budget`
	each(ys, func(int) { // capture-sized closure body: product pricing is deferred -> ⊤
		s := 0
		for _, v := range xs {
			s += v
		}
		_ = s
	})
}
