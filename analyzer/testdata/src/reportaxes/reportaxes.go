package reportaxes

// Zed sorts last alphabetically but is declared first, which is the order the
// report must follow.
func Zed(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}

// Alpha carries no directive and must still get BOTH axes: the kata rubric
// grades time and space alike.
func Alpha(xs []int) []int {
	out := make([]int, 0, len(xs))
	for _, x := range xs {
		out = append(out, x)
	}
	return out
}
