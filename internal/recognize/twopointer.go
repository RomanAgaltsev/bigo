package recognize

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/size"
	"github.com/RomanAgaltsev/bigo/internal/sizefacts"
)

// twoPointer matches an outer loop guarded by a comparison of two integer phis
// that move only toward each other: one never decreases, the other never
// increases, and neither is assigned any other way anywhere in the loop.
//
// The pair therefore makes at most (hi_init - lo_init) moves in total, so the
// loop is linear in hi's initial extent even though NEITHER loop individually
// has a provable trip count. That is exactly why proof cannot reach this shape
// and why it belongs in the advisory channel.
func twoPointer(fn *ssa.Function) []Recognition {
	forest := loopnest.Build(fn)
	facts := &sizefacts.Facts{Stab: fieldpath.Analyze(fn)}
	var out []Recognition
	for _, root := range forest.Roots {
		if r, ok := matchTwoPointer(root, facts); ok {
			out = append(out, r)
		}
	}
	return out
}

func matchTwoPointer(loop *loopnest.Loop, facts *sizefacts.Facts) (Recognition, bool) {
	if len(loop.Header.Instrs) == 0 {
		return Recognition{}, false
	}
	ifi, ok := loop.Header.Instrs[len(loop.Header.Instrs)-1].(*ssa.If)
	if !ok {
		return Recognition{}, false
	}
	cmp, ok := ifi.Cond.(*ssa.BinOp)
	if !ok || (cmp.Op != token.LSS && cmp.Op != token.LEQ) {
		return Recognition{}, false
	}
	lo, ok := cmp.X.(*ssa.Phi)
	if !ok {
		return Recognition{}, false
	}
	hi, ok := cmp.Y.(*ssa.Phi)
	if !ok {
		return Recognition{}, false
	}
	if lo == hi {
		return Recognition{}, false
	}
	if !movesOneWay(lo, loop, token.ADD) || !movesOneWay(hi, loop, token.SUB) {
		return Recognition{}, false
	}

	loNet, hiNet := network(lo), network(hi)
	loScans, hiScans := scansFor(loop, loNet), scansFor(loop, hiNet)

	// AMENDMENT 1 (A3): the nest conditions. These are a WRONG-BOUND guard, not
	// a no-fire nicety. R8 thresholds 3 lists "an inner loop that does not
	// advance a pointer, BOUNDED" as multiplying the nest bound per iteration
	// while R-A would still advertise O(N).
	if !nestAllScans(loop, loNet, hiNet) || !latchAdvances(loop, loScans, hiScans) {
		return Recognition{}, false
	}

	// AMENDMENT 1 (A2): the two indices must index a COMMON collection, and it
	// must be a slice, array or pointer-to-array; a map index does not qualify,
	// there being no memory bound on a map key. This is spec 5.1's condition
	// and R8 thresholds 1.4; it is also what licenses the in-range clause in
	// the rendered assumption below. Without it R-A fires on any converging
	// integer pair and the PATTERN NAME is false even where the bound is true.
	collName, ok := commonIndexed(loScans, hiScans, loNet, hiNet)
	if !ok {
		return Recognition{}, false
	}

	// hi must enter the loop from a single place. Two different entry values
	// mean two different starting distances, and neither names the measure.
	init, ok := outsideInit(hi, loop)
	if !ok {
		return Recognition{}, false
	}
	// The extent of hi's initial value is not required to be nameable — the
	// bound is len(C), not the starting distance — but a value the size
	// machinery cannot see at all is a shape we have not understood, so it is
	// refused rather than bounded.
	if _, ok := facts.UpperExtent(init, 0); !ok {
		return Recognition{}, false
	}
	// The bound is len(C), per spec 5.1, and NOT hi's initial extent. For the
	// real partition those diverge: hi enters as the parameter `right`, so the
	// starting-distance rendering would be O(right) while R8 measured — and the
	// spec states — O(len(participants)). Both are true, and len(C) is the one
	// that says "linear in the collection" to a learner. It is sound for the
	// same reason the in-range clause is: an index past len(C) panics rather
	// than completes, so memory safety caps the moves at len(C) even when hi
	// starts higher.
	return Recognition{
		// The guard's own position, not the *ssa.If's: control-flow
		// instructions carry token.NoPos, and a recognition with no position
		// cannot be reported against the loop it describes.
		Pos:     cmp.Pos(),
		Pattern: "amortized two-pointer scan",
		Kind:    KindAmortized,
		Bound:   "O(" + string(collName) + ")",
		// AMENDMENT 1 (A1, A4): the monotonicity clause justifies TERMINATION;
		// the in-range clause is what justifies the BOUND (R8 thresholds 1:
		// "Go memory safety is the bound"). The trailing clause is A4, decided
		// as Option 1 on 2026-08-22: the advance precondition is semantic and
		// unverifiable, which an ADVISORY recognition may rest on and a proved
		// bound may not. Do not trim it; see Amendment 1 A4.
		Assumption: "both indices advance monotonically toward each other and " +
			"never reset, and both stay in range because the scan guards read " +
			string(collName) + " at the current index, so the pair makes at most " +
			string(collName) + " moves in total; and no outer iteration passes " +
			"without advancing a pointer (true when the comparison is strict " +
			"on distinct elements)",
	}, true
}

