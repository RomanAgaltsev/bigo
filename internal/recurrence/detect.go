package recurrence

// Recorded SSA shapes (go/ssa dump, Task 3 Step 1 — the source of truth):
//
//	f(xs[1:])       block: t4 = slice xs[1:int:]            *ssa.Slice{X:xs, Low:const 1, High:nil}
//	f(xs[:m])       m=len/2: t13 = slice xs[:t3]            *ssa.Slice{X:xs, Low:nil, High:t3}
//	                         t3 = t2 / 2:int                *ssa.BinOp{QUO, X:len(xs), Y:const 2}
//	f(xs[m+1:])     m=len/2: t11 = slice xs[t10:]           *ssa.Slice{X:xs, Low:t10, High:nil}
//	                         t10 = t3 + 1:int               *ssa.BinOp{ADD, X:(len/2), Y:const 1}
//	f(n-1)          t1 = n - 1:int                          *ssa.BinOp{SUB, X:n(param), Y:const 1}
//	f(n/2)          t = n / 2:int                           *ssa.BinOp{QUO, X:n(param), Y:const 2}
//	f(n+1)          t1 = n + 1:int                          *ssa.BinOp{ADD, ...}  (growth -> rejected)
//
// Guard shape for a guarded integer measure `if n <= 0 { return 0 }; ...f(n-1)`:
//	block 0: t0 = n <= 0:int; if t0 goto 1 else 2   (Succs=[base, recurse])
// The recursing block sits on the FALSE side of `n <= 0`; equivalently the
// recursion fires only while n > 0. A base guard on the false side of a `<=`
// test is therefore a valid lower-bound floor — the polarity is handled in
// boundsMeasureBelow, not by hard-coding which successor recurses.
//
// Subtractive and divisive steps have different well-foundedness obligations: a
// subtractive measure halts at any floor (the arithmetic sequence crosses it),
// but a divisive measure has a fixed point at 0 (0/b == 0), so it graduates only
// when the recursing side proves the measure >= 1 — see boundsMeasureBelow and
// guardedBySliceBase.

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/size"
	"github.com/RomanAgaltsev/bigo/internal/sizefacts"
)

type stepKind int

const (
	stepBad  stepKind = iota
	stepSub           // arg is measure - c (c >= 1)
	stepDiv           // arg is measure / b (b >= 2)
	stepSame          // arg is the measure unchanged
)

type sizeStep struct {
	kind     stepKind
	sub, div int64
}

// rec is an extracted self-recurrence in the measure variable.
type rec struct {
	measure bound.Var
	param   *ssa.Parameter // the measure parameter
	terms   []sizeStep     // one strict step per self-call (stepSame filtered out)
	work    bound.Bound    // f(n)
	// mult is the branching factor a: the maximum number of self-calls executed
	// on any single path (not the static call count). Calls in mutually
	// exclusive branches — binary search's two arms — count once; sequential
	// calls — merge sort's two halves — count twice.
	mult int
}

// extract builds the recurrence, or ok=false when any soundness precondition
// fails: no measure, a non-decreasing/growing arg, a self-call under a size
// loop, an unguarded integer measure, or a ⊤ body.
func extract(fn *ssa.Function, model engine.CostModel) (rec, bool) {
	calls := selfCalls(fn)
	if len(calls) == 0 {
		return rec{}, false
	}
	// Constant multiplicity: no self-call may sit inside an enclosing loop.
	forest := loopnest.Build(fn)
	for _, c := range calls {
		if underLoop(forest, callBlock(fn, c)) {
			return rec{}, false
		}
	}
	// Pick the measure parameter: the first parameter every self-call strictly
	// steps (Sub or Div) and none grows.
	for pi, p := range fn.Params {
		terms, ok := stepsFor(p, pi, calls)
		if !ok {
			continue
		}
		if !terminates(fn, p, terms, calls) {
			continue // integer measure without a base guard, etc.
		}
		work, ok := localWork(fn, model)
		if !ok {
			return rec{}, false
		}
		if !varsSubset(work, measureVar(p)) {
			return rec{}, false // multivariate recurrence: out of scope
		}
		return rec{
			measure: measureVar(p),
			param:   p,
			terms:   terms,
			work:    work,
			mult:    selfCallMult(fn, calls),
		}, true
	}
	return rec{}, false
}

