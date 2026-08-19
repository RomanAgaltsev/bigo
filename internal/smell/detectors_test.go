package smell

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func ruleset(ids ...string) map[string]bool {
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out
}

func detectOne(t *testing.T, src, fnName string, enabled map[string]bool) []Finding {
	t.Helper()
	pkg, _, err := ssasupport.BuildGeneric(src)
	if err != nil {
		t.Fatal(err)
	}
	return Detect(ssasupport.Func(pkg, fnName), enabled)
}

// detectLines returns the 1-based source lines of every finding for rule, in
// finding order. It is the anchoring counterpart to detectOne: where a finding
// points matters as much as whether it fires.
func detectLines(t *testing.T, src, fnName, rule string) []int {
	t.Helper()
	pkg, fset, err := ssasupport.BuildGeneric(src)
	if err != nil {
		t.Fatal(err)
	}
	var out []int
	for _, f := range Detect(ssasupport.Func(pkg, fnName), ruleset(rule)) {
		if f.Rule == rule {
			out = append(out, fset.Position(f.Pos).Line)
		}
	}
	return out
}

func ruleCount(findings []Finding, rule string) int {
	n := 0
	for _, f := range findings {
		if f.Rule == rule {
			n++
		}
	}
	return n
}

func wantRuleCount(t *testing.T, findings []Finding, rule string, want int) {
	t.Helper()
	if got := ruleCount(findings, rule); got != want {
		t.Errorf("rule %s: got %d findings, want %d (%+v)", rule, got, want, findings)
	}
}

func TestSM1ConcatInLoop(t *testing.T) {
	src := `package input
func RangeConcat(xs []string) string {
	s := ""
	for _, x := range xs { s += x }
	return s
}
func Builder(xs []string) string {
	var b []byte
	for _, x := range xs { b = append(b, x...) }
	return string(b)
}
func FreshString(xs []string) string {
	out := ""
	for _, x := range xs { t := x + "y"; _ = t }
	return out
}
func ConstTrip() string {
	s := ""
	for i := 0; i < 10; i++ { s += "x" }
	return s
}
`
	wantRuleCount(t, detectOne(t, src, "RangeConcat", ruleset("SM1")), "SM1", 1)
	wantRuleCount(t, detectOne(t, src, "Builder", ruleset("SM1")), "SM1", 0)
	wantRuleCount(t, detectOne(t, src, "FreshString", ruleset("SM1")), "SM1", 0)
	wantRuleCount(t, detectOne(t, src, "ConstTrip", ruleset("SM1")), "SM1", 0)
}

func TestSM1SprintfInLoop(t *testing.T) {
	src := `package input
import "fmt"
func S(xs []string) string {
	s := ""
	for i, x := range xs { s = fmt.Sprintf("%s%d%s", s, i, x) }
	return s
}
`
	wantRuleCount(t, detectOne(t, src, "S", ruleset("SM1")), "SM1", 1)
}

