// Package costtable maps builtins and curated stdlib calls to asymptotic costs.
package costtable

import (
	"go/types"
	"sync"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/fieldpath"
	"github.com/RomanAgaltsev/bigo/internal/size"
	"github.com/RomanAgaltsev/bigo/internal/sizefacts"
	"golang.org/x/tools/go/ssa"
)

// stabMemo caches fieldpath.Stability per function. Entries are immutable and
// live for the process — acceptable for a batch CLI/analyzer run; a daemon-mode
// consumer must revisit (spec §5).
var stabMemo sync.Map // *ssa.Function -> *fieldpath.Stability

func stabilityOf(fn *ssa.Function) *fieldpath.Stability {
	if s, ok := stabMemo.Load(fn); ok {
		return s.(*fieldpath.Stability)
	}
	s, _ := stabMemo.LoadOrStore(fn, fieldpath.Analyze(fn))
	return s.(*fieldpath.Stability)
}

// argExtent resolves an argument's size: parameters first (size.Value,
// unchanged behavior), then locally-derived values through sizefacts.ArgSize
// in the argument's enclosing function. Constants, globals, and builtins have
// no Parent and stay unresolved.
func argExtent(v ssa.Value) (bound.Var, bool) {
	if av, ok := size.Value(v); ok {
		return av, true
	}
	fn := v.Parent()
	if fn == nil {
		return "", false
	}
	f := &sizefacts.Facts{Stab: stabilityOf(fn)}
	return f.ArgSize(v)
}

// Lookup returns the cost of a builtin or curated stdlib call.
// ok=false means the callee is not in the table.
func Lookup(c *ssa.CallCommon) (bound.Bound, bool) {
	if b, ok := c.Value.(*ssa.Builtin); ok {
		return builtinCost(b.Name(), c.Args)
	}
	key, ok := calleeKey(c)
	if !ok {
		return bound.Bound{}, false
	}
	fn, ok := stdlib[key]
	if !ok {
		return bound.Bound{}, false
	}
	return fn(c.Args), true
}

// calleeKey resolves the cost-table key of a non-builtin call: the package-
// qualified callee name, or the receiver-qualified name for methods (so
// Mutex.Lock and RWMutex.Lock cannot collide on a bare "sync.Lock"). An
// instantiation of a generic function has a nil Pkg and a name like
// "Contains[[]int, int]"; its origin carries the package and plain name.
func calleeKey(c *ssa.CallCommon) (string, bool) {
	callee := c.StaticCallee()
	if callee == nil {
		return "", false
	}
	if orig := callee.Origin(); orig != nil {
		callee = orig
	}
	if callee.Pkg == nil || callee.Pkg.Pkg == nil {
		return "", false
	}
	key := callee.Pkg.Pkg.Path() + "." + callee.Name()
	if callee.Signature.Recv() != nil {
		obj, ok := callee.Object().(*types.Func)
		if !ok {
			return "", false
		}
		key = obj.FullName()
	}
	return key, true
}

// Priced reports whether the callee has an entry that can cost this call. The
// engine uses it to distinguish "the callee has no cost" from "the callee is
// priced but the ARGUMENT SIZE is unresolved" — misreported as the former
// through v1.28.1 (the cause text lied on MergeSort's copy and chaotic's
// specSignature).
//
// It answers from the same tables the cost path uses, asking each builtin
// entry itself rather than testing name membership, because some entries
// decline on operand type (min/max on strings, clear on maps) and a name-only
// answer would call those priced. The cost of that is one argument-size
// resolution on a path that runs only when the bound is already ⊤ — the right
// trade for making drift between "priced" and "costed" structurally impossible
// rather than a comment someone must remember to honour.
func Priced(c *ssa.CallCommon) bool {
	if b, ok := c.Value.(*ssa.Builtin); ok {
		_, priced := builtinCost(b.Name(), c.Args)
		return priced
	}
	key, ok := calleeKey(c)
	if !ok {
		return false
	}
	_, ok = stdlib[key]
	return ok
}