// selfCallMult returns the branching factor a: the maximum number of self-calls
// on any single entry→exit path. Self-calls in one basic block are sequential
// and all count; across blocks, a call counts toward another only when one
// block reaches the other (they lie on a common path). Because self-calls never
// sit in a loop (extract rejects those), reachability among distinct call
// blocks is a strict partial order, so the longest weighted chain is the answer.
func selfCallMult(fn *ssa.Function, calls []*ssa.CallCommon) int {
	perBlock := map[*ssa.BasicBlock]int{}
	var blocks []*ssa.BasicBlock
	for _, c := range calls {
		b := callBlock(fn, c)
		if b == nil {
			continue
		}
		if perBlock[b] == 0 {
			blocks = append(blocks, b)
		}
		perBlock[b]++
	}
	memo := map[*ssa.BasicBlock]int{}
	var longest func(b *ssa.BasicBlock) int
	longest = func(b *ssa.BasicBlock) int {
		if v, ok := memo[b]; ok {
			return v
		}
		down := 0
		for _, other := range blocks {
			if other != b && reaches(b, other) {
				if d := longest(other); d > down {
					down = d
				}
			}
		}
		res := perBlock[b] + down
		memo[b] = res
		return res
	}
	best := 0
	for _, b := range blocks {
		if v := longest(b); v > best {
			best = v
		}
	}
	return best
}

// stepsFor classifies the pi-th argument of every self-call against parameter
// p. Returns the strict steps (Sub/Div) when every call is a strict step or an
// unchanged pass-through with at least one strict step; else ok=false.
func stepsFor(p *ssa.Parameter, pi int, calls []*ssa.CallCommon) ([]sizeStep, bool) {
	terms := make([]sizeStep, 0, len(calls))
	hasStrict := false
	for _, c := range calls {
		if pi >= len(c.Args) {
			return nil, false
		}
		st := sizeStepOf(c.Args[pi], p)
		switch st.kind {
		case stepBad:
			return nil, false
		case stepSame:
			// pass-through: fine, contributes no term
		default:
			hasStrict = true
			terms = append(terms, st)
		}
	}
	if !hasStrict {
		return nil, false
	}
	return terms, true
}

// sizeStepOf classifies arg's size relation to parameter p. Slice measures use
// len; integer measures use value. Anything unrecognized is stepBad.
func sizeStepOf(arg ssa.Value, p *ssa.Parameter) sizeStep {
	if isSliceLike(p.Type()) {
		return sliceStep(arg, p)
	}
	if sizefacts.IsInteger(p.Type()) {
		return intStep(arg, p)
	}
	return sizeStep{kind: stepBad}
}

// sliceStep matches p[low:], p[:high], p[low:high] on parameter p. len = high-low.
//
//	low positive const & high nil          -> Sub by low.
//	low nil & high == len(p)/b (floor div) -> Div by b.
//	low nil & high == len(p) - c           -> Sub by c.
//	low >= len(p)/2 & high nil              -> Div by 2 (half-split; kept <= len/2).
//
// The half-split (binary search / merge sort's xs[m+1:]) is the only variable
// low recognized in v1: the removed prefix is at least half, so the kept suffix
// is bounded above by len/2, a sound b=2 divisive step. Other fractions and
// growing/unrecognized reslices stay stepBad.
func sliceStep(arg ssa.Value, p *ssa.Parameter) sizeStep {
	if arg == ssa.Value(p) {
		return sizeStep{kind: stepSame}
	}
	// The append-copy idiom: len(append(zero, x...)) == len(x) EXACTLY, so the
	// copy's step relation to p is x's. The zero-length gate is load-bearing —
	// a decrease claim needs that equality, and a non-zero dst (or the growth
	// append(a, b...)) must stay stepBad.
	if c, ok := arg.(*ssa.Call); ok {
		if b, isBuiltin := c.Call.Value.(*ssa.Builtin); isBuiltin && b.Name() == "append" && len(c.Call.Args) == 2 {
			if sizefacts.ZeroLen(c.Call.Args[0]) {
				return sliceStep(c.Call.Args[1], p)
			}
		}
	}
	sl, ok := arg.(*ssa.Slice)
	if !ok || sl.X != ssa.Value(p) {
		return sizeStep{kind: stepBad}
	}
	// p[low:] with low a positive const, high nil  => len - low
	if lo, ok := sizefacts.ConstIntV(sl.Low); ok && lo >= 1 && sl.High == nil {
		return sizeStep{kind: stepSub, sub: lo}
	}
	// p[:high] with high == len(p)/b  => len / b; or high == len(p) - c => len - c
	if sl.Low == nil {
		if b, ok := divOfLen(sl.High, p); ok {
			return sizeStep{kind: stepDiv, div: b}
		}
		if c, ok := lenMinusConst(sl.High, p); ok {
			return sizeStep{kind: stepSub, sub: c}
		}
	}
	// p[low:] with low >= len(p)/2 and high nil => kept suffix <= len/2 => Div by 2.
	if sl.High == nil && isHalfOfLen(sl.Low, p) {
		return sizeStep{kind: stepDiv, div: 2}
	}
	return sizeStep{kind: stepBad}
}

