package edge

// Shapes that must verify (Within is silent — any diagnostic fails the test).

//bigo:max O(n)
func EarlyReturn(xs []int, t int) int {
	for i := 0; i < len(xs); i++ {
		if xs[i] == t {
			return i
		}
	}
	return -1
}

//bigo:max O(n)
func BreakOut(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		if xs[i] < 0 {
			break
		}
		s += xs[i]
	}
	return s
}

//bigo:max O(n^2)
func LabeledBreak(xs []int) int {
	s := 0
outer:
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs); j++ {
			if xs[j] < 0 {
				break outer
			}
			s++
		}
	}
	return s
}

//bigo:max O(n)
func SwitchInLoop(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		switch {
		case xs[i] > 0:
			s++
		case xs[i] < 0:
			s--
		}
	}
	return s
}

// Shapes that must be unverifiable — a bounded verdict here is the B1 bug.

//bigo:max O(n)
func InfiniteGEQ(n int) int { // want `cannot verify budget O\(n\)`
	s := 0
	for i := 0; i >= n; i++ {
		s++
	}
	return s
}

//bigo:max O(n)
func NegativeStep(n int) int { // want `cannot verify budget O\(n\)`
	s := 0
	for i := 0; i < n; i += -1 {
		s++
	}
	return s
}

//bigo:max O(n)
func ZeroStep(n int) int { // want `cannot verify budget O\(n\)`
	s := 0
	for i := 0; i < n; i += 0 {
		s++
	}
	return s
}

//bigo:max O(n) where n=b
func ParamStart(a, b int) int { // want `cannot verify budget O\(b\)`
	s := 0
	for i := a; i < b; i++ {
		s++
	}
	return s
}

// Triangular nests bound since the loop-algebra slice: the inner bound i is
// dominated by its own loop's guard (i < len(xs)).

//bigo:max O(n^2)
func Triangular(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < i; j++ {
			s++
		}
	}
	return s
}

// Irreducible control flow (goto into a cycle from two entries) has no
// natural loop; the function must be unverifiable, never O(1).

//bigo:max O(1)
func IrreducibleGoto(n int, c bool) int { // want `cannot verify budget O\(1\)`
	i := 0
	if c {
		goto b
	}
a:
	i++
b:
	i++
	if i < n {
		goto a
	}
	return i
}

// A variable offset in the loop condition shifts the trip count to (n-j).
// With j = -1000000 and n = 1 this runs a million times, so O(n) would be a
// wrong bound — the loop must be unverifiable.

//bigo:max O(n)
func OffsetCondition(n, j int) int { // want `cannot verify budget O\(n\)`
	s := 0
	for i := 0; i+j < n; i++ {
		s++
	}
	return s
}

// A closed-guard bisection whose hi update is `hi = mid` does NOT terminate:
// when lo == hi, mid == lo == hi, so hi never moves. R6 accepts the closed
// guard `lo <= hi` only when both ends move strictly past mid, which is why
// this must stay unverifiable. Deleting the c >= 1 condition in isHiUpdate
// makes this loop claim O(log n) — a wrong bound on a non-terminating loop.

//bigo:max O(log n) where n = len(s)
func ClosedBisectionHiEqMid(s []int, x int) int { // want `cannot verify budget O\(log\(len\(s\)\)\)`
	lo, hi := 0, len(s)-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if s[mid] < x {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return -1
}

// The terminating sibling, same guard: both ends move strictly past mid, so R6
// bounds it. Pins that the pin above fails for its stated reason (the hi
// update), not because the closed guard is rejected wholesale.

//bigo:max O(log n) where n = len(s)
func ClosedBisectionTerminates(s []int, x int) int {
	lo, hi := 0, len(s)-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if s[mid] < x {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return -1
}

// A two-pointer loop with a path that advances NEITHER pointer does not
// terminate: the measure i+j stalls. R7 bounds this shape only when every
// back-edge path advances exactly one pointer, which is why this stays
// unverifiable. Deleting that condition makes this loop claim
// O(len(a) + len(b)) — a wrong bound on a non-terminating loop.

//bigo:max O(n + m) where n = len(a), m = len(b)
func TwoPointerStalls(a, b []int) int { // want `cannot verify budget`
	i, j, n := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] < 0 {
			n++
			continue
		}
		if a[i] <= b[j] {
			i++
		} else {
			j++
		}
		n++
	}
	return n
}

// The terminating sibling: exactly one pointer advances per path, so R7 bounds
// it. Pins that the pin above fails for its stated reason (the stalling path),
// not because the two-pointer shape is rejected wholesale.

//bigo:max O(n + m) where n = len(a), m = len(b)
func TwoPointerMerges(a, b []int) int {
	i, j, n := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			i++
		} else {
			j++
		}
		n++
	}
	return n
}

// make([]T, 0, cap) has LENGTH 0 — its capacity is not its length. A size rule
// reading Cap here would claim O(len(s)) iterations for a loop that never runs:
// a wrong bound, not an imprecise one. Pins the direction. (⊤ is the honest
// answer today: len is the constant 0 and a constant-trip loop is a separate
// documented gap. What must never happen is O(len(s)).)

//bigo:max O(1)
func MakeZeroLenCap(s []int) int { // want `cannot verify budget O\(1\)`
	m := make([]int, 0, len(s))
	n := 0
	for i := 0; i < len(m); i++ {
		n++
	}
	return n
}

// The sibling: make([]T, len(s)) DOES have length len(s), so this is O(n).

//bigo:max O(n) where n = len(s)
func MakeFullLen(s []int) int {
	m := make([]int, len(s))
	n := 0
	for i := 0; i < len(m); i++ {
		n++
	}
	return n
}