// builtins is the single source of truth for builtin pricing: both the cost
// path (builtinCost) and the diagnostic path (Priced) read it, so membership
// and pricing cannot drift apart. A hand-copied second list is exactly how the
// cause text would start lying again (review 2026-07-18, F3).
//
// An entry may still decline (ok=false) for operand types it cannot price
// soundly — see orderedBuiltin and clearBuiltin.
var builtins = map[string]func(args []ssa.Value) (bound.Bound, bool){
	// append/delete are amortized O(1); the rest are genuinely O(1).
	"len":     constBuiltin,
	"cap":     constBuiltin,
	"append":  constBuiltin,
	"delete":  constBuiltin,
	"close":   constBuiltin,
	"panic":   constBuiltin,
	"recover": constBuiltin,
	"print":   constBuiltin,
	"println": constBuiltin,
	"new":     constBuiltin,
	"complex": constBuiltin,
	"real":    constBuiltin,
	"imag":    constBuiltin,
	"copy":    func(a []ssa.Value) (bound.Bound, bool) { return linear(a, 0), true },
	"min":     orderedBuiltin,
	"max":     orderedBuiltin,
	"clear":   clearBuiltin,
}

func constBuiltin([]ssa.Value) (bound.Bound, bool) { return bound.Constant(), true }

// orderedBuiltin prices min/max. For numeric operands each comparison is O(1)
// and the argument count is fixed at the call site, so the call is O(1). For
// STRING operands a comparison is O(min(len)) — not constant — and a chain of
// them is not bounded by any single argument's length, so strings stay
// unpriced rather than under-approximated.
func orderedBuiltin(args []ssa.Value) (bound.Bound, bool) {
	for _, a := range args {
		if b, ok := a.Type().Underlying().(*types.Basic); ok && b.Info()&types.IsString != 0 {
			return bound.Bound{}, false
		}
	}
	return bound.Constant(), true
}

// clearBuiltin prices clear(x). For a SLICE it zeroes exactly len(x) elements:
// O(len(x)). For a MAP it is NOT O(len(m)) — the runtime walks the bucket
// array, whose size tracks the map's historical high-water capacity, so a map
// that once held a million entries and now holds one still costs a million.
// bigo has no cap(map) size variable to express that, so map clears stay
// unpriced. (Note this does NOT contradict R5's O(len(m)) for `range`: that
// bounds the number of ITERATIONS, each of which yields an element, not the
// runtime's bucket walk.)
func clearBuiltin(args []ssa.Value) (bound.Bound, bool) {
	if len(args) != 1 {
		return bound.Bound{}, false
	}
	if _, isSlice := args[0].Type().Underlying().(*types.Slice); !isSlice {
		return bound.Bound{}, false
	}
	return linear(args, 0), true
}

func builtinCost(name string, args []ssa.Value) (bound.Bound, bool) {
	fn, ok := builtins[name]
	if !ok {
		return bound.Bound{}, false
	}
	return fn(args)
}

// linear returns O(size of args[i]), or Top() if that size is unknown.
func linear(args []ssa.Value, i int) bound.Bound {
	if i >= len(args) {
		return bound.Top()
	}
	if v, ok := argExtent(args[i]); ok {
		return bound.Of(bound.Term(v))
	}
	return bound.Top()
}

// nLogN returns O(n log n) where n = size of args[i], or Top().
func nLogN(args []ssa.Value, i int) bound.Bound {
	if i >= len(args) {
		return bound.Top()
	}
	if v, ok := argExtent(args[i]); ok {
		return bound.Of(bound.Term(v).Mul(bound.LogOf(v)))
	}
	return bound.Top()
}

// logN returns O(log n) where n = size of args[i], or Top().
func logN(args []ssa.Value, i int) bound.Bound {
	if i >= len(args) {
		return bound.Top()
	}
	if v, ok := argExtent(args[i]); ok {
		return bound.Of(bound.LogOf(v))
	}
	return bound.Top()
}

