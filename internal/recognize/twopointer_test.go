package recognize

import (
	"strings"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func detectIn(t *testing.T, src, fname string) []Recognition {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, fname)
	if fn == nil {
		t.Fatalf("%s not found", fname)
	}
	return Detect(fn)
}

// The shape from the real kata: an outer guard over two indices that only
// ever move toward each other, with the moves in INNER loops.
const twoPointerSrc = `package input
func f(xs []int, left, right int) int {
	swaps := 0
	for left < right {
		for xs[left] < 0 {
			left++
		}
		for xs[right] > 0 {
			right--
		}
		if left < right {
			xs[left], xs[right] = xs[right], xs[left]
			swaps++
		}
	}
	return swaps
}`

func TestTwoPointerRecognized(t *testing.T) {
	rs := detectIn(t, twoPointerSrc, "f")
	if len(rs) != 1 {
		t.Fatalf("want exactly 1 recognition, got %d: %+v", len(rs), rs)
	}
	r := rs[0]
	if r.Pattern != "amortized two-pointer scan" {
		t.Errorf("Pattern = %q", r.Pattern)
	}
	if r.Kind != KindAmortized {
		t.Errorf("Kind = %q, want %q", r.Kind, KindAmortized)
	}
	// The bound is the COLLECTION's length, per spec 5.1, not hi's starting
	// distance. For the real partition those diverge (hi enters as the
	// parameter `right`), and len(C) is both what R8 measured and what says
	// "linear in the collection" to a learner.
	if r.Bound != "O(len(xs))" {
		t.Errorf("Bound = %q, want O(len(xs))", r.Bound)
	}
	if !strings.Contains(r.Assumption, "monotonically") {
		t.Errorf("Assumption must state the monotonicity it rests on, got %q", r.Assumption)
	}
	// AMENDMENT 1 (A1): termination is not the bound. The rendered assumption
	// must also name the memory-safety hook the bound actually rests on.
	if !strings.Contains(r.Assumption, "in range") {
		t.Errorf("Assumption must state why the indices stay in range, got %q", r.Assumption)
	}
	// AMENDMENT 1 (A2): the in-range clause is only meaningful if it names the
	// collection whose length bounds the pair.
	if !strings.Contains(r.Assumption, "len(xs)") {
		t.Errorf("Assumption must name the collection, got %q", r.Assumption)
	}
	// AMENDMENT 1 (A4, decided Option 1): the claim rests on a precondition bigo
	// cannot verify, that the comparison is strict on distinct elements. That is
	// legitimate only while the assumption travels with it. Trimming this clause
	// would silently turn an honest advisory claim into an unqualified one, so
	// the test fails rather than the prose merely asking.
	if !strings.Contains(r.Assumption, "advancing a pointer") {
		t.Errorf("Assumption must state the advance precondition, got %q", r.Assumption)
	}
	if !r.Pos.IsValid() {
		t.Error("Pos must point at the loop that justifies the claim")
	}
}

// The REAL target, reduced to its shape: effectiveQuicksort.partition from the
// owner's sprint3/final2 solution. It differs from twoPointerSrc in the ways
// that matter for matching -- the scan guards call a comparator on the indexed
// element rather than comparing it inline, and the swap sits behind a guard --
// so a recognizer that fires on the tidy fixture but not on this one would have
// bought nothing. R-A's whole purpose is this function.
const partitionSrc = `package input

type Participant struct {
	Login string
	Score int
}

func less(a, b Participant) int {
	if a.Score != b.Score {
		if a.Score > b.Score {
			return -1
		}
		return 1
	}
	return 0
}

func f(participants []Participant, left, right int) int {
	pivot := participants[left]
	for left < right {
		for less(participants[left], pivot) == -1 {
			left++
		}
		for less(pivot, participants[right]) == -1 {
			right--
		}
		if left < right {
			participants[left], participants[right] = participants[right], participants[left]
		}
	}
	return right
}`

