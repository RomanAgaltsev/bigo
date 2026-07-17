package costignore

// The trust loop: an unverifiable callee is resolved by asserting
// its cost or trusting it — the budget on the caller then verifies silently.

//bigo:cost O(1)
func opaque(x int) int

//bigo:max O(n)
func UsesCost(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		s += opaque(xs[i])
	}
	return s
}

//bigo:ignore
func trusted(x int) int

//bigo:max O(n)
func UsesIgnore(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		s += trusted(xs[i])
	}
	return s
}

// A cost in the callee's own size: opaqueLinear costs O(k) in its k.

//bigo:cost O(k) where k=m
func opaqueLinear(m int) int

//bigo:max O(n*k) where n=len(xs), k=m
func CostWithSize(xs []int, m int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		s += opaqueLinear(m)
	}
	return s
}

// Un-annotated opaque callee: still unverifiable, and the diagnostic names it.

func unannotated(x int) int

//bigo:max O(n)
func StillBlocked(xs []int) int { // want `cannot verify budget O\(len\(xs\)\): unresolved cost at call to costignore.unannotated`
	s := 0
	for i := 0; i < len(xs); i++ {
		s += unannotated(xs[i])
	}
	return s
}

type Doer interface {
	//bigo:cost O(1)
	Do(x int) int
}

//bigo:max O(n)
func UsesInterfaceCost(xs []int, d Doer) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		s += d.Do(xs[i])
	}
	return s
}

// A budget on a declared-only function cannot be verified: there is no body to
// analyze. Review finding B5 fixed the engine, but the analyzer used to skip
// bodyless declarations before checking, so the symptom survived (finding S3).
// The fix a user wants here is //bigo:cost, which the diagnostic recommends.

//bigo:max O(1)
func BodylessBudget(n int) int // want `cannot verify budget O\(1\): function has no analyzable body`
