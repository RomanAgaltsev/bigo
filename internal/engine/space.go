package engine

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/size"
	"github.com/RomanAgaltsev/bigo/internal/sizefacts"
	"github.com/RomanAgaltsev/bigo/internal/tripcount"
)

// SpaceModel resolves the space cost of a call in canonical size variables.
type SpaceModel interface {
	CallSpace(c *ssa.CallCommon) bound.Bound
}

// Space splits a function's asymptotic space by soundness class. Heap is an
// UPPER bound on peak live memory (total allocated — never a lower bound);
// Stack is the true peak recursion depth (filled by the recurrence slice).
type Space struct {
	Heap, Stack bound.Bound
}

// InferSpace returns the function's heap upper bound (this slice) and an empty
// Stack (O(1)); recursive stack depth is injected by the analyzer, which — unlike
// engine — may import recurrence. Model: Σ_blocks blockAlloc(b) × Π(trip-counts of
// enclosing loops), joined; ⊤ is absorbing, so any ⊤ allocation makes Heap ⊤.
func InferSpace(fn *ssa.Function, model SpaceModel) (Space, []Cause) {
	if fn == nil || len(fn.Blocks) == 0 {
		return Space{Heap: bound.Top(), Stack: bound.Constant()},
			[]Cause{{Kind: CauseNoBody, What: "function has no analyzable body"}}
	}
	forest := loopnest.Build(fn)
	if forest.UncoveredCycle(fn) {
		return Space{Heap: bound.Top(), Stack: bound.Constant()},
			[]Cause{{Pos: fn.Pos(), Kind: CauseIrreducible, What: "irreducible control flow (goto into a cycle)"}}
	}
	stab := fieldpath.Analyze(fn)
	var causes []Cause
	total, started := bound.Constant(), false
	for _, b := range fn.Blocks {
		bc, allocated, bcauses := blockAlloc(b, model, stab)
		causes = append(causes, bcauses...)
		// A block that allocates nothing contributes O(1) total heap regardless
		// of enclosing loops — the loop repeats work, not allocation. Only a
		// block that actually allocates is scaled by its loop trips (a per-
		// iteration allocation accumulates as a safe over-approximation of peak).
		contrib := bc
		if allocated {
			factor := bound.Constant()
			for _, lp := range forest.EnclosingLoops(b) {
				factor = factor.Mul(tripFactor(lp, stab, &causes))
			}
			contrib = bc.Mul(factor)
		}
		if !started {
			total, started = contrib, true
			continue
		}
		total = total.Join(contrib)
	}
	sp := Space{Heap: total, Stack: bound.Constant()}
	if total.IsTop() {
		return sp, causes
	}
	return sp, nil
}

// tripFactor is InferDetailed's per-loop trip-count computation, extracted so
// the space walk records the same CauseLoop diagnostic on an unrecognized loop.
func tripFactor(lp *loopnest.Loop, stab *fieldpath.Stability, causes *[]Cause) bound.Bound {
	tc := tripcount.Of(lp, stab)
	if tc.IsTop() {
		*causes = append(*causes, Cause{
			Pos:  lp.Header.Instrs[len(lp.Header.Instrs)-1].Pos(),
			Kind: CauseLoop,
			What: "loop with unrecognized trip count",
		})
	}
	return tc
}

