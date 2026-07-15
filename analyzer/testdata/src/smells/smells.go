// Package smells is the fire/no-fire corpus for the advisory smell rules
// SM1 through SM8. Each function is annotated with a want comment where a smell
// is expected to fire; functions without wants must stay silent (analysistest
// fails on any unexpected diagnostic, that is the zero-spray baseline).
package smells

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
)

// Clean is the zero-spray baseline: no smell fires on clean code.
func Clean(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}

// --- SM1: concat-in-loop ---

// SM1Concat fires: string built by += in a data-dependent loop.
func SM1Concat(xs []string) string {
	s := "" // want `smell\(SM1\): string built by repeated concatenation in a loop`
	for _, x := range xs {
		s += x
	}
	return s
}

// SM1ConcatChained fires: chained + still accumulates the phi.
func SM1ConcatChained(xs []string, sep string) string {
	s := "" // want `smell\(SM1\): string built by repeated concatenation in a loop`
	for _, x := range xs {
		s = s + x + sep
	}
	return s
}

// SM1Sprintf fires: Sprintf self-accumulation.
func SM1Sprintf(xs []string) string {
	s := "" // want `smell\(SM1\): string built by repeated concatenation in a loop`
	for i, x := range xs {
		s = fmt.Sprintf("%s%d%s", s, i, x)
	}
	return s
}

// SM1NoFireBuilder uses bytes append (no string phi), no smell.
func SM1NoFireBuilder(xs []string) string {
	var b []byte
	for _, x := range xs {
		b = append(b, x...)
	}
	return string(b)
}

// SM1NoFireConstTrip does not fire: constant-trip loop is not a smell.
func SM1NoFireConstTrip() string {
	s := ""
	for i := 0; i < 10; i++ {
		s += "x"
	}
	return s
}

// --- SM4: regexp compile-in-loop ---

// SM4Compile fires: MustCompile inside any natural loop.
func SM4Compile(patterns []string) []bool {
	out := make([]bool, 0, len(patterns))
	for _, p := range patterns {
		re := regexp.MustCompile(p) // want `smell\(SM4\): regexp compiled inside a loop`
		out = append(out, re.MatchString("x"))
	}
	return out
}

// SM4NoFireHoisted does not fire: compile before the loop.
func SM4NoFireHoisted(patterns []string) []bool {
	re := regexp.MustCompile("x")
	out := make([]bool, 0, len(patterns))
	for _, p := range patterns {
		out = append(out, re.MatchString(p))
	}
	return out
}

// --- SM5: sort-in-loop ---

// SM5Sort fires: slices.Sort inside a data-dependent loop.
func SM5Sort(groups [][]int) {
	for _, g := range groups {
		slices.Sort(g) // want `smell\(SM5\): sort inside a data-dependent loop`
	}
}

// SM5SortSlice fires: sort.Slice inside a data-dependent loop.
func SM5SortSlice(groups [][]int) {
	for _, g := range groups {
		sort.Slice(g, func(i, j int) bool { return g[i] < g[j] }) // want `smell\(SM5\): sort inside a data-dependent loop`
	}
}

// SM5NoFireConstTrip does not fire: constant-trip loop.
func SM5NoFireConstTrip(groups [][]int) {
	for i := 0; i < 10; i++ {
		slices.Sort(groups[i])
	}
}

// SM5NoFireOutside does not fire: sort outside any loop.
func SM5NoFireOutside(g []int) {
	slices.Sort(g)
}
