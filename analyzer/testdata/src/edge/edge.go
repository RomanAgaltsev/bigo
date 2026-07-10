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

// Triangular nests are a documented v1 precision gap (inner bound is a phi,
// not a size) — pinned here so a Phase-2 graduation moves this entry.

//bigo:max O(n^2)
func Triangular(xs []int) int { // want `cannot verify budget O\(len\(xs\)\^2\)`
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