// intStep matches p-c (c>=1 const) and p/b (b>=2 const) on integer parameter p.
func intStep(arg ssa.Value, p *ssa.Parameter) sizeStep {
	if arg == ssa.Value(p) {
		return sizeStep{kind: stepSame}
	}
	bo, ok := arg.(*ssa.BinOp)
	if !ok || bo.X != ssa.Value(p) {
		return sizeStep{kind: stepBad}
	}
	c, ok := sizefacts.ConstIntV(bo.Y)
	if !ok {
		return sizeStep{kind: stepBad}
	}
	switch bo.Op {
	case token.SUB:
		if c >= 1 {
			return sizeStep{kind: stepSub, sub: c}
		}
	case token.QUO:
		if c >= 2 {
			return sizeStep{kind: stepDiv, div: c}
		}
	case token.ADD:
		// p + c is a GROWTH (or p-|c|); reject either way for a clean measure.
	}
	return sizeStep{kind: stepBad}
}

func measureVar(p *ssa.Parameter) bound.Var {
	if isSliceLike(p.Type()) {
		return size.Len(p.Name())
	}
	return size.Num(p.Name())
}

// terminates proves the recursion is well-founded. The proof obligation depends
// on the step kind, because a DIVISIVE measure has a fixed point at 0 that a
// SUBTRACTIVE one does not: Go integer division truncates toward zero, so
// 0/b == 0 and |n|/b < |n| only while |n| >= 1, and a divisive slice step
// xs[:len/2] on an empty slice is xs[:0] — still empty, no panic.
//
//   - Subtractive slice (xs[1:]): structurally well-founded — shrinking to empty
//     eventually panics on xs[0]/xs[1:], which halts. No guard required.
//   - Divisive slice (xs[:len/2]): requires a base guard proving len(p) >= 1 on
//     the recursing side (a dominating len==0 / len<c / len<=c base), else it
//     stalls at an empty slice with no base.
//   - Integer measure (subtractive or divisive): requires a base guard. A
//     subtractive step needs any lower-bound floor (an arithmetic sequence
//     crosses it); a divisive step needs the recursing side to prove measure >= 1.
func terminates(fn *ssa.Function, p *ssa.Parameter, terms []sizeStep, calls []*ssa.CallCommon) bool {
	strictDiv := hasDivStep(terms)
	if isSliceLike(p.Type()) {
		if !strictDiv {
			return true // subtractive slice: the empty-slice panic is the base
		}
		for _, c := range calls {
			if !guardedBySliceBase(callBlock(fn, c), p) {
				return false
			}
		}
		return true
	}
	for _, c := range calls {
		if !guardedByMeasure(callBlock(fn, c), p, strictDiv) {
			return false
		}
	}
	return true
}

// hasDivStep reports whether any recorded step divides the measure — the trigger
// for the stricter, fixed-point-aware termination proof.
func hasDivStep(terms []sizeStep) bool {
	for _, t := range terms {
		if t.kind == stepDiv {
			return true
		}
	}
	return false
}

// guardedByMeasure walks the dominator chain of blk looking for an If whose
// condition constrains the measure parameter p on the side that reaches blk. A
// base guard on a non-measure condition, or one whose recursing side does NOT
// lower-bound p as required by strictDiv, correctly fails to prove termination.
func guardedByMeasure(blk *ssa.BasicBlock, p *ssa.Parameter, strictDiv bool) bool {
	if blk == nil {
		return false
	}
	for d := blk.Idom(); d != nil; d = d.Idom() {
		ifi, ok := lastInstr(d).(*ssa.If)
		if !ok || len(d.Succs) != 2 {
			continue
		}
		cmp, ok := ifi.Cond.(*ssa.BinOp)
		if !ok {
			continue
		}
		onTrue := reaches(d.Succs[0], blk)
		onFalse := reaches(d.Succs[1], blk)
		if onTrue == onFalse {
			continue // blk reachable from both sides or neither: no clean guard
		}
		if boundsMeasureBelow(cmp, p, onTrue, strictDiv) {
			return true
		}
	}
	return false
}

