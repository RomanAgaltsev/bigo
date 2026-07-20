package edge

import "slices"

// Shapes that must verify (Within is silent — any diagnostic fails the test).

//bigo:max O(n)
func EarlyReturn(xs []int, t int) int {
	for i := 0; i < len(xs); i++ {
		if xs[i] == t {
			return i
		}
	}
	return -1
}

//bigo:max O(n)
func BreakOut(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		if xs[i] < 0 {
			break
		}
		s += xs[i]
	}
	return s
}

//bigo:max O(n^2)
func LabeledBreak(xs []int) int {
	s := 0
outer:
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs); j++ {
			if xs[j] < 0 {
				break outer
			}
			s++
		}
	}
	return s
}

//bigo:max O(n)
func SwitchInLoop(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		switch {
		case xs[i] > 0:
			s++
		case xs[i] < 0:
			s--
		}
	}
	return s
}

// Shapes that must be unverifiable — a bounded verdict here is the B1 bug.

//bigo:max O(n)
func InfiniteGEQ(n int) int { // want `cannot verify budget O\(n\)`
	s := 0
	for i := 0; i >= n; i++ {
		s++
	}
	return s
}

//bigo:max O(n)
func NegativeStep(n int) int { // want `cannot verify budget O\(n\)`
	s := 0
	for i := 0; i < n; i += -1 {
		s++
	}
	return s
}

//bigo:max O(n)
func ZeroStep(n int) int { // want `cannot verify budget O\(n\)`
	s := 0
	for i := 0; i < n; i += 0 {
		s++
	}
	return s
}

//bigo:max O(n) where n=b
func ParamStart(a, b int) int { // want `cannot verify budget O\(b\)`
	s := 0
	for i := a; i < b; i++ {
		s++
	}
	return s
}

// Triangular nests bound since the loop-algebra slice: the inner bound i is
// dominated by its own loop's guard (i < len(xs)).

//bigo:max O(n^2)
func Triangular(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		for j := 0; j < i; j++ {
			s++
		}
	}
	return s
}

// Irreducible control flow (goto into a cycle from two entries) has no
// natural loop; the function must be unverifiable, never O(1).

//bigo:max O(1)
func IrreducibleGoto(n int, c bool) int { // want `cannot verify budget O\(1\)`
	i := 0
	if c {
		goto b
	}
a:
	i++
b:
	i++
	if i < n {
		goto a
	}
	return i
}

// A variable offset in the loop condition shifts the trip count to (n-j).
// With j = -1000000 and n = 1 this runs a million times, so O(n) would be a
// wrong bound — the loop must be unverifiable.

//bigo:max O(n)
func OffsetCondition(n, j int) int { // want `cannot verify budget O\(n\)`
	s := 0
	for i := 0; i+j < n; i++ {
		s++
	}
	return s
}

// A closed-guard bisection whose hi update is `hi = mid` does NOT terminate:
// when lo == hi, mid == lo == hi, so hi never moves. R6 accepts the closed
// guard `lo <= hi` only when both ends move strictly past mid, which is why
// this must stay unverifiable. Deleting the c >= 1 condition in isHiUpdate
// makes this loop claim O(log n) — a wrong bound on a non-terminating loop.

//bigo:max O(log n) where n = len(s)
func ClosedBisectionHiEqMid(s []int, x int) int { // want `cannot verify budget O\(log\(len\(s\)\)\)`
	lo, hi := 0, len(s)-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if s[mid] < x {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return -1
}

// The terminating sibling, same guard: both ends move strictly past mid, so R6
// bounds it. Pins that the pin above fails for its stated reason (the hi
// update), not because the closed guard is rejected wholesale.

//bigo:max O(log n) where n = len(s)
func ClosedBisectionTerminates(s []int, x int) int {
	lo, hi := 0, len(s)-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if s[mid] < x {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return -1
}

// A two-pointer loop with a path that advances NEITHER pointer does not
// terminate: the measure i+j stalls. R7 bounds this shape only when every
// back-edge path advances exactly one pointer, which is why this stays
// unverifiable. Deleting that condition makes this loop claim
// O(len(a) + len(b)) — a wrong bound on a non-terminating loop.

//bigo:max O(n + m) where n = len(a), m = len(b)
func TwoPointerStalls(a, b []int) int { // want `cannot verify budget`
	i, j, n := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] < 0 {
			n++
			continue
		}
		if a[i] <= b[j] {
			i++
		} else {
			j++
		}
		n++
	}
	return n
}

// The terminating sibling: exactly one pointer advances per path, so R7 bounds
// it. Pins that the pin above fails for its stated reason (the stalling path),
// not because the two-pointer shape is rejected wholesale.

