package tripcount

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// lastLoopOf builds src and returns f's loop with the highest header block
// index — the second of two sequential loops — plus f's Stability.
func lastLoopOf(t *testing.T, src string) (*loopnest.Loop, *fieldpath.Stability) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	forest := loopnest.Build(fn)
	if len(forest.Roots) == 0 {
		t.Fatal("no loops in f")
	}
	last := forest.Roots[0]
	for _, lp := range forest.Roots[1:] {
		if lp.Header.Index > last.Header.Index {
			last = lp
		}
	}
	return last, fieldpath.Analyze(fn)
}

func TestRuleIncreasingContinuedIndex(t *testing.T) {
	// The CountInversions tail-loop shape: the second loop's init is the
	// merge loop's exit value — non-constant, provably >= 0 through the
	// two-phi cycle. Trips <= upper(len(a)) - init <= len(a).
	loop, stab := lastLoopOf(t, `package input
func f(a, b []int) int {
	i, j, t := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			i++
		} else {
			j++
		}
		t++
	}
	for i < len(a) {
		i++
		t++
	}
	return t
}`)
	got := Of(loop, stab)
	if got.String() != "O(len(a))" {
		t.Errorf("Of = %q, want O(len(a))", got.String())
	}
}

func TestRuleIncreasingParamInitStillRejected(t *testing.T) {
	// A parameter proves nothing about sign: `for i := m; i < len(a)` with
	// m = -1000000 runs len(a)+1000000 times. Must stay ⊤ (finding S1/B1).
	loop, stab := lastLoopOf(t, `package input
func f(a []int, m int) int {
	t := 0
	for i := m; i < len(a); i++ {
		t++
	}
	return t
}`)
	if got := Of(loop, stab); !got.IsTop() {
		t.Errorf("Of = %q, want Top — parameter init has no provable sign", got.String())
	}
}

func TestRuleIncreasingDerivedHalfInit(t *testing.T) {
	// The MaxSubarrayDC shape: the init is len(s)/2, non-constant but provably
	// >= 0 through the len and QUO arms. Trips <= upper(len(a)) - init.
	loop, stab := lastLoopOf(t, `package input
func f(a []int) int {
	mid := len(a) / 2
	t := 0
	for i := mid; i < len(a); i++ {
		t++
	}
	return t
}`)
	got := Of(loop, stab)
	if got.String() != "O(len(a))" {
		t.Errorf("Of = %q, want O(len(a))", got.String())
	}
}

func TestRuleIncreasingUnsignedDivisorRejected(t *testing.T) {
	// The divisor gate at the trip-count level. Both shapes have an init the
	// engine must NOT read as non-negative:
	//   - a variable divisor: k = -1 makes len(a)/k negative, and the loop
	//     then runs len(a) + len(a) times;
	//   - a negative constant divisor: len(a)/-2 is <= 0 and shrinks with n,
	//     so trips <= upper(len(a)) - init is not bounded by len(a).
	cases := []struct{ name, src string }{
		{"variable divisor", `package input
func f(a []int, k int) int {
	t := 0
	for i := len(a) / k; i < len(a); i++ {
		t++
	}
	return t
}`},
		{"negative const divisor", `package input
func f(a []int) int {
	t := 0
	for i := len(a) / -2; i < len(a); i++ {
		t++
	}
	return t
}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			loop, stab := lastLoopOf(t, c.src)
			if got := Of(loop, stab); !got.IsTop() {
				t.Errorf("Of = %q, want Top — divisor does not preserve the sign", got.String())
			}
		})
	}
}
