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

// ParseLine WAS the overlay-boundary fixture and is now bounded, O(len(line)),
// identically with and without -kata. internal/resultsize curates
// len(strings.Split(s, sep)) as O(len(s)) (2026-08-29), so the loop finally has
// a nameable trip count.
//
// It is kept, and its assertion inverted, because it now pins something
// sharper than it did as a refusal: a curated SIZE fact is not a cost model, so
// it must apply in BOTH modes. See ParseUser for the boundary this used to
// illustrate.
func ParseLine(line string) []int {
	parts := strings.Split(line, " ")
	out := make([]int, len(parts))
	for i := range parts {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}

// tokenize is a USER function returning a slice. Nothing curates its result
// length and nothing ever will, which is what makes it the durable example.
func tokenize(line string) []string {
	if line == "" {
		return nil
	}
	return strings.Fields(line)
}

// ParseUser is what ParseLine used to be: a loop over a call result with no
// nameable trip count, unverifiable under every cost model. It replaces
// ParseLine as the overlay's-boundary fixture because ParseLine graduated when
// internal/resultsize curated strings.Split (2026-08-29) — the shape it was
// written to illustrate stopped being an example of itself.
//
// The boundary it pins is unchanged: a cost model answers what a call COSTS
// and never what its result MEASURES.
func ParseUser(line string) []int {
	parts := tokenize(line)
	out := make([]int, len(parts))
	for i := range parts {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}
