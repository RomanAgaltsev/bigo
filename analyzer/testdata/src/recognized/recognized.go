package recognized

// Scan is the two-pointer shape. Its budget is DELIBERATE and load-bearing:
// the recognition names exactly this bound, so if a recognized bound could
// reach a verdict the budget would pass. It must still fail, because the
// proved channel cannot bound this loop and a recognition carries no verdict
// power. The want comment is the assertion.
//
//bigo:max O(len(xs))
func Scan(xs []int, left, right int) int { // want `cannot verify budget O\(len\(xs\)\)`
	swaps := 0
	for left < right {
		for xs[left] < 0 {
			left++
		}
		for xs[right] > 0 {
			right--
		}
		if left < right {
			xs[left], xs[right] = xs[right], xs[left]
			swaps++
		}
	}
	return swaps
}
