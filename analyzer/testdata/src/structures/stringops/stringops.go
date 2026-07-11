// Package stringops is the canonical-structures corpus for string processing.
package stringops

// CountByte is O(len(s)). Bounded today: counted loop over a string length.
//
//bigo:max O(n)
func CountByte(s string, b byte) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			n++
		}
	}
	return n
}

// ConcatAll — KNOWN LIMITATION ENTRY, read carefully. True cost is O(n^2) in
// copied bytes (each += reallocates and copies). bigo's documented v0.x cost
// model treats string concatenation as O(1) per operation
// (allocation-blindness; see README "What bigo does not count"), so the
// engine infers O(len(parts)) and the budget below verifies SILENTLY. That is
// the documented under-approximation, pinned here on purpose; the deferred
// smell-rule bundle (design spec §11.9) is the intended catcher. If a future
// cost model charges for concatenation, this entry flips to Exceeds — that is
// an improvement: delete this entry's budget and move it to a smell corpus.
//
//bigo:max O(n)
func ConcatAll(parts []string) string {
	s := ""
	for i := 0; i < len(parts); i++ {
		s += parts[i]
	}
	return s
}
