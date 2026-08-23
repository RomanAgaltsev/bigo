package engine

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// CostExplainer is an OPTIONAL CostModel that also reports where a price came
// from. InferTraced type-asserts for it; a model that does not implement it is
// traced with no source tag rather than not traced at all.
//
// CostModel itself is deliberately NOT widened: it has six implementers, three
// of them test models, and widening would break all of them to serve one flag.
type CostExplainer interface {
	CallCostExplained(c *ssa.CallCommon) (bound.Bound, string)
}

// LoopStep is one loop's contribution: where it is, which rule bounded it, and
// the trip count that rule produced. Rule is "" when no rule matched, and a
// renderer must show that as "no rule matched" rather than inventing a name.
type LoopStep struct {
	Pos   token.Pos
	Rule  string
	Count bound.Bound
}

// CallStep is one priced call: where it is, what was called, what it cost, and
// WHICH SOURCE answered. Source is "" when the call is unresolved.
type CallStep struct {
	Pos    token.Pos
	Callee string
	Cost   bound.Bound
	Source string
}

// TermStep attributes one SURVIVING term of the final bound to the loops
// enclosing the block that produced it.
//
// Only surviving terms are recorded. bound.Join is a reduce: a block whose
// contribution is dominated contributes nothing to the result, so enumerating
// every block would enumerate things that provably did not matter. The term is
// located by its enclosing LOOPS, not by a source range reconstructed from SSA
// block positions, which would be a plausible-looking guess.
type TermStep struct {
	Term  string
	Loops []token.Pos
}

// Trace is the derivation `-report -explain` renders. A nil *Trace disables tracing
// entirely and leaves every walk exactly as it is without one.
//
// A Trace records what the REAL derivation did. Nothing here re-derives, so a
// trace cannot disagree with the verdict it accompanies.
type Trace struct {
	Loops []LoopStep
	Calls []CallStep
	Terms []TermStep
}

// contribution is one block's cost and the loops enclosing it, kept only long
// enough to attribute the surviving terms.
type contribution struct {
	cost  bound.Bound
	loops []token.Pos
}

// attribute fills tr.Terms by finding, for each surviving term of total, a
// block contribution that produced it verbatim.
func (tr *Trace) attribute(total bound.Bound, contribs []contribution) {
	if tr == nil || total.IsTop() {
		return
	}
	for _, m := range total.Terms() {
		want := m.String()
		for _, c := range contribs {
			if c.cost.IsTop() {
				continue
			}
			found := false
			for _, cm := range c.cost.Terms() {
				if cm.String() == want {
					found = true
					break
				}
			}
			if found {
				tr.Terms = append(tr.Terms, TermStep{Term: want, Loops: c.loops})
				break
			}
		}
	}
}
