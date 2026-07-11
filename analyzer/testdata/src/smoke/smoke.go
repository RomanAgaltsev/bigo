package smoke

func Noop(xs []int) int {
	total := 0
	for i := 0; i < len(xs); i++ {
		total += xs[i]
	}
	return total
}

// Closure calls a func value, which bigo cannot see through: unverifiable.
// Report mode must still name it (and its blocker), not silently omit it.
func Closure(f func(int) int, xs []int) int {
	total := 0
	for i := 0; i < len(xs); i++ {
		total += f(xs[i])
	}
	return total
}
