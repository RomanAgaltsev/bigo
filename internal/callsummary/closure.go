package callsummary

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// SSA shape (recorded per plan Task 4 dump). For an in-scope closure such as
// `each(xs, func(i int){ _ = xs[0] + i })`, the argument is an *ssa.MakeClosure
// whose Fn is the anonymous *ssa.Function (e.g. "Use$1") and whose Bindings are
// parallel to Fn.FreeVars. Go captures by REFERENCE: a captured slice `xs`
// appears as a FreeVar of type *[]int, and the matching Binding is an
// *ssa.Alloc spill of the outer variable (never a bare *ssa.Parameter). Reads
// in the body go through a load of the free var.
//
// Consequence for sizing: the plain summary of a closure whose body loops over
// a captured value is ⊤ — sizefacts recognizes only *ssa.Parameter roots, and
// a captured *ssa.FreeVar (pointer to slice) is not one. Renaming free-var
// sizes into caller vocabulary would require teaching the soundness-critical
// size/sizefacts core to treat free vars as size roots (and guarding the
// resulting leak in substArgs). That precision is DEFERRED; see closureCost.

// closureCost prices one invocation of an in-scope closure. It is sound but
// deliberately narrow: it accepts a closure only when its body summary is
// finite AND carries no residual size variable. That covers the headline case
// — the sort.Slice/each comparator that captures a slice purely for O(1)
// index reads (the capture never appears in the body's bound). Everything
// else refuses (⊤):
//
//   - a body that loops over a captured value summarizes to ⊤ already, because
//     free-var sizes are not canonical size roots (see the file comment);
//   - a body whose cost depends on the closure's OWN parameters cannot be
//     priced here either — the values the consumer feeds the closure are
//     invisible at this site (same refusal as a size-dependent static func
//     argument in resolveFuncArg).
//
// SOUNDNESS: a finite, var-free body summary is a genuine O(1)-per-invocation
// cost regardless of what the closure captured, so multiplying it by the
// invocation count at the call site never undercounts.
func (r *Resolver) closureCost(mc *ssa.MakeClosure) (bound.Bound, bool) {
	closureFn, ok := mc.Fn.(*ssa.Function)
	if !ok {
		return bound.Top(), false
	}
	s := r.summary(closureFn)
	if s.IsTop() {
		return bound.Top(), false
	}
	for _, m := range s.Terms() {
		if len(m.Vars()) > 0 {
			return bound.Top(), false // own-param or capture size dependence: unpriceable here
		}
	}
	return s, true
}