func TestTwoPointerRecognizesTheRealPartition(t *testing.T) {
	rs := detectIn(t, partitionSrc, "f")
	if len(rs) != 1 {
		t.Fatalf("want exactly 1 recognition on the real partition shape, got %d: %+v", len(rs), rs)
	}
	r := rs[0]
	if r.Bound != "O(len(participants))" {
		t.Errorf("Bound = %q, want O(len(participants))", r.Bound)
	}
	if r.Kind != KindAmortized {
		t.Errorf("Kind = %q, want %q", r.Kind, KindAmortized)
	}
	if !strings.Contains(r.Assumption, "len(participants)") {
		t.Errorf("Assumption must name the collection, got %q", r.Assumption)
	}
}

// Near-miss fixtures. A recognizer that fires on these is worse than one that
// never fires: the channel's whole value is that a named pattern IS that
// pattern.
//
// Every fixture here is written in the SCAN shape unless it is specifically
// testing the absence of one, so that it reaches the gate it is named for
// rather than being refused earlier for an unrelated reason. A near-miss test
// that passes for the wrong reason is not evidence.
func TestTwoPointerRefusesNearMisses(t *testing.T) {
	cases := map[string]string{
		// movesOneWay: a reset releases the measure.
		"left resets inside the loop": `package input
func f(xs []int, left, right int) int {
	n := 0
	for left < right {
		for xs[left] < 0 {
			left++
		}
		for xs[right] > 0 {
			right--
		}
		if xs[left] == 0 {
			left = 0
		}
		n++
	}
	return n
}`,
		// movesOneWay: the pair does not converge, so there is no measure.
		"both indices increase": `package input
func f(xs []int, left, right int) int {
	n := 0
	for left < right {
		for xs[left] < 0 {
			left++
		}
		for xs[right] < 0 {
			right++
		}
		n++
	}
	return n
}`,
		// The guard must compare the two moving values themselves.
		"guard is not a comparison of the two moving values": `package input
func f(xs []int, left, right int) int {
	n := 0
	for n < len(xs) {
		left++
		right--
		n++
	}
	return n
}`,
		// AMENDMENT 1 (A2): no collection is indexed at all. The move count is
		// still bounded by the initial distance, so the BOUND would be true,
		// but "amortized two-pointer scan" would be a false name for a numeric
		// bisection, and there is no memory-safety hook to state. Refusing is
		// the point.
		"no collection is indexed": `package input
func f(left, right int) int {
	n := 0
	for left < right {
		left++
		right--
		n++
	}
	return n
}`,
		// AMENDMENT 1 (A2): two scans, two DIFFERENT collections, so no common
		// measure and no shared range argument.
		"scans index different collections": `package input
func f(xs, ys []int, left, right int) int {
	n := 0
	for left < right {
		for xs[left] < 0 {
			left++
		}
		for ys[right] > 0 {
			right--
		}
		n++
	}
	return n
}`,
		// AMENDMENT 1 (A2): a map index carries no memory bound on its key, so
		// memory safety cannot supply the range argument (R8 thresholds 1.4).
		"map index does not qualify": `package input
func f(m map[int]int, left, right int) int {
	n := 0
	for left < right {
		for m[left] < 0 {
			left++
		}
		for m[right] > 0 {
			right--
		}
		n++
	}
	return n
}`,
		// AMENDMENT 1 (A3): the WRONG-BOUND vector. The extra inner loop
		// advances neither pointer and is bounded, so it multiplies the real
		// cost by len(xs) per outer iteration while the pair still makes at
		// most len(xs) moves. R-A must refuse the whole nest.
		"nested loop that advances neither pointer": `package input
func f(xs []int, left, right int) int {
	n := 0
	for left < right {
		for k := range xs {
			n += k
		}
		for xs[left] < 0 {
			left++
		}
		for xs[right] > 0 {
			right--
		}
	}
	return n
}`,
	}
	for name, src := range cases {
		if rs := detectIn(t, src, "f"); len(rs) != 0 {
			t.Errorf("%s: expected no recognition, got %+v", name, rs)
		}
	}
}
