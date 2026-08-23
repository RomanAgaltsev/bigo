// Package engine composes an intraprocedural asymptotic time bound for a function.
package engine

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/costtable"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/loopnest"
)

// causeText distinguishes a priced callee whose argument size is unresolved
// from a callee with no cost at all. Read-only use of costtable for
// diagnostics; resolution still flows exclusively through CostModel.
func causeText(c *ssa.CallCommon, deferred bool) string {
	site := "call"
	if deferred {
		site = "deferred call"
	}
	if costtable.Priced(c) {
		return "unresolved argument size at " + site + " to " + calleeName(c)
	}
	return "unresolved cost at " + site + " to " + calleeName(c)
}

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

	// Callee is the cost-table key of the callee, for call and defer causes
	// with a static callee; empty otherwise.
	//
	// It is NOT a parse of What. What renders the callee with calleeName
	// (RelString), while this is costtable.CalleeKey, which resolves a generic
	// instantiation to its origin — so the two differ on generic code and must
	// never be joined against each other.
	//
	// Empty for interface dispatch and function values, which are exactly the
	// blockers no trust file can express, so consumers filter on presence
	// rather than keeping a second list of what is inexpressible.
	Callee string
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
	return InferTraced(fn, model, nil)
}

// InferTraced is InferDetailed plus an optional derivation trace. A nil trace
// leaves the walk exactly as it is without one: every tracing branch below is a
// nil check on a path that already runs per block.
//
// The trace is a BYPRODUCT of this walk, never a reconstruction of it. That is
// why `-report -explain` cannot print a derivation the verdict disagrees with.
func InferTraced(fn *ssa.Function, model CostModel, tr *Trace) (bound.Bound, []Cause) {
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
	lf := newLoopFactor(fn, stab, tr)
	var contribs []contribution
	for _, b := range fn.Blocks {
		factor := bound.Constant()
		var enclosing []token.Pos
		for _, lp := range forest.EnclosingLoops(b) {
			factor = factor.Mul(lf.of(lp, &causes))
			if tr != nil {
				enclosing = append(enclosing, loopPos(fn, lp))
			}
		}
		bc, bcauses := blockCost(b, model)
		causes = append(causes, bcauses...)
		contrib := bc.Mul(factor)
		if tr != nil {
			traceCalls(tr, b, model)
			contribs = append(contribs, contribution{cost: contrib, loops: enclosing})
		}
		if !started {
			total = contrib
			started = true
			continue
		}
		total = total.Join(contrib)
	}
	if !total.IsTop() {
		tr.attribute(total, contribs)
		return total, nil
	}
	return total, causes
}

// traceCalls records one CallStep per call-shaped instruction in b, using the
// model's own answer so the trace cannot diverge from the cost the walk used.
func traceCalls(tr *Trace, b *ssa.BasicBlock, model CostModel) {
	ex, _ := model.(CostExplainer)
	for _, instr := range b.Instrs {
		c, ok := instr.(*ssa.Call)
		if !ok {
			continue
		}
		step := CallStep{Pos: c.Pos(), Callee: calleeName(&c.Call)}
		if ex != nil {
			step.Cost, step.Source = ex.CallCostExplained(&c.Call)
		} else {
			step.Cost = model.CallCost(&c.Call)
		}
		tr.Calls = append(tr.Calls, step)
	}
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
				// The blank is deliberate: no key means the callee cannot be
				// named, which is the empty string, which is the signal.
				key, _ := costtable.CalleeKey(&v.Call)
				causes = append(causes, Cause{Pos: v.Pos(), Kind: CauseCall, What: causeText(&v.Call, false), Callee: key})
			}
			cost = cost.Join(c)
		case *ssa.Defer:
			c := model.CallCost(&v.Call)
			if c.IsTop() {
				key, _ := costtable.CalleeKey(&v.Call)
				causes = append(causes, Cause{Pos: v.Pos(), Kind: CauseDefer, What: causeText(&v.Call, true), Callee: key})
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
