package tripcount

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func firstLoop(t *testing.T, src string) *loopnest.Loop {
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
	return forest.Roots[0]
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