func TestSM4CompileInLoop(t *testing.T) {
	src := `package input
import "regexp"

// ConstPattern: a literal recompiled every iteration. Hoistable — the rule's
// primary positive control.
func ConstPattern() int {
	c := 0
	for i := 0; i < 10; i++ {
		re := regexp.MustCompile("x")
		if re.MatchString("x") {
			c++
		}
	}
	return c
}

// ParamPattern: invariant across the loop because a parameter cannot be
// redefined inside it.
func ParamPattern(pat string, xs []string) int {
	c := 0
	for _, x := range xs {
		re := regexp.MustCompile(pat)
		if re.MatchString(x) {
			c++
		}
	}
	return c
}

// InnerInvariant: pat varies across the OUTER loop but is constant across the
// inner one, so the compile hoists out of the inner loop. This fixture is the
// discriminator for evaluating invariance against the innermost enclosing
// loop; an outermost check silences it.
func InnerInvariant(xs, ys []string) int {
	c := 0
	for _, x := range xs {
		pat := "^" + x
		for _, y := range ys {
			re := regexp.MustCompile(pat)
			if re.MatchString(y) {
				c++
			}
		}
	}
	return c
}

// VaryingPattern: NOTHING to hoist — the pattern is the range variable. This
// was the rule's positive fixture before the invariance analysis landed, and
// it is the shape that made 12 of 12 sampled real-world findings unactionable.
func VaryingPattern(patterns []string) int {
	c := 0
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		if re.MatchString("x") {
			c++
		}
	}
	return c
}

// Hoisted: already correct.
func Hoisted(patterns []string) int {
	re := regexp.MustCompile("x")
	c := 0
	for _, p := range patterns {
		if re.MatchString(p) {
			c++
		}
	}
	return c
}
`
	wantRuleCount(t, detectOne(t, src, "ConstPattern", ruleset("SM4")), "SM4", 1)
	wantRuleCount(t, detectOne(t, src, "ParamPattern", ruleset("SM4")), "SM4", 1)
	wantRuleCount(t, detectOne(t, src, "InnerInvariant", ruleset("SM4")), "SM4", 1)
	wantRuleCount(t, detectOne(t, src, "VaryingPattern", ruleset("SM4")), "SM4", 0)
	wantRuleCount(t, detectOne(t, src, "Hoisted", ruleset("SM4")), "SM4", 0)
}

func TestSM5SortInLoop(t *testing.T) {
	src := `package input
import ("slices"; "sort")
func InDataDep(groups [][]int) {
	for _, g := range groups { slices.Sort(g) }
}
func InConstTrip(groups [][]int) {
	for i := 0; i < 10; i++ { slices.Sort(groups[i]) }
}
func Outside(g []int) { slices.Sort(g) }
func SortSliceInLoop(groups [][]int) {
	for _, g := range groups { sort.Slice(g, func(i, j int) bool { return g[i] < g[j] }) }
}
`
	wantRuleCount(t, detectOne(t, src, "InDataDep", ruleset("SM5")), "SM5", 1)
	wantRuleCount(t, detectOne(t, src, "InConstTrip", ruleset("SM5")), "SM5", 0)
	wantRuleCount(t, detectOne(t, src, "Outside", ruleset("SM5")), "SM5", 0)
	wantRuleCount(t, detectOne(t, src, "SortSliceInLoop", ruleset("SM5")), "SM5", 1)
}

func TestSM3AppendNoPrealloc(t *testing.T) {
	src := `package input
func Append(xs []int) []int {
	var out []int
	for _, x := range xs { out = append(out, x) }
	return out
}
func Prealloc(xs []int) []int {
	out := make([]int, 0, len(xs))
	for _, x := range xs { out = append(out, x) }
	return out
}
`
	wantRuleCount(t, detectOne(t, src, "Append", ruleset("SM3")), "SM3", 1)
	wantRuleCount(t, detectOne(t, src, "Prealloc", ruleset("SM3")), "SM3", 0)
}

func TestSM6MapNoSizeHint(t *testing.T) {
	src := `package input
func MapNoHint(ks []string, vs []int) map[string]int {
	m := make(map[string]int)
	for i, k := range ks { m[k] = vs[i] }
	return m
}
func MapHint(ks []string, vs []int) map[string]int {
	m := make(map[string]int, len(ks))
	for i, k := range ks { m[k] = vs[i] }
	return m
}
`
	wantRuleCount(t, detectOne(t, src, "MapNoHint", ruleset("SM6")), "SM6", 1)
	wantRuleCount(t, detectOne(t, src, "MapHint", ruleset("SM6")), "SM6", 0)
}

