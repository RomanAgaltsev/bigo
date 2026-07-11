package tripcount

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func firstLoop(t *testing.T, src string) (*loopnest.Loop, *fieldpath.Stability) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	forest := loopnest.Build(fn)
	if len(forest.Roots) == 0 {
		t.Fatal("no loops found")
	}
	return forest.Roots[0], fieldpath.Analyze(fn)
}

func TestOf(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string // bound.String()
	}{
		{
			"counter over len",
			`package input
func f(xs []int) { s := 0; for i := 0; i < len(xs); i++ { s += xs[i] }; _ = s }`,
			"O(len(xs))",
		},
		{
			"counter over int param",
			`package input
func f(n int) { s := 0; for i := 0; i < n; i++ { s += i }; _ = s }`,
			"O(n)",
		},
		{
			"range over slice",
			`package input
func f(xs []int) { s := 0; for range xs { s++ }; _ = s }`,
			"O(len(xs))",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Of(firstLoop(t, tt.src))
			if got.String() != tt.want {
				t.Errorf("Of = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestOfUnrecognizedIsTop(t *testing.T) {
	// Loop bounded by an unrelated function call -> not recognized.
	src := `package input
func g() int
func f() { for i := 0; i < g(); i++ { _ = i } }`
	if got := Of(firstLoop(t, src)); !got.IsTop() {
		t.Errorf("want Top(), got %q", got.String())
	}
}

func TestOfRejectsUnsoundShapes(t *testing.T) {
	tests := []struct{ name, src string }{
		{
			"increasing induction with >= bound (may never terminate)",
			`package input
func f(n int) int { s := 0; for i := 0; i >= n; i++ { s++ }; return s }`,
		},
		{
			"negative constant step",
			`package input
func f(n int) int { s := 0; for i := 0; i < n; i += -1 { s++ }; return s }`,
		},
		{
			"zero constant step",
			`package input
func f(n int) int { s := 0; for i := 0; i < n; i += 0 { s++ }; return s }`,
		},
		{
			"non-constant start (trip count is n-m, not O(n))",
			`package input
func f(m, n int) int { s := 0; for i := m; i < n; i++ { s++ }; return s }`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(firstLoop(t, tt.src)); !got.IsTop() {
				t.Errorf("Of = %q, want Top (unverifiable)", got.String())
			}
		})
	}
}

func TestOfAcceptsSoundVariants(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"bound-on-left with > (user wrote n > i)",
			`package input
func f(n int) int { s := 0; for i := 0; n > i; i++ { s++ }; return s }`,
			"O(n)",
		},
		{
			"constant step 2 (same asymptotic class)",
			`package input
func f(n int) int { s := 0; for i := 0; i < n; i += 2 { s++ }; return s }`,
			"O(n)",
		},
		{
			"negative constant start (still O(n))",
			`package input
func f(n int) int { s := 0; for i := -5; i < n; i++ { s++ }; return s }`,
			"O(n)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(firstLoop(t, tt.src)); got.String() != tt.want {
				t.Errorf("Of = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestOfRejectsVariableOffsetInComparison(t *testing.T) {
	// `i+j < n` is `i < n-j`: for j < 0 the loop runs more than n times, so a
	// bound of O(n) is a wrong bound, not an imprecise one.
	tests := []struct{ name, src string }{
		{
			"induction plus parameter offset (trip count is n-j)",
			`package input
func f(n, j int) int { s := 0; for i := 0; i+j < n; i++ { s++ }; return s }`,
		},
		{
			"induction plus parameter offset against a length",
			`package input
func f(xs []int, k int) int { s := 0; for i := 0; i+k < len(xs); i++ { s++ }; return s }`,
		},
		{
			"offset written on the left of the induction variable",
			`package input
func f(n, j int) int { s := 0; for i := 0; j+i < n; i++ { s++ }; return s }`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(firstLoop(t, tt.src)); !got.IsTop() {
				t.Errorf("Of = %q, want Top (trip count is bound-offset, not O(bound))", got.String())
			}
		})
	}
}

func TestOfAcceptsConstantOffsetInComparison(t *testing.T) {
	// A constant offset shifts the trip count by O(1) and stays in the same
	// asymptotic class. This is the shape the `for range` lowering compares,
	// so it must keep working.
	const src = `package input
func f(n int) int { s := 0; for i := 0; i+1 < n; i++ { s++ }; return s }`
	if got, want := Of(firstLoop(t, src)).String(), "O(n)"; got != want {
		t.Errorf("Of = %q, want %q", got, want)
	}
}

func TestOfFieldBounds(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"value-receiver style field length",
			`package input
type S struct{ items []int }
func f(s S) int { t := 0; for i := 0; i < len(s.items); i++ { t++ }; return t }`,
			"O(len(s.items))",
		},
		{
			"pointer param hoisted length",
			`package input
type S struct{ items []int }
func f(s *S) int { n := len(s.items); t := 0; for i := 0; i < n; i++ { t++ }; return t }`,
			"O(len(s.items))",
		},
		{
			"pointer param numeric field",
			`package input
type S struct{ limit int }
func f(s *S) int { t := 0; for i := 0; i < s.limit; i++ { t++ }; return t }`,
			"O(s.limit)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(firstLoop(t, tt.src)).String(); got != tt.want {
				t.Errorf("Of = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOfFieldBoundMutatedIsTop(t *testing.T) {
	const src = `package input
type S struct{ items []int }
func f(s *S, v int) int {
	s.items = append(s.items, v)
	t := 0
	for i := 0; i < len(s.items); i++ { t++ }
	return t
}`
	if got := Of(firstLoop(t, src)); !got.IsTop() {
		t.Errorf("Of = %q, want Top — the function grew the field before the loop", got.String())
	}
}

func TestRuleIncreasingGraduations(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"selection-sort inner: start i+1 has constant lower bound 1",
			`package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := i + 1; j < len(xs); j++ { s++ }
	}
	return s
}`,
			"O(len(xs))", // the INNER loop — see innerLoop helper below
		},
		{
			"triangular inner: bound i is guard-bounded by len(xs)",
			`package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < i; j++ { s++ }
	}
	return s
}`,
			"O(len(xs))",
		},
		{
			"bubble inner: bound len(xs)-1-i",
			`package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs)-1-i; j++ { s++ }
	}
	return s
}`,
			"O(len(xs))",
		},
		{
			"half-length reverse index form",
			`package input
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs)/2; i++ { s++ }
	return s
}`,
			"O(len(xs))",
		},
		{
			"two-pointer reverse: extent is the decreasing phi's init",
			`package input
func f(xs []int) int {
	s := 0
	for i, j := 0, len(xs)-1; i < j; i, j = i+1, j-1 { s++ }
	return s
}`,
			"O(len(xs))",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)).String(); got != tt.want {
				t.Errorf("Of(inner) = %q, want %q", got, tt.want)
			}
		})
	}
}