// boundsMeasureBelow reports whether cmp — a comparison of measure parameter p
// against a constant — constrains p enough on the recursing side (recurseOnTrue
// = the recursion is reached from the If's true successor) to prove the measure
// is well-founded. For a SUBTRACTIVE measure any lower-bound floor halts the
// arithmetic sequence, so the constant's value is immaterial. For a DIVISIVE
// measure the sequence is stuck at the fixed point 0 unless the recursing side
// proves measure >= 1, so the floor's value is load-bearing: `n > k` gives
// n >= k+1 (need k >= 0) and `n >= k` needs k >= 1. Thus `n > 0` / `n >= 1`
// graduate while the unsound `n >= 0`, `n > -5` are rejected. A `measure == 0`
// base (recursing side measure != 0) also graduates: division truncates the
// magnitude toward zero to exactly 0, which the base then catches — this is the
// canonical power-by-squaring / fast-exponentiation shape.
func boundsMeasureBelow(cmp *ssa.BinOp, p *ssa.Parameter, recurseOnTrue, strictDiv bool) bool {
	op, k, ok := measureCmpOp(cmp, p)
	if !ok {
		return false
	}
	if !recurseOnTrue {
		op = negateOp(op)
	}
	if strictDiv {
		return (op == token.GTR && k >= 0) ||
			(op == token.GEQ && k >= 1) ||
			(op == token.NEQ && k == 0)
	}
	return op == token.GTR || op == token.GEQ
}

// measureCmpOp returns cmp's operator normalized with parameter p as the LEFT
// operand and the constant on the other side, or ok=false when cmp is neither
// `p <cmp> const` nor `const <cmp> p`.
func measureCmpOp(cmp *ssa.BinOp, p *ssa.Parameter) (token.Token, int64, bool) {
	switch {
	case cmp.X == ssa.Value(p):
		if k, ok := sizefacts.ConstIntV(cmp.Y); ok {
			return cmp.Op, k, true
		}
	case cmp.Y == ssa.Value(p):
		if k, ok := sizefacts.ConstIntV(cmp.X); ok {
			return swapOp(cmp.Op), k, true
		}
	}
	return token.ILLEGAL, 0, false
}

// guardedBySliceBase walks the dominator chain of blk looking for an If on
// len(p) that proves len(p) >= 1 on the side reaching blk — the base guard a
// DIVISIVE slice recursion needs, because xs[:len/2] on an empty (or one-element)
// slice is a fixed point (xs[:0] stays empty) with no panic and no base. A
// dominating len==0 / len<c / len<=c base, or the equivalent len>k / len>=k
// recurse guard, satisfies it.
func guardedBySliceBase(blk *ssa.BasicBlock, p *ssa.Parameter) bool {
	if blk == nil {
		return false
	}
	for d := blk.Idom(); d != nil; d = d.Idom() {
		ifi, ok := lastInstr(d).(*ssa.If)
		if !ok || len(d.Succs) != 2 {
			continue
		}
		cmp, ok := ifi.Cond.(*ssa.BinOp)
		if !ok {
			continue
		}
		onTrue := reaches(d.Succs[0], blk)
		onFalse := reaches(d.Succs[1], blk)
		if onTrue == onFalse {
			continue
		}
		if sliceLenBoundsNonEmpty(cmp, p, onTrue) {
			return true
		}
	}
	return false
}

// sliceLenBoundsNonEmpty reports whether cmp — a comparison of len(p) against a
// constant — keeps len(p) >= 1 on the recursing side. len(p) is always >= 0, so
// `len > k` (need k >= 0), `len >= k` (need k >= 1), and `len != 0` (a len==0
// base; >= 0 with != 0 is >= 1) all prove the divisive step cannot stall.
func sliceLenBoundsNonEmpty(cmp *ssa.BinOp, p *ssa.Parameter, recurseOnTrue bool) bool {
	op, k, ok := lenCmpOp(cmp, p)
	if !ok {
		return false
	}
	if !recurseOnTrue {
		op = negateOp(op)
	}
	switch op {
	case token.GTR:
		return k >= 0
	case token.GEQ:
		return k >= 1
	case token.NEQ:
		return k == 0
	}
	return false
}

// lenCmpOp is measureCmpOp for a slice measure: it matches len(p) against a
// constant, normalizing len(p) to the LEFT operand.
func lenCmpOp(cmp *ssa.BinOp, p *ssa.Parameter) (token.Token, int64, bool) {
	switch {
	case isLenOf(cmp.X, p):
		if k, ok := sizefacts.ConstIntV(cmp.Y); ok {
			return cmp.Op, k, true
		}
	case isLenOf(cmp.Y, p):
		if k, ok := sizefacts.ConstIntV(cmp.X); ok {
			return swapOp(cmp.Op), k, true
		}
	}
	return token.ILLEGAL, 0, false
}