func TestSM7DoubleLookup(t *testing.T) {
	src := `package input
import "slices"
func MapDouble(m map[int]int, k int) int {
	if _, ok := m[k]; ok { return m[k] }
	return 0
}
func MapSingle(m map[int]int, k int) int {
	if v, ok := m[k]; ok { return v }
	return 0
}
func ContainsIndex(s []int, items []int) int {
	n := 0
	for _, v := range items {
		if slices.Contains(s, v) { n += slices.Index(s, v) }
	}
	return n
}
func ContainsOnly(s []int, v int) bool { return slices.Contains(s, v) }
`
	wantRuleCount(t, detectOne(t, src, "MapDouble", ruleset("SM7")), "SM7", 1)
	wantRuleCount(t, detectOne(t, src, "MapSingle", ruleset("SM7")), "SM7", 0)
	wantRuleCount(t, detectOne(t, src, "ContainsIndex", ruleset("SM7")), "SM7", 1)
	wantRuleCount(t, detectOne(t, src, "ContainsOnly", ruleset("SM7")), "SM7", 0)
}

func TestSM2LinearScan(t *testing.T) {
	src := `package input
import "slices"
// fires: Contains over a parameter slice, loop-varying needle.
func Scan(s []int, items []int) int {
	n := 0
	for _, v := range items {
		if slices.Contains(s, v) { n += v }
	}
	return n
}
// no-fire: needle loop-invariant.
func InvariantNeedle(s []int, v int) int {
	n := 0
	for _, x := range s { _ = x; if slices.Contains(s, v) { n++ } }
	return n
}
// no-fire: scan target is a range variable, not a parameter.
func NonParam(items [][]int) int {
	n := 0
	for _, g := range items { if slices.Contains(g, 0) { n++ } }
	return n
}
// no-fire: outside a data-dependent loop.
func OutsideLoop(s []int, v int) bool { return slices.Contains(s, v) }
`
	wantRuleCount(t, detectOne(t, src, "Scan", ruleset("SM2")), "SM2", 1)
	wantRuleCount(t, detectOne(t, src, "InvariantNeedle", ruleset("SM2")), "SM2", 0)
	wantRuleCount(t, detectOne(t, src, "NonParam", ruleset("SM2")), "SM2", 0)
	wantRuleCount(t, detectOne(t, src, "OutsideLoop", ruleset("SM2")), "SM2", 0)
}

func TestSM8Exponential(t *testing.T) {
	src := `package input
func Fib(n int) int {
	if n < 2 { return n }
	return Fib(n-1) + Fib(n-2)
}
func Linear(n int) int {
	if n <= 0 { return 0 }
	return 1 + Linear(n-1)
}
func BinSearch(n int) int {
	if n > 0 { return BinSearch(n / 2) }
	return 0
}
func Unguarded(n int) int {
	return Unguarded(n-1) + Unguarded(n-2)
}
func FibMemo(n int, memo map[int]int) int {
	if n < 2 { return n }
	if v, ok := memo[n]; ok { return v }
	r := FibMemo(n-1, memo) + FibMemo(n-2, memo)
	memo[n] = r
	return r
}
`
	wantRuleCount(t, detectOne(t, src, "Fib", ruleset("SM8")), "SM8", 1)
	wantRuleCount(t, detectOne(t, src, "Linear", ruleset("SM8")), "SM8", 0)
	wantRuleCount(t, detectOne(t, src, "BinSearch", ruleset("SM8")), "SM8", 0)
	wantRuleCount(t, detectOne(t, src, "Unguarded", ruleset("SM8")), "SM8", 0)
	// Memoized recursion is O(n), not exponential — SM8 must stay silent.
	wantRuleCount(t, detectOne(t, src, "FibMemo", ruleset("SM8")), "SM8", 0)
}

// TestSM5NestedConstTripInsideDataDep pins the gate: SM5 asks whether ANY
// enclosing loop is data-dependent, not whether the innermost one is. The
// operand predicate is evaluated against the innermost loop, but the gate is
// not — otherwise a sort in a constant-trip inner loop would escape even though
// the outer loop repeats it a data-dependent number of times.
func TestSM5NestedConstTripInsideDataDep(t *testing.T) {
	src := `package input
import "slices"
func Nested(s []int, xs []string) {
	for range xs {
		for i := 0; i < 3; i++ {
			slices.Sort(s)
		}
	}
}
`
	wantRuleCount(t, detectOne(t, src, "Nested", ruleset("SM5")), "SM5", 1)
}