// movesOneWay reports whether every in-loop definition of phi moves it in the
// given direction by a positive constant, following phi chains so a move made
// in a NESTED loop still counts. Any other assignment disqualifies the shape.
//
// The edge is judged by the PREDECESSOR BLOCK it arrives from, not by where its
// value was defined. R8 thresholds 1 puts the rejection exactly there: "a reset
// (p = 0 inside the loop) is an edge from an in-loop block that is neither step
// nor member". A constant has no defining block at all, so a defining-block test
// silently accepts `left = 0` as if it were the loop's initial value — which is
// how a resetting pointer would have been handed an amortized bound.
func movesOneWay(phi *ssa.Phi, loop *loopnest.Loop, dir token.Token) bool {
	seen := map[*ssa.Phi]bool{}
	var okPhi func(p *ssa.Phi) bool
	var okVal func(v ssa.Value) bool

	// okVal validates a value arriving along an in-loop edge.
	okVal = func(v ssa.Value) bool {
		switch t := v.(type) {
		case *ssa.Phi:
			return okPhi(t)
		case *ssa.BinOp:
			if t.Op != dir {
				return false
			}
			c, isConst := t.Y.(*ssa.Const)
			if !isConst {
				return false
			}
			n, exact := sizefacts.ConstIntV(c)
			return exact && n > 0 && okVal(t.X)
		default:
			// Defined outside the loop, so it cannot change during the loop and
			// is not a reset. Anything else arriving from inside — a constant,
			// a load, a call result — releases the measure.
			d, isInstr := v.(ssa.Instruction)
			return isInstr && !loop.Blocks[d.Block()]
		}
	}
	okPhi = func(p *ssa.Phi) bool {
		if seen[p] {
			return true
		}
		seen[p] = true
		preds := p.Block().Preds
		if len(preds) != len(p.Edges) {
			return false
		}
		for i, e := range p.Edges {
			if !loop.Blocks[preds[i]] {
				continue // an entry edge: this is the initial value
			}
			if !okVal(e) {
				return false
			}
		}
		return true
	}
	return okPhi(phi)
}

// outsideInit returns phi's single edge defined outside the loop.
func outsideInit(phi *ssa.Phi, loop *loopnest.Loop) (ssa.Value, bool) {
	var init ssa.Value
	for _, e := range phi.Edges {
		d, isInstr := e.(ssa.Instruction)
		if isInstr && loop.Blocks[d.Block()] {
			continue
		}
		if init != nil && init != e {
			return nil, false // two different entries: refuse
		}
		init = e
	}
	return init, init != nil
}

// network returns every value in phi's definition network: the phi itself, the
// phis its edges reach, and the step BinOps between them.
//
// Walking edges BACKWARDS reaches the phis of nested loops, because an outer
// header phi's back-edge operand is the value the inner loops produced. That is
// what lets the scan-loop phis be recognized as the same pointer.
func network(phi *ssa.Phi) map[ssa.Value]bool {
	seen := map[ssa.Value]bool{}
	var walk func(v ssa.Value)
	walk = func(v ssa.Value) {
		if v == nil || seen[v] {
			return
		}
		seen[v] = true
		switch t := v.(type) {
		case *ssa.Phi:
			for _, e := range t.Edges {
				walk(e)
			}
		case *ssa.BinOp:
			walk(t.X)
		}
	}
	walk(phi)
	return seen
}