// swapOp reflects a comparison operator across its operands (a<b <=> b>a).
func swapOp(op token.Token) token.Token {
	switch op {
	case token.LSS:
		return token.GTR
	case token.LEQ:
		return token.GEQ
	case token.GTR:
		return token.LSS
	case token.GEQ:
		return token.LEQ
	}
	return op
}

// negateOp returns the operator of the logical negation of `a op b`.
func negateOp(op token.Token) token.Token {
	switch op {
	case token.LSS:
		return token.GEQ
	case token.LEQ:
		return token.GTR
	case token.GTR:
		return token.LEQ
	case token.GEQ:
		return token.LSS
	case token.EQL:
		return token.NEQ
	case token.NEQ:
		return token.EQL
	}
	return token.ILLEGAL
}

// isSliceLike reports whether t's underlying type is a slice.
func isSliceLike(t types.Type) bool {
	_, ok := t.Underlying().(*types.Slice)
	return ok
}

// isLenOf reports whether v is the builtin call len(p).
func isLenOf(v ssa.Value, p *ssa.Parameter) bool {
	call, ok := v.(*ssa.Call)
	if !ok {
		return false
	}
	b, ok := call.Call.Value.(*ssa.Builtin)
	if !ok || b.Name() != "len" || len(call.Call.Args) != 1 {
		return false
	}
	return call.Call.Args[0] == ssa.Value(p)
}

// divOfLen matches len(p) / b for a constant b >= 2, returning b.
func divOfLen(v ssa.Value, p *ssa.Parameter) (int64, bool) {
	bo, ok := v.(*ssa.BinOp)
	if !ok || bo.Op != token.QUO || !isLenOf(bo.X, p) {
		return 0, false
	}
	b, ok := sizefacts.ConstIntV(bo.Y)
	if !ok || b < 2 {
		return 0, false
	}
	return b, true
}

// lenMinusConst matches len(p) - c for a constant c >= 1, returning c.
func lenMinusConst(v ssa.Value, p *ssa.Parameter) (int64, bool) {
	bo, ok := v.(*ssa.BinOp)
	if !ok || bo.Op != token.SUB || !isLenOf(bo.X, p) {
		return 0, false
	}
	c, ok := sizefacts.ConstIntV(bo.Y)
	if !ok || c < 1 {
		return 0, false
	}
	return c, true
}

// isHalfOfLen reports whether v equals len(p)/2, optionally plus a non-negative
// constant — so that removing v elements from the front keeps at most len/2 (a
// sound b=2 split). A negative offset is rejected: it could keep more than half.
func isHalfOfLen(v ssa.Value, p *ssa.Parameter) bool {
	if bo, ok := v.(*ssa.BinOp); ok && bo.Op == token.ADD {
		if c, ok := sizefacts.ConstIntV(bo.Y); ok && c >= 0 {
			v = bo.X
		} else if c, ok := sizefacts.ConstIntV(bo.X); ok && c >= 0 {
			v = bo.Y
		}
	}
	b, ok := divOfLen(v, p)
	return ok && b == 2
}

// underLoop reports whether blk sits inside any natural loop of its function —
// the constant-multiplicity guard against self-calls under a size loop.
func underLoop(forest *loopnest.Forest, blk *ssa.BasicBlock) bool {
	return blk != nil && len(forest.EnclosingLoops(blk)) > 0
}

// callBlock returns the basic block containing the call whose common is cc, by
// pointer identity with the common selfCalls recorded.
func callBlock(fn *ssa.Function, cc *ssa.CallCommon) *ssa.BasicBlock {
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			if callCommon(instr) == cc {
				return b
			}
		}
	}
	return nil
}

// lastInstr returns b's final instruction, or nil for an empty block.
func lastInstr(b *ssa.BasicBlock) ssa.Instruction {
	if len(b.Instrs) == 0 {
		return nil
	}
	return b.Instrs[len(b.Instrs)-1]
}

// reaches reports whether target is reachable from start by following successor
// edges (start counts as reaching itself).
func reaches(start, target *ssa.BasicBlock) bool {
	seen := map[*ssa.BasicBlock]bool{}
	stack := []*ssa.BasicBlock{start}
	for len(stack) > 0 {
		b := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if b == target {
			return true
		}
		if seen[b] {
			continue
		}
		seen[b] = true
		stack = append(stack, b.Succs...)
	}
	return false
}

// varsSubset reports whether every variable appearing in b is the allowed
// measure variable — the single-variable (univariate recurrence) requirement.
func varsSubset(b bound.Bound, allowed bound.Var) bool {
	for _, m := range b.Terms() {
		for _, v := range m.Vars() {
			if v != allowed {
				return false
			}
		}
	}
	return true
}
