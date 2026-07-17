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
	m := make(map[string]int) // want `smell\(SM6\): map built without a size hint in a loop bounded by`
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