// carriesPhiIn reports whether l's header carries a phi belonging to net, which
// is what makes l a scan advancing that pointer.
func carriesPhiIn(l *loopnest.Loop, net map[ssa.Value]bool) bool {
	for _, in := range l.Header.Instrs {
		if p, ok := in.(*ssa.Phi); ok && net[p] {
			return true
		}
	}
	return false
}

// scansFor returns the loops nested directly in loop that advance net's pointer.
func scansFor(loop *loopnest.Loop, net map[ssa.Value]bool) []*loopnest.Loop {
	out := make([]*loopnest.Loop, 0, len(loop.Children))
	for _, c := range loop.Children {
		if carriesPhiIn(c, net) {
			out = append(out, c)
		}
	}
	return out
}

// nestAllScans is R8 thresholds 1.2: EVERY loop nested anywhere in loop must be
// a pointer-advancing scan. An inner loop that advances neither pointer
// multiplies the real cost per outer iteration while the pair's move count
// stays the same, so the whole nest must be refused rather than mis-bounded.
func nestAllScans(loop *loopnest.Loop, loNet, hiNet map[ssa.Value]bool) bool {
	for _, c := range loop.Children {
		if !carriesPhiIn(c, loNet) && !carriesPhiIn(c, hiNet) {
			return false
		}
		if !nestAllScans(c, loNet, hiNet) {
			return false
		}
	}
	return true
}

// latchAdvances is R8 thresholds 1.3: every path through one outer iteration
// must pass an advancing scan's header, so that the outer loop cannot spin
// without the pair moving.
func latchAdvances(loop *loopnest.Loop, loScans, hiScans []*loopnest.Loop) bool {
	if len(loScans) == 0 || len(hiScans) == 0 {
		return false
	}
	for _, p := range loop.Header.Preds {
		if !loop.Blocks[p] {
			continue // an entry edge, not a latch
		}
		if !dominatedByAny(p, loScans) || !dominatedByAny(p, hiScans) {
			return false
		}
	}
	return true
}

func dominatedByAny(b *ssa.BasicBlock, scans []*loopnest.Loop) bool {
	for _, s := range scans {
		if dominates(s.Header, b) {
			return true
		}
	}
	return false
}

// dominates reports whether a dominates b by walking b's immediate-dominator
// chain. A block dominates itself.
func dominates(a, b *ssa.BasicBlock) bool {
	for x := b; x != nil; x = x.Idom() {
		if x == a {
			return true
		}
	}
	return false
}

// commonIndexed is R8 thresholds 1.4: the advancing scans' guards must index a
// COMMON collection at the current pointer values, and its length must be
// nameable in the bound grammar.
//
// This is what supplies the bound's range argument. Each pointer can move at
// most len(C)+1 times in a completing run, because the scan guard reads C at
// the new index before the next move can occur, and an out-of-range index
// panics rather than completes. Go memory safety is the bound.
func commonIndexed(loScans, hiScans []*loopnest.Loop, loNet, hiNet map[ssa.Value]bool) (bound.Var, bool) {
	loColl, ok := indexedIn(loScans, loNet)
	if !ok {
		return "", false
	}
	hiColl, ok := indexedIn(hiScans, hiNet)
	if !ok {
		return "", false
	}
	if loColl != hiColl {
		return "", false
	}
	p, ok := loColl.(*ssa.Parameter)
	if !ok {
		return "", false // not nameable in the bound grammar
	}
	return size.Len(p.Name()), true
}

// indexedIn returns the single collection the scans index at a pointer value of
// net, refusing when they index more than one or index something whose range is
// not memory-checked.
func indexedIn(scans []*loopnest.Loop, net map[ssa.Value]bool) (ssa.Value, bool) {
	var found ssa.Value
	for _, s := range scans {
		for _, in := range s.Header.Instrs {
			ia, ok := in.(*ssa.IndexAddr)
			if !ok || !net[ia.Index] {
				continue
			}
			if !rangeChecked(ia.X.Type()) {
				return nil, false
			}
			if found != nil && found != ia.X {
				return nil, false
			}
			found = ia.X
		}
	}
	return found, found != nil
}

// rangeChecked reports whether indexing t is bounds-checked against a length,
// which is what makes memory safety a usable bound. A map is deliberately
// excluded: a map key carries no range.
func rangeChecked(t types.Type) bool {
	switch u := t.Underlying().(type) {
	case *types.Slice, *types.Array:
		return true
	case *types.Pointer:
		_, isArr := u.Elem().Underlying().(*types.Array)
		return isArr
	}
	return false
}