//bigo:max O(n + m) where n = len(a), m = len(b)
func TwoPointerMerges(a, b []int) int {
	i, j, n := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			i++
		} else {
			j++
		}
		n++
	}
	return n
}

// make([]T, 0, cap) has LENGTH 0 — its capacity is not its length. A size rule
// reading Cap here would claim O(len(s)) iterations for a loop that never runs:
// a wrong bound, not an imprecise one. Pins the direction. (⊤ is the honest
// answer today: len is the constant 0 and a constant-trip loop is a separate
// documented gap. What must never happen is O(len(s)).)

//bigo:max O(1)
func MakeZeroLenCap(s []int) int { // want `cannot verify budget O\(1\)`
	m := make([]int, 0, len(s))
	n := 0
	for i := 0; i < len(m); i++ {
		n++
	}
	return n
}

// The sibling: make([]T, len(s)) DOES have length len(s), so this is O(n).

//bigo:max O(n) where n = len(s)
func MakeFullLen(s []int) int {
	m := make([]int, len(s))
	n := 0
	for i := 0; i < len(m); i++ {
		n++
	}
	return n
}

// --- Guard-false edges that stay INSIDE the loop (v1.28.0 review, F1) ---
//
// In each shape below the `if` block IS the loop header, and BOTH of its edges
// are in-loop: the guard failing does not exit, it just takes the else branch
// and loops again. Every trip-count rule argues "the guard fails => the loop
// ends", so none of them may bound these. Through v1.28.0 all of them did —
// tripcount.Of checked that the guard's TRUE edge stays in the loop but never
// that its FALSE edge leaves it, which is a wrong bound (the fifth
// prime-directive break, live since v1.6.0).
//
// Neither the metrics golden nor the canonical corpus can see this family:
// corpus entries are well-formed textbook algorithms and a well-formed
// `for cond {}` always exits at its header. These pins are the only guard.
// Any NEW trip-count rule must add its own member here.

// R1 — increasing. Never terminates: i grows forever past len(a).

//bigo:max O(1)
func GuardFalseInLoopR1(a []int) int { // want `cannot verify budget O\(1\)`
	i, t := 0, 0
	for {
		if i < len(a) {
			t++
		} else {
			t += 2
		}
		i++
	}
}

// R1, TERMINATING — the case that matters. The real exit is `t >= limit`, so
// the trip count is limit/2 and has nothing to do with len(a). Reported
// O(len(a)) through v1.28.0: with a empty, a constant claimed for a loop that
// runs as long as limit says.

//bigo:max O(1)
func GuardFalseInLoopR1Terminating(a []int, limit int) int { // want `cannot verify budget O\(1\)`
	i, t := 0, 0
	for {
		if i < len(a) {
			t++
		} else {
			t += 2
		}
		i++
		if t >= limit {
			break
		}
	}
	return t
}

// R2 — decreasing.

//bigo:max O(1)
func GuardFalseInLoopR2(n int) int { // want `cannot verify budget O\(1\)`
	t := 0
	for {
		if n > 0 {
			t++
		} else {
			t += 2
		}
		n--
	}
}

// R3 — geometric up.

//bigo:max O(1)
func GuardFalseInLoopR3(n int) int { // want `cannot verify budget O\(1\)`
	i, t := 1, 0
	for {
		if i < n {
			t++
		} else {
			t += 2
		}
		i *= 2
	}
}

// R4 — geometric down.

//bigo:max O(1)
func GuardFalseInLoopR4(n int) int { // want `cannot verify budget O\(1\)`
	t := 0
	for {
		if n > 0 {
			t++
		} else {
			t += 2
		}
		n /= 2
	}
}

// R7 — two-pointer. Go lowers `if A && B { X } else { Y }` with BOTH failure
// paths jumping to the single else block, so R7's "same exit" conjunction check
// is satisfied by a block that is not an exit. Alternation passes too: the else
// advances i, the then advances j — exactly one per path. Never terminates.

//bigo:max O(1)
func GuardFalseInLoopR7(a, b []int) int { // want `cannot verify budget O\(1\)`
	i, j, t := 0, 0, 0
	for {
		if i < len(a) && j < len(b) {
			j++
		} else {
			i++
		}
		t++
	}
}

// The sibling that must still bound: same `for { if cond … else … }` shape as
// GuardFalseInLoopR1, but the else EXITS. R1 keeps its graduation. Pins that
// the six above fail for the stated reason — the in-loop false edge — and not
// because the header-if shape is rejected wholesale.

