// Package costtable maps builtins and curated stdlib calls to asymptotic costs.
package costtable

import (
	"go/types"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/size"
	"golang.org/x/tools/go/ssa"
)

// Lookup returns the cost of a builtin or curated stdlib call.
// ok=false means the callee is not in the table.
func Lookup(c *ssa.CallCommon) (bound.Bound, bool) {
	if b, ok := c.Value.(*ssa.Builtin); ok {
		return builtinCost(b.Name(), c.Args)
	}
	callee := c.StaticCallee()
	if callee == nil {
		return bound.Bound{}, false
	}
	// An instantiation of a generic function has a nil Pkg and a name like
	// "Contains[[]int, int]"; its origin carries the package and plain name.
	if orig := callee.Origin(); orig != nil {
		callee = orig
	}
	if callee.Pkg == nil || callee.Pkg.Pkg == nil {
		return bound.Bound{}, false
	}
	key := callee.Pkg.Pkg.Path() + "." + callee.Name()
	// A method keys on its receiver-qualified name ("(*sync.Mutex).Lock"), so
	// that same-named methods on different types in one package — Mutex.Lock and
	// RWMutex.Lock — cannot collide on a bare "sync.Lock".
	if callee.Signature.Recv() != nil {
		obj, ok := callee.Object().(*types.Func)
		if !ok {
			return bound.Bound{}, false
		}
		key = obj.FullName()
	}
	fn, ok := stdlib[key]
	if !ok {
		return bound.Bound{}, false
	}
	return fn(c.Args), true
}

func builtinCost(name string, args []ssa.Value) (bound.Bound, bool) {
	switch name {
	case "len", "cap", "append", "delete", "close", "panic", "recover", "print", "println", "new":
		// append/delete are amortized O(1); the rest are O(1).
		return bound.Constant(), true
	case "copy":
		return linear(args, 0), true
	default:
		return bound.Bound{}, false
	}
}

// linear returns O(size of args[i]), or Top() if that size is unknown.
func linear(args []ssa.Value, i int) bound.Bound {
	if i >= len(args) {
		return bound.Top()
	}
	if v, ok := size.Value(args[i]); ok {
		return bound.Of(bound.Term(v))
	}
	return bound.Top()
}

// nLogN returns O(n log n) where n = size of args[i], or Top().
func nLogN(args []ssa.Value, i int) bound.Bound {
	if i >= len(args) {
		return bound.Top()
	}
	if v, ok := size.Value(args[i]); ok {
		return bound.Of(bound.Term(v).Mul(bound.LogOf(v)))
	}
	return bound.Top()
}

// logN returns O(log n) where n = size of args[i], or Top().
func logN(args []ssa.Value, i int) bound.Bound {
	if i >= len(args) {
		return bound.Top()
	}
	if v, ok := size.Value(args[i]); ok {
		return bound.Of(bound.LogOf(v))
	}
	return bound.Top()
}

// prodOf returns O(vᵢ · vⱼ), or Top() when either size is unknown.
func prodOf(args []ssa.Value, i, j int) bound.Bound {
	if i >= len(args) || j >= len(args) {
		return bound.Top()
	}
	vi, ok := size.Value(args[i])
	if !ok {
		return bound.Top()
	}
	vj, ok := size.Value(args[j])
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