// blockAlloc scores one block's allocation and reports whether the block
// allocates at all. MakeSlice(n) -> O(n); heap Alloc, MakeMap, MakeChan -> O(1);
// append(a, b...) -> O(len(b)) when b is a sized value, else O(1) per call (a
// loop factor scales it); any other call -> model.CallSpace; a go statement
// makes the block ⊤ (concurrent alloc is unverifiable in v1). The allocated flag
// lets InferSpace leave a non-allocating block at O(1) instead of multiplying it
// by an enclosing loop's trip count. Allocations within a straight-line block
// Join (their max dominates asymptotically).
func blockAlloc(b *ssa.BasicBlock, model SpaceModel, stab *fieldpath.Stability) (bound.Bound, bool, []Cause) {
	cost := bound.Constant()
	allocated := false
	var causes []Cause
	for _, instr := range b.Instrs {
		switch v := instr.(type) {
		case *ssa.MakeSlice:
			allocated = true
			if sv, ok := size.Value(v.Len); ok {
				cost = cost.Join(bound.Of(bound.Term(sv)))
			} else if fv, ok := stab.VarFor(v.Len); ok {
				cost = cost.Join(bound.Of(bound.Term(fv)))
			} else if lv, ok := (&sizefacts.Facts{Stab: stab}).ArgSize(v.Len); ok {
				// make([]T, len(s)/2): a derived length is still a length.
				cost = cost.Join(bound.Of(bound.Term(lv)))
			} else if !isConstLen(v.Len) {
				causes = append(causes, Cause{Pos: v.Pos(), Kind: CauseCall, What: "make with unknown length"})
				return bound.Top(), true, causes
			}
		case *ssa.Alloc:
			allocated = true
			cost = cost.Join(bound.Constant())
		case *ssa.MakeMap, *ssa.MakeChan:
			allocated = true
			cost = cost.Join(bound.Constant())
		case *ssa.MapUpdate:
			// A map assign is amortized O(1) allocation: inserting k distinct
			// keys grows the table to O(k) total, so charging O(1) per assign and
			// letting InferSpace scale it by the enclosing loop trips upper-bounds
			// the growth — exactly how append is costed. Without this, a map built
			// to the size of its input inferred O(1) heap and passed an O(1) space
			// budget silently (issue #49).
			//
			// Sound in the over-approximating direction: an assign to an existing
			// key allocates nothing, so this can over-charge. Heap is an upper
			// bound on peak and drives Within only (SpaceVerdict takes Exceeds
			// from Stack alone), so over-charging can only turn a Within into
			// "cannot verify" — a false negative, never a false Exceeds.
			allocated = true
			cost = cost.Join(bound.Constant())
		case *ssa.Call:
			c, alloc := callSpaceOf(v, model, stab, &causes)
			allocated = allocated || alloc
			cost = cost.Join(c)
		case *ssa.Defer:
			allocated = true
			cost = cost.Join(model.CallSpace(&v.Call))
		case *ssa.Go:
			causes = append(causes, Cause{Pos: v.Pos(), Kind: CauseGo, What: "goroutine allocation is unverifiable in v1"})
			return bound.Top(), true, causes
		}
	}
	return cost, allocated, causes
}

// callSpaceOf resolves a call's allocation and reports whether it allocates.
// append is the one allocating builtin (go/ssa lowers every append to
// append(a, b...), packing scalar varargs into a fresh slice); other builtins
// (len, cap, copy, ...) allocate nothing. A non-builtin call delegates to the
// space model and is treated as allocating (its body may allocate O(1) the
// resolver reports as Constant), so a call inside a loop is conservatively
// scaled by the loop trips.
func callSpaceOf(c *ssa.Call, model SpaceModel, stab *fieldpath.Stability, causes *[]Cause) (bound.Bound, bool) {
	if bi, ok := c.Call.Value.(*ssa.Builtin); ok {
		if bi.Name() == "append" {
			return appendSpace(c, stab), true
		}
		return bound.Constant(), false
	}
	sp := model.CallSpace(&c.Call)
	if sp.IsTop() {
		*causes = append(*causes, Cause{Pos: c.Pos(), Kind: CauseCall, What: "unresolved space at call to " + calleeName(&c.Call)})
	}
	return sp, true
}

// appendSpace scores append(a, b...): O(len(b)) when the spread argument is a
// sized value (a slice parameter or a stable field read) — appending a whole
// slice adds len(b) elements — else O(1) per call, the shape go/ssa emits for a
// scalar append(a, x) after packing x into a one-element varargs slice.
func appendSpace(c *ssa.Call, stab *fieldpath.Stability) bound.Bound {
	args := c.Call.Args
	if len(args) < 2 {
		return bound.Constant()
	}
	last := args[len(args)-1]
	if sv, ok := size.Value(last); ok {
		return bound.Of(bound.Term(sv))
	}
	if fv, ok := stab.VarFor(last); ok {
		return bound.Of(bound.Term(fv))
	}
	// A locally-derived spread (append(nil, s[:mid]...), a make, a slice
	// expression) adds its length in elements. Without this the per-level
	// copies of a solved divisive recursion charged O(1) heap — an
	// oracle-confirmed wrong space bound (#87 probe, B3).
	f := &sizefacts.Facts{Stab: stab}
	if lv, ok := f.ArgSize(last); ok {
		return bound.Of(bound.Term(lv))
	}
	return bound.Constant()
}

// isConstLen reports whether an SSA length operand is a compile-time constant,
// which contributes O(1) heap (a loop factor, if any, scales it).
func isConstLen(v ssa.Value) bool {
	_, ok := v.(*ssa.Const)
	return ok
}

// SpaceVerdict applies the heap/stack asymmetry: stack (a real peak) can prove
// Within and Exceeds; heap (an upper bound on peak) proves Within only. So a
// budget can only be Exceeded on the stack term, never on heap over-approximation.
func SpaceVerdict(sp Space, budget bound.Bound) bound.Verdict {
	if bound.Check(sp.Stack, budget) == bound.Exceeds {
		return bound.Exceeds
	}
	if bound.Check(sp.Heap.Join(sp.Stack), budget) == bound.Within {
		return bound.Within
	}
	return bound.Unknown
}
