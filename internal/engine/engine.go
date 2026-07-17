// Package engine composes an intraprocedural asymptotic time bound for a function.
package engine

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
	"github.com/RomanAgaltsev/bigo/internal/tripcount"
)

// CostModel resolves the cost of a call in canonical size variables.
type CostModel interface {
	CallCost(c *ssa.CallCommon) bound.Bound
}

// CauseKind is the machine-readable category of an unverifiable cause. The
// metrics harness buckets on Kind; diagnostics render What. Never bucket on
// What — it is presentation text.
type CauseKind int

const (
	// CauseCall - unresolved cost at a call
	CauseCall CauseKind = iota
	// CauseDefer - unresolved cost at a deferred call
	CauseDefer
	// CauseGo - goroutine launch
	CauseGo
	// CauseLoop - loop with unrecognized trip count
	CauseLoop
	// CauseIrreducible - irreducible control flow
	CauseIrreducible
	// CauseNoBody - function has no analyzable body
	CauseNoBody
)

func (k CauseKind) String() string {
	switch k {
	case CauseCall:
		return "call"
	case CauseDefer:
		return "defer"
	case CauseGo:
		return "go"
	case CauseLoop:
		return "loop"
	case CauseIrreducible:
		return "irreducible"
	case CauseNoBody:
		return "nobody"
	default:
		return "unknown"
	}
}

// Cause records why a bound became unverifiable: the source position, the
// machine-readable kind, and a human-readable description.
type Cause struct {
	Pos  token.Pos
	Kind CauseKind
	What string
}

// Infer returns the function's intraprocedural time bound, delegating call
// costs to model. Model: Σ_blocks blockCost(b) × Π(trip-counts of enclosing
// loops); ⊤ is absorbing, so any ⊤ anywhere makes the function ⊤.
func Infer(fn *ssa.Function, model CostModel) bound.Bound {
	b, _ := InferDetailed(fn, model)
	return b
}

// InferDetailed is Infer plus the reasons the bound (when ⊤) is unverifiable.
// Causes are nil when the bound is not ⊤.
func InferDetailed(fn *ssa.Function, model CostModel) (bound.Bound, []Cause) {
	if fn == nil || len(fn.Blocks) == 0 {
		return bound.Top(), []Cause{{Kind: CauseNoBody, What: "function has no analyzable body"}}
	}
	forest := loopnest.Build(fn)
	if forest.UncoveredCycle(fn) {
		return bound.Top(), []Cause{{Pos: fn.Pos(), Kind: CauseIrreducible, What: "irreducible control flow (goto into a cycle)"}}
	}
	stab := fieldpath.Analyze(fn)

	var causes []Cause
	total := bound.Constant()
	started := false
	for _, b := range fn.Blocks {
		factor := bound.Constant()
		for _, lp := range forest.EnclosingLoops(b) {
			tc := tripcount.Of(lp, stab)
			if tc.IsTop() {
				causes = append(causes, Cause{
					Pos:  lp.Header.Instrs[len(lp.Header.Instrs)-1].Pos(),
					Kind: CauseLoop,
					What: "loop with unrecognized trip count",
				})
			}
			factor = factor.Mul(tc)
		}
		bc, bcauses := blockCost(b, model)
		causes = append(causes, bcauses...)
		contrib := bc.Mul(factor)
		if !started {
			total = contrib
			started = true
			continue
		}
		total = total.Join(contrib)
	}
	if !total.IsTop() {
		return total, nil
	}
	return total, causes
}

// blockCost is O(1) plus the model's cost for each call-shaped instruction.
// Deferred calls are joined like plain calls: they all run at function exit,
// and the enclosing-loop factor applied by InferDetailed upper-bounds "one
// deferred call per iteration". A go statement makes the block unverifiable —
// v1 does not model concurrent work (spec §9).
func blockCost(b *ssa.BasicBlock, model CostModel) (bound.Bound, []Cause) {
	cost := bound.Constant()
	var causes []Cause
	for _, instr := range b.Instrs {
		switch v := instr.(type) {
		case *ssa.Call:
			c := model.CallCost(&v.Call)
			if c.IsTop() {
				causes = append(causes, Cause{Pos: v.Pos(), Kind: CauseCall, What: "unresolved cost at call to " + calleeName(&v.Call)})
			}
			cost = cost.Join(c)
		case *ssa.Defer:
			c := model.CallCost(&v.Call)
			if c.IsTop() {
				causes = append(causes, Cause{Pos: v.Pos(), Kind: CauseDefer, What: "unresolved cost at deferred call to " + calleeName(&v.Call)})
			}
			cost = cost.Join(c)
		case *ssa.Go:
			causes = append(causes, Cause{Pos: v.Pos(), Kind: CauseGo, What: "goroutine launch (concurrency is unverifiable in v1)"})
			return bound.Top(), causes
		}
	}
	return cost, causes
}

// calleeName is a best-effort human-readable name for a call target.
//
// Names are qualified: a function carries its package ("time.Now"), a method its
// receiver type ("(*sync.Mutex).Lock"), and an interface dispatch its interface
// ("(io.Writer).Write"). A bare name is ambiguous in the one place this text is
// read — "call to Close" could be self-recursion, a stdlib call, or delegation —
// and made delegation to a same-named callee look like recursion (issue #47).
//
// A dynamically-called function value has no static target to qualify, so it
// keeps its SSA value name.
func calleeName(c *ssa.CallCommon) string {
	if c.Method != nil {
		// Interface dispatch: FullName renders "(pkg.Iface).Method".
		return c.Method.FullName()
	}
	if f := c.StaticCallee(); f != nil {
		// RelString(nil) renders "pkg/path.Func" and "(*pkg.T).Method".
		return f.RelString(nil)
	}
	if c.Value != nil {
		if n := c.Value.Name(); n != "" {
			return n
		}
	}
	return "unknown callee"
}