//bigo:max O(n) where n = len(a)
func GuardFalseExits(a []int) int {
	i, t := 0, 0
	for {
		if i < len(a) {
			t++
		} else {
			break
		}
		i++
	}
	return t
}

// ---- Sized cost-table arguments (shared-resolver slice, spec §8) ----
//
// The cost table now sizes locally-derived arguments via sizefacts. These pins
// hold the UPPER-bound direction at the call-cost layer: a Within on any of
// the three unverifiable shapes below is a wrong-direction resolution.

// Positive control: the append-copy idiom resolves and verifies silently.
//
//bigo:max O(n log n) where n=len(s)
func SortLocalCopy(s []int) []int {
	out := append([]int(nil), s...)
	slices.Sort(out)
	return out
}

// make([]T, 0, cap) has LENGTH zero; resolving the sort by the CAP would be
// the wrong direction. The engine cannot yet prove the O(1) truth (constant
// extents are unsupported), so the pin holds "not Within" — unverifiable.
//
//bigo:max O(1)
func SortEmptyMake(s []int) []int { // want `cannot verify budget O\(1\)`
	out := make([]int, 0, len(s))
	slices.Sort(out)
	return out
}

// s[0:cap(s)] can exceed len(s): costing the sort by len(s) would be a wrong
// bound. The sound resolution is the cap-derived extent, which cannot verify
// a len-based budget.
//
//bigo:max O(n log n) where n=len(s)
func SortCapSlice(s []int) []int { // want `cannot verify budget`
	out := s[0:cap(s)]
	slices.Sort(out)
	return out
}

// len(append(a, b...)) is len(a)+len(b) — inexpressible as one extent, so it
// must stay unresolved. Costing by len(b) alone would under-approximate.
//
//bigo:max O(n log n) where n=len(b)
func SortAppendTwo(a, b []int) []int { // want `cannot verify budget`
	out := append(a, b...)
	slices.Sort(out)
	return out
}

// ---- Non-constant loop inits (recurrence slice, spec §5) ----
//
// R1 now accepts an init that is non-constant but provably >= 0 AND enters
// from outside the loop. The two pins hold the two halves of that condition.

// Positive control: two sequential loops sharing an index — the second loop's
// init is the first's exit value, >= 0 by the coinductive invariant. Total
// O(len(s)); verifies silently.
//
//bigo:max O(n) where n=len(s)
func SequentialSharedIndex(s []int) int {
	i, t := 0, 0
	for i < len(s) {
		i++
		t++
	}
	for i < len(s) {
		i++
		t++
	}
	return t
}

// A loop-carried latch phi that is lower-boundable must NOT be read as an
// init: on the stalling path neither pointer advances, so len(a) alone does
// not bound the loop. The probe's first B2 attempt read the carried edge as an
// init and emitted exactly O(len(a)) — caught by R7's unit no-fire pin; this
// is the analyzer-level twin, budgeted at precisely that wrong bound.
//
//bigo:max O(n) where n=len(a)
func CarriedEdgeNotInit(a, b []int) int { // want `cannot verify budget`
	i, j, t := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] < 0 {
			t++
			continue
		}
		if a[i] <= b[j] {
			i++
		} else {
			j++
		}
	}
	return t
}

// --- Resetting back edges: a carried edge is NOT an init (review F1, #97) ---
//
// Each shape below has TWO back edges to the header: one that steps the
// induction variable and one that RESETS it. The reset edge is not a step, and
// through v1.30.0 R2/R3/R4 classified it as an "init" — so they bounded loops
// that never terminate. R1 was fixed in v1.30.0; its siblings were not, because
// that slice scoped the guard by what its corpus row needed rather than by
// which rules shared the hole.
//
// These use `continue`, NOT `goto`: two continues in an ordinary `for cond {}`
// body produce two back edges, which is the whole point — a reader who sees
// only a goto form will conclude the class is exotic and relax the guard.
//
// Neither golden can see this family: the corpus is well-formed textbook
// algorithms, and the guard-false (v1.28.1) and two-pointer-stall (v1.30.0)
// pins are single-back-edge shapes. Any NEW rule that splits phi edges into
// step/init must add its own member here.

// R2 — decreasing. Never terminates when a is false and b is true.
//
//bigo:max O(n) where n=len(s)
func ResetBackEdgeR2(s []int, a, b bool) int { // want `cannot verify budget`
	i, t := len(s), 0
	for i > 0 {
		t++
		if a {
			i--
			continue
		}
		if b {
			i = len(s)
			continue
		}
		i--
	}
	return t
}

