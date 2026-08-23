package explain

// Bounded exercises the loop line, the term line, and a priced call.
func Bounded(xs []int) int {
	n := 0
	for i := 0; i < len(xs); i++ {
		n += len(xs)
	}
	return n
}

// Unbounded exercises the "no rule matched" line: -v must show how far the
// derivation got, and must not invent a rule name.
func Unbounded(xs []int) int {
	n := 0
	for len(xs) > 0 {
		xs = xs[n:]
		n++
	}
	return n
}

// Recursive exercises the recursion guard. InferTop answers a self-recursive
// function from the RECURRENCE SOLVER, not from the body walk, so a term line
// built from the traced walk would explain a bound the verdict never used.
func Recursive(n int) int {
	if n <= 0 {
		return 0
	}
	return 1 + Recursive(n-1)
}

