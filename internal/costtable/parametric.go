package costtable

import (
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/size"
)

// ParamEntry is a curated cost for a stdlib function that takes a callback.
// Base is the cost independent of the callback; PerArg[i] upper-bounds how many
// times the callback at argument index i is invoked. The total at a call site
// is Base ⊔ Σ PerArg[i](args) × cost(one invocation of args[i]) — the last
// factor is supplied by the caller (only it can price a func value). All bounds
// are in caller size vocabulary. Counts are documented-contract upper bounds.
type ParamEntry struct {
	Base   func(args []ssa.Value) bound.Bound
	PerArg map[int]func(args []ssa.Value) bound.Bound
}

// LookupParametric returns the curated entry for a static callback-taking
// stdlib call, looking through generic instantiations to their origin.
// ok=false when the callee is not curated.
func LookupParametric(c *ssa.CallCommon) (ParamEntry, bool) {
	callee := c.StaticCallee()
	if callee == nil {
		return ParamEntry{}, false
	}
	if orig := callee.Origin(); orig != nil {
		callee = orig
	}
	if callee.Pkg == nil || callee.Pkg.Pkg == nil {
		return ParamEntry{}, false
	}
	key := callee.Pkg.Pkg.Path() + "." + callee.Name()
	e, ok := parametric[key]
	return e, ok
}

// argSizeVar resolves the size variable of a callback-function's collection
// argument, seeing through the *ssa.MakeInterface wrapper an `any` parameter
// (sort.Slice's x) imposes and through a read-only closure's captured-slice
// spill (fieldpath.SpillArgSize). Falls back to the plain classification.
func argSizeVar(v ssa.Value) (bound.Var, bool) {
	if sv, _, ok := fieldpath.SpillArgSize(v); ok {
		return sv, true
	}
	if mi, ok := v.(*ssa.MakeInterface); ok {
		v = mi.X
	}
	return size.Value(v)
}

func linearP(args []ssa.Value, i int) bound.Bound {
	if i >= len(args) {
		return bound.Top()
	}
	if v, ok := argSizeVar(args[i]); ok {
		return bound.Of(bound.Term(v))
	}
	return bound.Top()
}

func nLogNP(args []ssa.Value, i int) bound.Bound {
	if i >= len(args) {
		return bound.Top()
	}
	if v, ok := argSizeVar(args[i]); ok {
		return bound.Of(bound.Term(v).Mul(bound.LogOf(v)))
	}
	return bound.Top()
}

func logNP(args []ssa.Value, i int) bound.Bound {
	if i >= len(args) {
		return bound.Top()
	}
	if v, ok := argSizeVar(args[i]); ok {
		return bound.Of(bound.LogOf(v))
	}
	return bound.Top()
}

// parametric maps "pkgpath.Name" to a curated callback cost. Every count is an
// upper bound from the function's documented contract.
//
// Task 0 finding: the plain cost table has no fixed-cost entry for any
// callback-taking function (only sort.Ints/Float64s/Strings, which sort
// concrete elements), so there was no latent wrong bound to remove here.
//
// The slices.Collect / slices.Sorted / slices.SortedFunc family is deferred:
// its Base depends on the produced result length (a result-size model bigo
// does not have yet), so those stay ⊤ via the normal path.
var parametric = map[string]ParamEntry{
	// sort.Slice(x any, less func(i, j int) bool): ~n log n comparisons.
	"sort.Slice": {
		Base:   func(a []ssa.Value) bound.Bound { return nLogNP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{1: func(a []ssa.Value) bound.Bound { return nLogNP(a, 0) }},
	},
	"sort.SliceStable": {
		Base:   func(a []ssa.Value) bound.Bound { return nLogNP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{1: func(a []ssa.Value) bound.Bound { return nLogNP(a, 0) }},
	},
	// sort.Search(n int, f func(int) bool): O(log n) probes.
	"sort.Search": {
		Base:   func(a []ssa.Value) bound.Bound { return logNP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{1: func(a []ssa.Value) bound.Bound { return logNP(a, 0) }},
	},
	// slices.SortFunc(x, cmp): ~n log n comparisons.
	"slices.SortFunc": {
		Base:   func(a []ssa.Value) bound.Bound { return nLogNP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{1: func(a []ssa.Value) bound.Bound { return nLogNP(a, 0) }},
	},
	"slices.SortStableFunc": {
		Base:   func(a []ssa.Value) bound.Bound { return nLogNP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{1: func(a []ssa.Value) bound.Bound { return nLogNP(a, 0) }},
	},
	// slices.BinarySearchFunc(x, target, cmp): O(log n) probes; cmp is arg 2.
	"slices.BinarySearchFunc": {
		Base:   func(a []ssa.Value) bound.Bound { return logNP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{2: func(a []ssa.Value) bound.Bound { return logNP(a, 0) }},
	},
	// slices.ContainsFunc(s, f), slices.IndexFunc(s, f): one linear pass.
	"slices.ContainsFunc": {
		Base:   func(a []ssa.Value) bound.Bound { return linearP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{1: func(a []ssa.Value) bound.Bound { return linearP(a, 0) }},
	},
	"slices.IndexFunc": {
		Base:   func(a []ssa.Value) bound.Bound { return linearP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{1: func(a []ssa.Value) bound.Bound { return linearP(a, 0) }},
	},
	// slices.MaxFunc(s, cmp), slices.MinFunc(s, cmp), slices.CompactFunc(s, eq).
	"slices.MaxFunc": {
		Base:   func(a []ssa.Value) bound.Bound { return linearP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{1: func(a []ssa.Value) bound.Bound { return linearP(a, 0) }},
	},
	"slices.MinFunc": {
		Base:   func(a []ssa.Value) bound.Bound { return linearP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{1: func(a []ssa.Value) bound.Bound { return linearP(a, 0) }},
	},
	"slices.CompactFunc": {
		Base:   func(a []ssa.Value) bound.Bound { return linearP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{1: func(a []ssa.Value) bound.Bound { return linearP(a, 0) }},
	},
	// slices.EqualFunc(s1, s2, eq): O(min(len)) ≤ O(len s1); eq is arg 2.
	"slices.EqualFunc": {
		Base:   func(a []ssa.Value) bound.Bound { return linearP(a, 0) },
		PerArg: map[int]func([]ssa.Value) bound.Bound{2: func(a []ssa.Value) bound.Bound { return linearP(a, 0) }},
	},
}
