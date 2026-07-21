// Package costtable maps builtins and curated stdlib calls to asymptotic costs.
package costtable

import (
	"go/constant"
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

// product returns O(size(args[i]) * size(args[j])), or Top() if either size is
// unknown.
//
// For entries whose cost is bounded by NEITHER argument alone. The Trim family
// is the case that earned it: those functions test membership of every element
// of one sequence in another, so the two lengths multiply, and pricing them by
// the first argument alone was the seventh wrong bound this project shipped.
func product(args []ssa.Value, i, j int) bound.Bound {
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

// trimCost prices the strings.Trim family, whose cost is bounded by NEITHER
// argument alone: they test membership of every rune of args[0] in the CUTSET
// args[1], so the two lengths multiply.
//
// A CONSTANT cutset is the common case and keeps the linear bound, because a
// compile-time string has a fixed length and contributes only a constant
// factor — `strings.Trim(s, " \t\n")` really is O(len(s)). Refusing that would
// be a precondition stricter than soundness requires, which this project has
// already paid for once (C5's loop-invariance requirement, v1.33.0).
func trimCost(args []ssa.Value) bound.Bound {
	if len(args) < 2 {
		return bound.Top()
	}
	if c, ok := args[1].(*ssa.Const); ok && c.Value != nil && c.Value.Kind() == constant.String {
		return linear(args, 0)
	}
	return product(args, 0, 1)
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

	// --- Survey-ranked entries (v1.35.0) ---
	//
	// Added because the v1.34.0 real-world survey measured them blocking real
	// Go across 12 repositories. Each carries its bound below. What is NOT here
	// matters as much as what is: the fmt family (8,367 sites), encoding/json,
	// and errors.Is are deliberately absent because they have no sound bound —
	// see the block after this map, and the no-fire pins in
	// stdlib_survey_test.go.

	// errors.New allocates a struct holding the string; it does not copy it.
	"errors.New": constCost,

	// One clock read; Since is Now() minus its argument.
	"time.Now":   constCost,
	"time.Since": constCost,
	"time.Until": constCost,

	// Both return a package-level singleton.
	"context.Background": constCost,
	"context.TODO":       constCost,

	// Integer formatting is CONSTANT-bounded, not linear in the value: at most
	// 20 digits for an int64 in base 10, and at most 64 in base 2.
	"strconv.Itoa":       constCost,
	"strconv.FormatInt":  constCost,
	"strconv.FormatUint": constCost,
	"strconv.FormatBool": constCost,

	// Parsing scans the string.
	"strconv.Atoi":      func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strconv.ParseInt":  func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strconv.ParseUint": func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strconv.ParseBool": func(a []ssa.Value) bound.Bound { return linear(a, 0) },

	// Bit reinterpretation.
	"math.Float64bits":     constCost,
	"math.Float64frombits": constCost,
	"math.Float32bits":     constCost,
	"math.Float32frombits": constCost,

	// One atomic instruction each. Same argument as the sync block above:
	// contention is wall-clock, not work.
	"sync/atomic.LoadInt32":             constCost,
	"sync/atomic.LoadInt64":             constCost,
	"sync/atomic.LoadUint32":            constCost,
	"sync/atomic.LoadUint64":            constCost,
	"sync/atomic.LoadPointer":           constCost,
	"sync/atomic.StoreInt32":            constCost,
	"sync/atomic.StoreInt64":            constCost,
	"sync/atomic.StoreUint32":           constCost,
	"sync/atomic.StoreUint64":           constCost,
	"sync/atomic.AddInt32":              constCost,
	"sync/atomic.AddInt64":              constCost,
	"sync/atomic.AddUint32":             constCost,
	"sync/atomic.AddUint64":             constCost,
	"sync/atomic.SwapInt32":             constCost,
	"sync/atomic.SwapInt64":             constCost,
	"sync/atomic.SwapUint32":            constCost,
	"sync/atomic.SwapUint64":            constCost,
	"sync/atomic.CompareAndSwapInt32":   constCost,
	"sync/atomic.CompareAndSwapInt64":   constCost,
	"sync/atomic.CompareAndSwapUint32":  constCost,
	"sync/atomic.CompareAndSwapUint64":  constCost,
	"sync/atomic.CompareAndSwapPointer": constCost,

	// Does not return.
	"os.Exit": constCost,

	// reflect: constant work on the interface header.
	//
	// ROADMAP §2 lists reflect as permanent annotate-or-trust. That is about
	// inferring THROUGH reflection — bigo cannot know what a reflected call
	// does — and it does NOT mean these operations perform unbounded work.
	// Pricing them claims nothing about the program's use of reflection.
	//
	// Deliberately absent: (reflect.Value).Call invokes arbitrary code, and
	// .Interface copies a value of size unknown at this level. Both are pinned
	// ⊤ in stdlib_survey_test.go.
	"reflect.TypeOf":           constCost,
	"reflect.ValueOf":          constCost,
	"(reflect.Value).Kind":     constCost,
	"(reflect.Value).Type":     constCost,
	"(reflect.Value).IsValid":  constCost,
	"(reflect.Value).IsNil":    constCost,
	"(reflect.Value).Len":      constCost,
	"(reflect.Type).Kind":      constCost,
	"(reflect.Type).Name":      constCost,
	"(reflect.Type).String":    constCost,
	"(reflect.Value).NumField": constCost,
	"(reflect.Type).NumField":  constCost,
	"(reflect.Value).CanSet":   constCost,
	"(reflect.Value).CanAddr":  constCost,
	"(reflect.Value).IsZero":   constCost,

	// O(len(arg 0)) — the strings.HasPrefix / slices.Equal precedents, which
	// apply because each of these COMPARES two sequences and stops at the
	// shorter, so the cost is O(min(len)) <= O(len(arg 0)).
	//
	// The precedent is about the IMPLEMENTATION, not the signature: it does not
	// extend to a second argument that is scanned repeatedly. See the Trim
	// family below, which took it by analogy and shipped a wrong bound.
	"strings.HasSuffix":  func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.TrimPrefix": func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.TrimSuffix": func(a []ssa.Value) bound.Bound { return linear(a, 0) },

	// O(len(s) * len(cutset)) — NOT O(len(s)).
	//
	// These take a CUTSET, not a prefix, and test membership of every rune of s
	// in it. Both of Go's paths carry a cutset term:
	//
	//   - makeASCIISet walks the entire cutset before any trimming begins, so
	//     even the ASCII fast path is O(len(s) + len(cutset));
	//   - a cutset holding any byte >= utf8.RuneSelf makes makeASCIISet fail,
	//     and trimLeftUnicode then calls ContainsRune — a scan of cutset — once
	//     per rune of s: O(len(s) * len(cutset)).
	//
	// The product dominates both. Priced O(len(s)) from v1.35.0 to v1.38.0 by
	// inheriting the HasPrefix precedent above; that was the SEVENTH wrong
	// bound, found by the 2026-07-21 review reading the stdlib source rather
	// than by any of the three instruments. Pinned in stdlib_survey_test.go.
	"strings.Trim":      trimCost,
	"strings.TrimLeft":  trimCost,
	"strings.TrimRight": trimCost,

	// Safe on a second string argument because Index returns early when the
	// needle is longer than the haystack, so all work is bounded by len(arg 0).
	"strings.LastIndex": func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.IndexByte": func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.SplitN":    func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.Title":     func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"bytes.Equal":       func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"bytes.HasPrefix":   func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"bytes.HasSuffix":   func(a []ssa.Value) bound.Bound { return linear(a, 0) },
}