// R2's terminating sibling: two continues, but EVERY back edge steps. Pins
// that the shape above fails for the reset, not for having two back edges.
//
//bigo:max O(n) where n=len(s)
func TwoContinuesBothStepR2(s []int, a bool) int {
	i, t := len(s), 0
	for i > 0 {
		t++
		if a {
			i -= 2
			continue
		}
		i--
	}
	return t
}

// R3 — geometric up. Never terminates when a is false and b is true.
//
//bigo:max O(log n) where n=n
func ResetBackEdgeR3(n int, a, b bool) int { // want `cannot verify budget`
	i, t := 1, 0
	for i < n {
		t++
		if a {
			i *= 2
			continue
		}
		if b {
			i = 1
			continue
		}
		i *= 3
	}
	return t
}

// R3's terminating sibling: both back edges multiply.
//
//bigo:max O(log n) where n=n
func TwoContinuesBothStepR3(n int, a bool) int {
	i, t := 1, 0
	for i < n {
		t++
		if a {
			i *= 2
			continue
		}
		i *= 3
	}
	return t
}

// R4 — geometric down. Never terminates when a is false and b is true.
//
//bigo:max O(log n) where n=n
func ResetBackEdgeR4(n int, a, b bool) int { // want `cannot verify budget`
	i, t := n, 0
	for i > 1 {
		t++
		if a {
			i /= 2
			continue
		}
		if b {
			i = n
			continue
		}
		i /= 3
	}
	return t
}

// R4's terminating sibling: both back edges divide.
//
//bigo:max O(log n) where n=n
func TwoContinuesBothStepR4(n int, a bool) int {
	i, t := n, 0
	for i > 1 {
		t++
		if a {
			i /= 2
			continue
		}
		i /= 3
	}
	return t
}

// --- v1.31.0: the non-negativity divisor gate (nonNegInvariant's QUO arm).
//
// The arm accepts `x / c` only for a CONSTANT c >= 1. These two pins hold that
// gate shut: each is budgeted at the bound the engine would emit if the gate
// were dropped, so relaxing it turns them red.

// A variable divisor proves nothing: k = -1 makes the init -len(s), and the
// loop runs 2*len(s) times.
//
//bigo:max O(n) where n=len(s)
func VarDivisorInit(s []int, k int) int { // want `cannot verify budget`
	t := 0
	for i := len(s) / k; i < len(s); i++ {
		t++
	}
	return t
}

// A negative constant divisor makes the init <= 0 and shrinking in n, so the
// trip count is not bounded by len(s).
//
//bigo:max O(n) where n=len(s)
func NegDivisorInit(s []int) int { // want `cannot verify budget`
	t := 0
	for i := len(s) / -2; i < len(s); i++ {
		t++
	}
	return t
}

// Positive control: the MaxSubarrayDC form. The gate must ACCEPT this one —
// a pin that only ever rejects would stay green if the arms were deleted.
//
//bigo:max O(n) where n=len(s)
func HalfDivisorInit(s []int) int {
	t := 0
	for i := len(s) / 2; i < len(s); i++ {
		t++
	}
	return t
}

// --- v1.32.0: C10's shift gate and C6's square gate.
//
// Each no-fire pin is budgeted at the bound the engine would emit if its gate
// were dropped, so relaxing either turns them red. The positive controls exist
// because no-fire pins alone would stay green if the arms were deleted outright.

// A zero shift is the identity — b never moves, so the loop does not terminate.
//
//bigo:max O(log n) where n=b
func ZeroShiftAmount(b int) int { // want `cannot verify budget`
	t := 0
	for b > 0 {
		b >>= 0
		t++
	}
	return t
}

// A variable shift amount may be 0 at runtime, proving no decrease.
//
//bigo:max O(log n) where n=b
func VarShiftAmount(b, k int) int { // want `cannot verify budget`
	t := 0
	for b > 0 {
		b >>= k
		t++
	}
	return t
}

// Positive control: the binary-exponentiation form must verify.
//
//bigo:max O(log n) where n=b
func ConstShift(b int) int {
	t := 0
	for b > 0 {
		b >>= 1
		t++
	}
	return t
}

// A distinct-operand product bounds neither factor: m may be tiny (or zero,
// or negative) while d runs far past n.
//
//bigo:max O(n) where n=n
func DistinctMulGuard(n, m int) int { // want `cannot verify budget`
	t := 0
	for d := 2; d*m <= n; d++ {
		t++
	}
	return t
}

// Positive control: the trial-division form must verify.
//
//bigo:max O(n) where n=n
func SquareGuard(n int) int {
	t := 0
	for d := 2; d*d <= n; d++ {
		t++
	}
	return t
}
