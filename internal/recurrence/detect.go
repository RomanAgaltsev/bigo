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
		if !terminates(fn, p, calls) {
			continue // integer measure without a base guard, etc.
		}
		work, ok := localWork(fn, model)
		if !ok {
			return rec{}, false
		}
		if !varsSubset(work, measureVar(p)) {
			return rec{}, false // multivariate recurrence: out of scope
		}
		return rec{measure: measureVar(p), param: p, terms: terms, work: work}, true
	}
	return rec{}, false
}

// stepsFor classifies the pi-th argument of every self-call against parameter
// p. Returns the strict steps (Sub/Div) when every call is a strict step or an
// unchanged pass-through with at least one strict step; else ok=false.
func stepsFor(p *ssa.Parameter, pi int, calls []*ssa.CallCommon) ([]sizeStep, bool) {
	var terms []sizeStep
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

// terminates proves the recursion is well-founded. Slice measures are
// structurally well-founded (len >= 0, strictly decreasing, empty-slice base).
// Integer measures require a base guard: every self-call block must be
// dominated by an If whose condition compares the measure parameter against a
// constant such that the recursion only fires while the measure exceeds a
// floor.
func terminates(fn *ssa.Function, p *ssa.Parameter, calls []*ssa.CallCommon) bool {
	if isSliceLike(p.Type()) {
		return true
	}
	for _, c := range calls {
		if !guardedByMeasure(callBlock(fn, c), p) {
			return false
		}
	}
	return true
}

// guardedByMeasure walks the dominator chain of blk looking for an If whose
// condition constrains the measure parameter p below a constant floor on the
// side that reaches blk. A base guard on a non-measure condition, or one whose
// recursing side does NOT lower-bound p, correctly fails to prove termination.
func guardedByMeasure(blk *ssa.BasicBlock, p *ssa.Parameter) bool {
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
		if boundsMeasureBelow(cmp, p, onTrue) {
			return true
		}
	}
	return false
}

// boundsMeasureBelow reports whether cmp — a comparison of measure parameter p
// against a constant — constrains p to exceed a constant floor on the recursing
// side (recurseOnTrue = the recursion is reached from the If's true successor).
// A lower bound on a strictly decreasing measure is what proves termination.
func boundsMeasureBelow(cmp *ssa.BinOp, p *ssa.Parameter, recurseOnTrue bool) bool {
	op, ok := measureCmpOp(cmp, p)
	if !ok {
		return false
	}
	if !recurseOnTrue {
		op = negateOp(op)
	}
	return op == token.GTR || op == token.GEQ
}

// measureCmpOp returns cmp's operator normalized with parameter p as the LEFT
// operand and a constant on the right, or ok=false when cmp is neither
// `p <cmp> const` nor `const <cmp> p`.
func measureCmpOp(cmp *ssa.BinOp, p *ssa.Parameter) (token.Token, bool) {
	switch {
	case cmp.X == ssa.Value(p):
		if _, ok := sizefacts.ConstIntV(cmp.Y); ok {
			return cmp.Op, true
		}
	case cmp.Y == ssa.Value(p):
		if _, ok := sizefacts.ConstIntV(cmp.X); ok {
			return swapOp(cmp.Op), true
		}
	}
	return token.ILLEGAL, false
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
