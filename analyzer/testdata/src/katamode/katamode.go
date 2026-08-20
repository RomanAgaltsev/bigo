package katamode

import (
	"strconv"
	"strings"
)

// Convert is bounded with or without -kata; what changes is the BOUND itself.
// strconv.Atoi is curated linear in its string, correctly. Under K-1 that
// string is an input token, so it costs unit and Convert becomes constant.
// This is the precise test of the overlay: not that a blocker disappears, but
// that a curated cost is deliberately overridden.
func Convert(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// Compare is the K-2 case. strings.Compare is curated linear in its operand;
// one comparison of two elements is one element operation when an algorithm is
// what is being graded.
func Compare(a, b string) int {
	return strings.Compare(a, b)
}

// ParseLine pins the OVERLAY'S BOUNDARY, and it is unverifiable either way.
// The overlay answers call COSTS; it does not invent SIZES. Here parts is the
// result of a call, so the loop over it has no nameable trip count, and that
// blocker is untouched by any cost model. Recorded deliberately: clearing one
// blocker reveals the next, and this is what the next one looks like.
func ParseLine(line string) []int {
	parts := strings.Split(line, " ")
	out := make([]int, len(parts))
	for i := range parts {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}