// innerLoop returns the deepest loop plus stability — the loop the new rules
// must bound. For single-loop functions it is the only loop.
func innerLoop(t *testing.T, src string) (*loopnest.Loop, *fieldpath.Stability) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	forest := loopnest.Build(fn)
	var deepest *loopnest.Loop
	var walk func(l *loopnest.Loop)
	walk = func(l *loopnest.Loop) {
		if deepest == nil || l.Depth > deepest.Depth {
			deepest = l
		}
		for _, c := range l.Children {
			walk(c)
		}
	}
	for _, r := range forest.Roots {
		walk(r)
	}
	if deepest == nil {
		t.Fatal("no loops found")
	}
	return deepest, fieldpath.Analyze(fn)
}

// TestLoopAlgebraStaysTop pins the shapes the generalizations must NOT
// accept. Every PR of the loop-algebra plan re-runs this; a flip here is a
// wrong-bound bug by construction.
func TestLoopAlgebraStaysTop(t *testing.T) {
	tests := []struct{ name, src string }{
		{"B1 wrong guard direction", `package input
func f(n int) int { s := 0; for i := 0; i >= n; i++ { s++ }; return s }`},
		{"B1 negative step under upper guard", `package input
func f(n int) int { s := 0; for i := 0; i < n; i += -1 { s++ }; return s }`},
		{"B1 zero step", `package input
func f(n int) int { s := 0; for i := 0; i < n; i += 0 { s++ }; return s }`},
		{"B1 parameter start", `package input
func f(m, n int) int { s := 0; for i := m; i < n; i++ { s++ }; return s }`},
		{"S1 variable offset in comparand", `package input
func f(n, j int) int { s := 0; for i := 0; i+j < n; i++ { s++ }; return s }`},
		{"decreasing toward a parameter bound", `package input
func f(n, m int) int { s := 0; for j := n; j > m; j-- { s++ }; return s }`},
		{"geometric from zero never grows", `package input
func f(n int) int { s := 0; for i := 0; i < n; i *= 2 { s++ }; return s }`},
		{"param-start geometric (infinite for negative starts)", `package input
func f(h []int, i int) int {
	s := 0
	for 2*i+1 < len(h) {
		s++
		i = 2*i + 1
	}
	return s
}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)); !got.IsTop() {
				t.Errorf("Of = %q, want Top", got.String())
			}
		})
	}
}

func TestRuleDecreasing(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"countdown from a parameter",
			`package input
func f(n int) int { s := 0; for j := n; j > 0; j-- { s++ }; return s }`,
			"O(n)",
		},
		{
			"insertion-sort inner: init i is guard-bounded, && conjunct is body",
			`package input
func f(xs []int) int {
	s := 0
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j-1] > xs[j]; j-- { s++ }
	}
	return s
}`,
			"O(len(xs))",
		},
		{
			"countdown to a small positive constant",
			`package input
func f(n int) int { s := 0; for j := n; j >= 5; j -= 2 { s++ }; return s }`,
			"O(n)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)).String(); got != tt.want {
				t.Errorf("Of(inner) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRuleGeometric(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"doubling up from 1",
			`package input
func f(n int) int { s := 0; for i := 1; i < n; i *= 2 { s++ }; return s }`,
			"O(log(n))",
		},
		{
			"sift-down from the root: comparand 2i+1, steps 2i+1 / 2i+2",
			`package input
func f(h []int) int {
	s := 0
	i := 0
	for 2*i+1 < len(h) {
		c := 2*i + 1
		if c+1 < len(h) && h[c+1] < h[c] {
			c++
		}
		s++
		i = c
	}
	return s
}`,
			"O(log(len(h)))",
		},
		{
			"halving down to zero",
			`package input
func f(n int) int { s := 0; for i := n; i > 0; i /= 2 { s++ }; return s }`,
			"O(log(n))",
		},
		{
			"sift-up: (i-1)/2 from a parameter start",
			`package input
func f(h []int, i int) int {
	s := 0
	for i > 0 {
		s++
		i = (i - 1) / 2
	}
	return s
}`,
			"O(log(i))",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)).String(); got != tt.want {
				t.Errorf("Of(inner) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRuleRangeNext(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"range over a map parameter, clean body",
			`package input
func f(m map[string]int) int { s := 0; for _, v := range m { s += v }; return s }`,
			"O(len(m))",
		},
		{
			"range over a string parameter",
			`package input
func f(str string) int { s := 0; for _, r := range str { s += int(r) }; return s }`,
			"O(len(str))",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)).String(); got != tt.want {
				t.Errorf("Of(inner) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRuleRangeNextDirtyMapStaysTop(t *testing.T) {
	tests := []struct{ name, src string }{
		{
			"body inserts into a map (unspecified visitation of new keys)",
			`package input
func f(m map[string]int) int {
	s := 0
	for k, v := range m {
		m[k+"x"] = v
		s++
	}
	return s
}`,
		},
		{
			"body calls a function that could mutate the map",
			`package input
func g(m map[string]int)
func f(m map[string]int) int {
	s := 0
	for range m {
		g(m)
		s++
	}
	return s
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(innerLoop(t, tt.src)); !got.IsTop() {
				t.Errorf("Of = %q, want Top — mutation during map range is unbounded", got.String())
			}
		})
	}
}
