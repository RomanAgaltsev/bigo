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
	"strings"
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
	s := ""
	for _, x := range xs {
		s += x // want `smell\(SM1\): string built by repeated concatenation in a loop`
	}
	return s
}

// SM1ConcatChained fires: chained + still accumulates the phi.
func SM1ConcatChained(xs []string, sep string) string {
	s := ""
	for _, x := range xs {
		s = s + x + sep // want `smell\(SM1\): string built by repeated concatenation in a loop`
	}
	return s
}

// SM1Sprintf fires: Sprintf self-accumulation.
func SM1Sprintf(xs []string) string {
	s := ""
	for i, x := range xs {
		s = fmt.Sprintf("%s%d%s", s, i, x) // want `smell\(SM1\): string built by repeated concatenation in a loop`
	}
	return s
}

// SM1NoFireBuilder uses strings.Builder (no string phi), no SM1 smell.
func SM1NoFireBuilder(xs []string) string {
	var b strings.Builder
	for _, x := range xs {
		b.WriteString(x)
	}
	return b.String()
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

// SM4Compile fires: a constant pattern recompiled every iteration is
// hoistable, which is what makes the advice actionable.
func SM4Compile(patterns []string) []bool {
	out := make([]bool, 0, len(patterns))
	for _, p := range patterns {
		re := regexp.MustCompile("x") // want `smell\(SM4\): regexp compiled inside a loop`
		out = append(out, re.MatchString(p))
	}
	return out
}

// SM4NoFireVaryingPattern does not fire: the pattern is the range variable, so
// there is nothing to hoist. This was a firing fixture before the invariance
// analysis landed, and it is the shape that dominated the real-world findings.
func SM4NoFireVaryingPattern(patterns []string) []bool {
	out := make([]bool, 0, len(patterns))
	for _, p := range patterns {
		re := regexp.MustCompile(p)
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

// SM5Sort fires: the same parameter slice re-sorted every iteration, with
// nothing writing to it, so the 2nd..nth sorts are no-ops.
func SM5Sort(s []int, xs []string) {
	for range xs {
		slices.Sort(s) // want `smell\(SM5\): sort inside a data-dependent loop`
	}
}

// SM5NoFirePerIteration does not fire: a different slice each iteration, so
// every sort is necessary work. Formerly a firing fixture.
func SM5NoFirePerIteration(groups [][]int) {
	for _, g := range groups {
		slices.Sort(g)
	}
}

// SM5NoFireMutated does not fire: the slice grows inside the loop, so it must
// be re-sorted.
func SM5NoFireMutated(s []int, xs []int) []int {
	for _, x := range xs {
		s = append(s, x)
		slices.Sort(s)
	}
	return s
}

// SM5NoFireSortInterface does not fire: sort.Sort takes a sort.Interface,
// which exposes no slice operand whose stability could be checked.
func SM5NoFireSortInterface(s sort.Interface, xs []int) {
	for range xs {
		sort.Sort(s)
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

// --- SM3: append without prealloc ---

// SM3Append fires: zero-capacity slice grown by append in a resolvable loop.
func SM3Append(xs []int) []int {
	var out []int
	for _, x := range xs {
		out = append(out, x) // want `smell\(SM3\): append in a loop bounded by`
	}
	return out
}

// SM3NoFirePrealloc does not fire: capacity given.
func SM3NoFirePrealloc(xs []int) []int {
	out := make([]int, 0, len(xs))
	for _, x := range xs {
		out = append(out, x)
	}
	return out
}

// --- SM6: map without size hint ---

// SM6Map fires: make(map) without size hint, grown in a resolvable loop.
func SM6Map(ks []string, vs []int) map[string]int {
	m := make(map[string]int) // want `smell\(SM6\): map grown in a loop bounded by`
	for i, k := range ks {
		m[k] = vs[i]
	}
	return m
}

// SM6NoFireHint does not fire: size hint given.
func SM6NoFireHint(ks []string, vs []int) map[string]int {
	m := make(map[string]int, len(ks))
	for i, k := range ks {
		m[k] = vs[i]
	}
	return m
}

// --- SM7: double-lookup ---

// SM7Double fires: comma-ok then plain lookup, same X and key.
func SM7Double(m map[int]int, k int) int {
	if _, ok := m[k]; ok { // want `smell\(SM7\): redundant map lookup`
		return m[k]
	}
	return 0
}

// SM7NoFireNoSecond does not fire: only one lookup.
func SM7NoFireNoSecond(m map[int]int, k int) int {
	if v, ok := m[k]; ok {
		return v
	}
	return 0
}

// --- SM2: linear-scan-where-map-fits ---

// SM2Scan fires: repeated Contains over a parameter slice in a data-dependent loop.
func SM2Scan(s []int, items []int) int {
	n := 0
	for _, v := range items {
		if slices.Contains(s, v) { // want `smell\(SM2\): repeated linear scan over the same slice`
			n += v
		}
	}
	return n
}

// SM2NoFireInvariantNeedle does not fire: needle loop-invariant.
func SM2NoFireInvariantNeedle(s []int, v int) int {
	n := 0
	for _, x := range s {
		_ = x
		if slices.Contains(s, v) {
			n++
		}
	}
	return n
}

// SM2NoFireNonParam does not fire: scan target is not a parameter (rebuilt slice).
func SM2NoFireNonParam(items [][]int) int {
	n := 0
	for _, s := range items {
		if slices.Contains(s, 0) {
			n++
		}
	}
	return n
}

// --- SM8: exponential recursion ---

// SM8Fib fires: provably exponential recursion.
func SM8Fib(n int) int { // want `smell\(SM8\): provably exponential recursion`
	if n < 2 {
		return n
	}
	return SM8Fib(n-1) + SM8Fib(n-2)
}

// SM8NoFireLinear does not fire: a=1 countdown is linear.
func SM8NoFireLinear(n int) int {
	if n <= 0 {
		return 0
	}
	return 1 + SM8NoFireLinear(n-1)
}

// SM8NoFireMemo does not fire: the comma-ok cache hit dominates the self-calls,
// so each argument is computed once — O(n), not exponential.
func SM8NoFireMemo(n int, memo map[int]int) int {
	if n < 2 {
		return n
	}
	if v, ok := memo[n]; ok {
		return v
	}
	r := SM8NoFireMemo(n-1, memo) + SM8NoFireMemo(n-2, memo)
	memo[n] = r
	return r
}