// prodOf returns O(vᵢ · vⱼ), or Top() when either size is unknown.
func prodOf(args []ssa.Value, i, j int) bound.Bound {
	if i >= len(args) || j >= len(args) {
		return bound.Top()
	}
	vi, ok := argExtent(args[i])
	if !ok {
		return bound.Top()
	}
	vj, ok := argExtent(args[j])
	if !ok {
		return bound.Top()
	}
	return bound.Of(bound.Term(vi).Mul(bound.Term(vj)))
}

// constCost ignores its arguments: O(1).
func constCost([]ssa.Value) bound.Bound { return bound.Constant() }

// stdlib maps "pkgpath.Name" to a cost function of the call arguments.
var stdlib = map[string]func(args []ssa.Value) bound.Bound{
	"sort.Ints":     func(a []ssa.Value) bound.Bound { return nLogN(a, 0) },
	"sort.Float64s": func(a []ssa.Value) bound.Bound { return nLogN(a, 0) },
	"sort.Strings":  func(a []ssa.Value) bound.Bound { return nLogN(a, 0) },
	// slices: size-resolvable, no callback.
	"slices.Sort":         func(a []ssa.Value) bound.Bound { return nLogN(a, 0) },
	"slices.Contains":     func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"slices.Index":        func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"slices.Max":          func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"slices.Min":          func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"slices.Reverse":      func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"slices.Equal":        func(a []ssa.Value) bound.Bound { return linear(a, 0) }, // O(min) <= O(len(a))
	"slices.BinarySearch": func(a []ssa.Value) bound.Bound { return logN(a, 0) },
	// strings: linear passes over s. Replace/Join under-approximate output
	// blow-up (documented in README's limitations) — false negatives only.
	"strings.Contains":   func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.Index":      func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.Count":      func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.Replace":    func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.ReplaceAll": func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.Split":      func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.Join":       func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.Fields":     func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.ToLower":    func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.ToUpper":    func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.TrimSpace":  func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.HasPrefix":  func(a []ssa.Value) bound.Bound { return linear(a, 0) }, // O(len(prefix)) <= O(len(s))
	"strings.EqualFold":  func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.Repeat":     func(a []ssa.Value) bound.Bound { return prodOf(a, 0, 1) },
	// bytes mirrors strings.
	"bytes.Contains": func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"bytes.Index":    func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"bytes.Count":    func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	// Iterator producers return a lazy iter.Seq: construction is O(1); the
	// iteration cost is paid at the range site (see LookupIteratorProducer).
	"maps.Keys":       constCost,
	"maps.Values":     constCost,
	"maps.All":        constCost,
	"slices.Values":   constCost,
	"slices.All":      constCost,
	"slices.Backward": constCost,
	// sync: each operation is O(1) work.
	//
	// Soundness: blocking under contention is wall-clock, not work. bigo models
	// total work and never wall-clock (a `go f()` contributes cost(f), a channel
	// receive does not contribute its wait), so a contended Lock is O(1) work in
	// this model exactly as an uncontended one is. Costing these O(1) does not
	// under-approximate any *work* the program performs.
	//
	// Deliberately absent: sync.Once.Do(f) and sync.Map.Range(f) take a function
	// argument and cost cost(f), not O(1) — an O(1) entry would under-approximate
	// a call into a false Within, i.e. a wrong bound. They stay ⊤ until the
	// parametric path (paramsummary) models them.
	"(*sync.Mutex).Lock":       constCost,
	"(*sync.Mutex).Unlock":     constCost,
	"(*sync.Mutex).TryLock":    constCost,
	"(*sync.RWMutex).Lock":     constCost,
	"(*sync.RWMutex).Unlock":   constCost,
	"(*sync.RWMutex).RLock":    constCost,
	"(*sync.RWMutex).RUnlock":  constCost,
	"(*sync.RWMutex).TryLock":  constCost,
	"(*sync.RWMutex).TryRLock": constCost,
	"(*sync.WaitGroup).Add":    constCost,
	"(*sync.WaitGroup).Done":   constCost,
	"(*sync.WaitGroup).Wait":   constCost,
}
