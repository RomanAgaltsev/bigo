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
