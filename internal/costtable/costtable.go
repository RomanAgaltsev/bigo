// Package costtable maps builtins and curated stdlib calls to asymptotic costs.
//
// # The unit-element axiom
//
// A size variable in bigo's grammar is a COUNT: len(x) for a slice, string or
// map. The total CONTENT of a collection — the sum of its elements' lengths —
// has no name in that grammar and cannot be written in a bound.
//
// So the model charges UNIT COST for operating on one element. slices.Contains
// over a []string is O(len(xs)), not O(len(xs) * the longest string in it), and
// sort.Strings is O(n log n). This is an AXIOM, stated here because it was
// unwritten until the 2026-08-12 review asked what it was, and an unwritten
// axiom is indistinguishable from an oversight.
//
// The alternative was refusing every element operation whose cost depends on
// element content, which would make sorting a []string ⊤ and contradict the
// canonical corpus, where sorting is O(n log n) whatever it sorts. That is a
// large measured capability loss bought with an unnameable quantity.
//
// THE AXIOM HAS A BOUNDARY AND THE BOUNDARY IS THE WHOLE POINT. It covers the
// cost of touching an ELEMENT of the collection being sized. It does NOT cover
// scanning a SEPARATE argument that has its own name:
//
//   - strings.Trim's cutset, strings.Join's separator and strings.Replace's
//     replacement are not elements of anything. They are independent
//     parameters, they are nameable, and their lengths multiply. Pricing them
//     as if the axiom covered them produced the seventh and eighth wrong
//     bounds — see productUnlessConst.
//   - the substring-search family's needle is the same shape; see searchCost.
//
// A user reading O(len(xs)) is therefore reading "work proportional to the
// number of elements", which is what this package means and now says.
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
	return FuncKey(callee)
}

// CalleeKey is calleeKey for consumers outside the table (the assumption
// mechanism keys its entries in exactly this vocabulary).
func CalleeKey(c *ssa.CallCommon) (string, bool) { return calleeKey(c) }

// FuncKey is the key of a function itself rather than of a call to it.
func FuncKey(fn *ssa.Function) (string, bool) {
	if orig := fn.Origin(); orig != nil {
		fn = orig
	}
	if fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return "", false
	}
	key := fn.Pkg.Pkg.Path() + "." + fn.Name()
	if fn.Signature.Recv() != nil {
		obj, ok := fn.Object().(*types.Func)
		if !ok {
			return "", false
		}
		key = obj.FullName()
	}
	return key, true
}

// Curated reports whether the curated tables already answer this cost-table
// key — both the plain table and the parametric one.
//
// It exists for consumers deciding whether a key is worth ASSERTING. Precedence
// puts both curated tables above an external assumption, so a trust entry for a
// curated key is shadowed: it warns, changes no verdict, and wastes the
// reasoning someone did to write it. `bigo trust init` filters on this.
//
// Asking the table is deliberate, rather than parsing the cause text. A cause
// reading "unresolved ARGUMENT SIZE at call to X" means X is priced and only the
// argument's size is unknown, which is exactly the unhelpful case — but reading
// that from prose is what the callee key was added to avoid, and a table-driven
// answer stays correct as entries are added.
func Curated(key string) bool {
	if _, ok := stdlib[key]; ok {
		return true
	}
	_, ok := parametric[key]
	return ok
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

// productUnlessConst prices an entry whose cost is O(size(args[i]) *
// size(args[j])) in general but collapses to O(size(args[i])) when args[j] is a
// compile-time string constant — a fixed length contributes only a constant
// factor.
//
// This is the shared repair for a class this project has now shipped TWICE as a
// wrong bound. The members differ in mechanism and agree in arithmetic:
//
//   - the Trim family READS args[j] once per element of args[i] (membership of
//     every rune of s in the cutset) — the seventh wrong bound, v1.38.1;
//   - Join and Replace WRITE args[j] once per element of args[i] (the
//     separator, or the replacement) — the eighth, 2026-08-12 review F1.
//
// The constant arm is not an optimisation, it is a POSITIVE CONTROL the entries
// owe: `strings.Trim(s, " \t\n")` and `strings.Join(parts, ", ")` really are
// linear, and refusing them would be a precondition stricter than soundness
// requires — the C5 capability loss, which this project has already paid for.
//
// Generalised from trimCost 2026-08-12, when the class acquired its third and
// fourth entries. The sweep that produced them — every one-argument entry in
// this table, checked against its signature — is recorded in SWEEP.md beside
// this file; a fifth member should extend this helper, never hand-roll the
// check again.
func productUnlessConst(args []ssa.Value, i, j int) bound.Bound {
	if j < len(args) {
		if c, ok := args[j].(*ssa.Const); ok && c.Value != nil && c.Value.Kind() == constant.String {
			return linear(args, i)
		}
	}
	return product(args, i, j)
}

// trimCost prices the strings.Trim family, whose cost is bounded by NEITHER
// argument alone: they test membership of every rune of args[0] in the CUTSET
// args[1], so the two lengths multiply.
func trimCost(args []ssa.Value) bound.Bound { return productUnlessConst(args, 0, 1) }

// joinCost prices strings.Join, which writes the SEPARATOR args[1] exactly
// len(args[0])-1 times, so the cost carries a len(sep) factor whatever the
// elements hold.
func joinCost(args []ssa.Value) bound.Bound { return productUnlessConst(args, 0, 1) }

// searchCost prices the substring-search family: Index, Contains, Count,
// LastIndex, Cut, Split and SplitN, in both strings and bytes.
//
// The needle is a second length that can multiply, and the reason is one
// branch deep. For a needle at or below bytealg.MaxLen — 31 or 63 bytes
// depending on the CPU — Go brute-forces with a comparison bounded by that
// machine constant, and the linear bound is sound under the same licence
// GOMAXPROCS gets in Pool.Put and the word width gets in bits.Div64. For a
// LONGER needle the search falls through to IndexRabinKarp, which verifies
// every hash match with a full compare:
//
//	if h == hashss && string(s[i-n:i]) == string(sep) { ... }
//
// PrimeRK is a compile-time constant, so collisions are constructible, and the
// worst case is len(s) times len(sep).
//
// DECIDED 2026-08-12, sweep item C, rather than left as a caveat. The 2026-08-12
// review declined to call this a break because the worst case is adversarial
// and remote. That is true and it is not the test this project applies: the
// prime directive is a WORST-CASE upper bound, ⊤ is always safe, and a bound
// that holds for typical inputs is the reasoning that produced Trim. Pricing
// the product also makes the table internally consistent — Trim, Join, Replace
// and the search family are now one rule with one helper.
//
// The cost of deciding this way is small and was checked: a LITERAL needle
// takes the constant arm, which is the overwhelming majority of real calls, and
// a variable needle still gets a BOUND rather than ⊤ wherever both extents
// resolve. IndexByte is not here — a single byte has no length to multiply.
func searchCost(args []ssa.Value) bound.Bound { return productUnlessConst(args, 0, 1) }

// replaceCost prices strings.Replace and ReplaceAll, which write the
// REPLACEMENT args[2] once per replacement, and the replacement count is
// Count(s, old) — bounded only by len(s).
//
// Note the index: the replacement is argument 2, not 1. Sizing this by `old`
// would be a different wrong bound that happens to look like a fix.
func replaceCost(args []ssa.Value) bound.Bound { return productUnlessConst(args, 0, 2) }

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
	"strings.Contains":   searchCost,
	"strings.Index":      searchCost,
	"strings.Count":      searchCost,
	"strings.Replace":    replaceCost,
	"strings.ReplaceAll": replaceCost,
	"strings.Split":      searchCost,
	"strings.Join":       joinCost,
	"strings.Fields":     func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.ToLower":    func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.ToUpper":    func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.TrimSpace":  func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.HasPrefix":  func(a []ssa.Value) bound.Bound { return linear(a, 0) }, // O(len(prefix)) <= O(len(s))
	"strings.EqualFold":  func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.Repeat":     func(a []ssa.Value) bound.Bound { return prodOf(a, 0, 1) },
	// bytes mirrors strings.
	"bytes.Contains": searchCost,
	"bytes.Index":    searchCost,
	"bytes.Count":    searchCost,
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

	// (*sync.Pool).Put is O(1) AMORTIZED, priced from sync/pool.go and
	// poolqueue.go rather than from the block above. Two allocation paths sit
	// behind it and both grow geometrically: poolChain.pushHead allocates a
	// DOUBLED ring when the current one fills, the same shape as append, whose
	// amortization bigo already licenses as a documented primitive; and pinSlow
	// allocates make([]poolLocal, GOMAXPROCS) at most once per pool per GC
	// cycle, where GOMAXPROCS is a machine constant and not an input size.
	//
	// Its sibling (*sync.Pool).Get is deliberately ABSENT and must stay so: an
	// empty pool calls p.New, a func-typed STRUCT FIELD, i.e. arbitrary user
	// code (the funcvalue class, measured at zero reachable on real code
	// 2026-07-20). Get also loops over every P in getSlow. Either alone makes
	// an O(1) entry a wrong bound.
	"(*sync.Pool).Put": constCost,

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

	// The rest of the package-level API, added 2026-08-11 (review F4). The 22
	// entries above were chosen from a callee-name histogram, so whatever the
	// survey population did not happen to call was never considered — leaving
	// the package-level API 22 of 39 while the typed API it backs is complete,
	// i.e. (*atomic.Pointer[T]).Store was O(1) and StorePointer, the function
	// it is a one-line delegation to, was ⊤.
	//
	// All 39 are declared with NO Go body in doc.go / doc_32.go / doc_64.go and
	// implemented in assembly — read from those files, not inherited from the
	// neighbours above (the Trim rule). Where a machine lowers And/Or to a CAS
	// retry, that is contention, and the sync block above already records that
	// contention is wall-clock rather than work.
	//
	// Yield is UNMEASURED: a consistency fix, not a coverage claim.
	"sync/atomic.AddUintptr":            constCost,
	"sync/atomic.AndInt32":              constCost,
	"sync/atomic.AndInt64":              constCost,
	"sync/atomic.AndUint32":             constCost,
	"sync/atomic.AndUint64":             constCost,
	"sync/atomic.AndUintptr":            constCost,
	"sync/atomic.CompareAndSwapUintptr": constCost,
	"sync/atomic.LoadUintptr":           constCost,
	"sync/atomic.OrInt32":               constCost,
	"sync/atomic.OrInt64":               constCost,
	"sync/atomic.OrUint32":              constCost,
	"sync/atomic.OrUint64":              constCost,
	"sync/atomic.OrUintptr":             constCost,
	"sync/atomic.StorePointer":          constCost,
	"sync/atomic.StoreUintptr":          constCost,
	"sync/atomic.SwapPointer":           constCost,
	"sync/atomic.SwapUintptr":           constCost,

	// The TYPED atomic API, added 2026-07-24 after the S1 truthfulness probe
	// measured 163 hand-written graduations across the survey targets.
	//
	// Priced from sync/atomic/type.go, NOT by analogy with the package-level
	// entries above (the Trim rule forbids inheriting a precedent): every
	// method there is a one-line delegation to a compiler intrinsic, and
	// TYPE.GO CONTAINS NO LOOP. That absence is what makes one argument cover
	// all of them.
	//
	// Scope narrowed 2026-08-11: this argument is about type.go's seven struct
	// types only. atomic.Value lives in value.go, is NOT covered by it, and is
	// priced below under a different argument — the previous wording said "the
	// package's non-test sources contain no loop anywhere in this API", which
	// is false of value.go and would have to be re-read every time someone
	// checked it.
	"(*sync/atomic.Bool).Load":                 constCost,
	"(*sync/atomic.Bool).Store":                constCost,
	"(*sync/atomic.Bool).Swap":                 constCost,
	"(*sync/atomic.Bool).CompareAndSwap":       constCost,
	"(*sync/atomic.Int32).Load":                constCost,
	"(*sync/atomic.Int32).Store":               constCost,
	"(*sync/atomic.Int32).Swap":                constCost,
	"(*sync/atomic.Int32).CompareAndSwap":      constCost,
	"(*sync/atomic.Int32).Add":                 constCost,
	"(*sync/atomic.Int32).And":                 constCost,
	"(*sync/atomic.Int32).Or":                  constCost,
	"(*sync/atomic.Int64).Load":                constCost,
	"(*sync/atomic.Int64).Store":               constCost,
	"(*sync/atomic.Int64).Swap":                constCost,
	"(*sync/atomic.Int64).CompareAndSwap":      constCost,
	"(*sync/atomic.Int64).Add":                 constCost,
	"(*sync/atomic.Int64).And":                 constCost,
	"(*sync/atomic.Int64).Or":                  constCost,
	"(*sync/atomic.Uint32).Load":               constCost,
	"(*sync/atomic.Uint32).Store":              constCost,
	"(*sync/atomic.Uint32).Swap":               constCost,
	"(*sync/atomic.Uint32).CompareAndSwap":     constCost,
	"(*sync/atomic.Uint32).Add":                constCost,
	"(*sync/atomic.Uint32).And":                constCost,
	"(*sync/atomic.Uint32).Or":                 constCost,
	"(*sync/atomic.Uint64).Load":               constCost,
	"(*sync/atomic.Uint64).Store":              constCost,
	"(*sync/atomic.Uint64).Swap":               constCost,
	"(*sync/atomic.Uint64).CompareAndSwap":     constCost,
	"(*sync/atomic.Uint64).Add":                constCost,
	"(*sync/atomic.Uint64).And":                constCost,
	"(*sync/atomic.Uint64).Or":                 constCost,
	"(*sync/atomic.Uintptr).Load":              constCost,
	"(*sync/atomic.Uintptr).Store":             constCost,
	"(*sync/atomic.Uintptr).Swap":              constCost,
	"(*sync/atomic.Uintptr).CompareAndSwap":    constCost,
	"(*sync/atomic.Uintptr).Add":               constCost,
	"(*sync/atomic.Uintptr).And":               constCost,
	"(*sync/atomic.Uintptr).Or":                constCost,
	"(*sync/atomic.Pointer[T]).Load":           constCost,
	"(*sync/atomic.Pointer[T]).Store":          constCost,
	"(*sync/atomic.Pointer[T]).Swap":           constCost,
	"(*sync/atomic.Pointer[T]).CompareAndSwap": constCost,

	// atomic.Value (value.go), added 2026-08-11 (review F3). v1.40.0 excluded
	// the WHOLE TYPE with one sentence about Store's spin loop. Read against
	// the source that reason is wrong twice over, and the second half of that
	// is the dangerous half:
	//
	//  1. Load has NO LOOP AT ALL — two LoadPointers, a sentinel comparison and
	//     two word writes. It was refused by a criterion it does not meet, and
	//     the refusal was pinned, so the pin preserved the capability loss (the
	//     C5 lesson: a precondition stricter than soundness requires is a
	//     silent capability loss).
	//
	//  2. Store's and Swap's loops are PURE CONTENTION: the only two continues
	//     fire when another goroutine won the first-store CAS, or when a first
	//     store is mid-flight (the stdlib comments call it an active spin
	//     wait). No iteration count depends on any input size. That is the same
	//     doctrine the sync block above already applies to (*sync.Mutex).Lock,
	//     whose lockSlow spins too — so "it has a loop" was never this table's
	//     operative criterion, and leaving it written that way invited someone
	//     to notice the inconsistency and price ALL FOUR.
	//
	// Which would have been a wrong bound, because CompareAndSwap is NOT a
	// contention question and stays ⊤ on its own merits: it compares the stored
	// value with `i != old`, a RUNTIME INTERFACE EQUALITY check, and the stdlib
	// comment says that is deliberate — "This allows value types to be
	// compared, something not offered by the package functions." If the dynamic
	// type contains a string, that comparison costs O(len(string)); if it
	// contains an array, O(len of the array). Neither is nameable at the call
	// site. Same shape as min/max on strings (v1.30.1) and as Trim's cutset
	// (v1.38.1): the cost depends on something other than what the entry is
	// sized by, so it must not be priced.
	"(*sync/atomic.Value).Load":  constCost,
	"(*sync/atomic.Value).Store": constCost,
	"(*sync/atomic.Value).Swap":  constCost,

	// --- First-contact entries, added 2026-08-11 ---
	//
	// `bigo trust init` was run against four real repositories and most of what
	// it offered users to assert turned out to be bigo's OWN unfilled table: 17
	// of the 39 keys examined were soundly priceable here, for everyone, in one
	// line each. These are those.
	//
	// The lane is deliberately weighted to CONSTANTS, and the reason is
	// measured rather than aesthetic: constant entries delivered ~93% of their
	// graduation count (pgx, 27 of 29) while a size-dependent one delivered 6%
	// (goldmark, 2 of 32), because the argument size does not resolve at most
	// real call sites. A linear entry is still correct and still worth having;
	// it just buys less until argument-size resolution improves.
	//
	// Each entry is priced from its own implementation. Where one delegates,
	// that is said explicitly rather than left as an analogy.

	// Allocates one valueCtx and checks the key is comparable via a type
	// descriptor lookup. No traversal of the parent chain — Value() walks, this
	// does not.
	"context.WithValue": constCost,

	// Two word comparisons on the receiver (t.wall == 0 && t.ext == 0).
	"(time.Time).IsZero": constCost,

	// Allocates a Logger and assigns three fields behind a mutex.
	"log.New": constCost,

	// Straight-line bit arithmetic on a fixed-width address; no loop, and the
	// v6 path masks a 128-bit value rather than iterating it.
	"(net/netip.Prefix).Contains": constCost,

	// The encoding/binary fixed-width family. ONE argument covers all of them:
	// the whole littleEndian/bigEndian block contains NO LOOP, and every method
	// reads or writes a constant number of bytes at fixed offsets. Append*
	// additionally rides append's documented-primitive amortization, appending
	// a constant number of bytes per call.
	"(encoding/binary.littleEndian).Uint16":       constCost,
	"(encoding/binary.littleEndian).Uint32":       constCost,
	"(encoding/binary.littleEndian).Uint64":       constCost,
	"(encoding/binary.littleEndian).PutUint16":    constCost,
	"(encoding/binary.littleEndian).PutUint32":    constCost,
	"(encoding/binary.littleEndian).PutUint64":    constCost,
	"(encoding/binary.littleEndian).AppendUint16": constCost,
	"(encoding/binary.littleEndian).AppendUint32": constCost,
	"(encoding/binary.littleEndian).AppendUint64": constCost,
	"(encoding/binary.bigEndian).Uint16":          constCost,
	"(encoding/binary.bigEndian).Uint32":          constCost,
	"(encoding/binary.bigEndian).Uint64":          constCost,
	"(encoding/binary.bigEndian).PutUint16":       constCost,
	"(encoding/binary.bigEndian).PutUint32":       constCost,
	"(encoding/binary.bigEndian).PutUint64":       constCost,
	"(encoding/binary.bigEndian).AppendUint16":    constCost,
	"(encoding/binary.bigEndian).AppendUint32":    constCost,
	"(encoding/binary.bigEndian).AppendUint64":    constCost,

	// strings.Cut is Index(s, sep) plus two slice expressions. Its cost IS
	// strings.Index's, read from Cut's body rather than inherited from a
	// neighbour — which also means this entry is exactly as sound as
	// strings.Index's above, no more and no less. That sentence was written as
	// a reassurance and read, in the 2026-08-12 review, as a warning: the
	// neighbour it leaned on had never been swept. Both are searchCost now.
	"strings.Cut": searchCost,

	// bytealg.IndexByte scans b once for a single byte.
	"bytes.IndexByte": func(a []ssa.Value) bound.Bound { return linear(a, 0) },

	// textproto.CanonicalMIMEHeaderKey walks s a constant number of times.
	"net/http.CanonicalHeaderKey": func(a []ssa.Value) bound.Bound { return linear(a, 0) },

	// ParsePrefix splits s at the last '/' and parses each half; every step is
	// a linear scan of s. MustParsePrefix is ParsePrefix plus a panic.
	"net/netip.ParsePrefix":     func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"net/netip.MustParsePrefix": func(a []ssa.Value) bound.Bound { return linear(a, 0) },

	// Deliberately ABSENT, with reasons, so nobody re-derives them:
	//
	//   - unicode.Is: is16/is32 scan the RANGE TABLE linearly for small tables
	//     and binary search it otherwise, so the cost depends on rangeTab — an
	//     argument other than the one an O(1) entry would be sized by. Fifth
	//     instance of that shape. Pinned by TestUnicodeIsStaysTop.
	//   - bytes.Repeat: genuinely O(len(b)·count), which needs a product of a
	//     length and a numeric argument that no helper here expresses yet. It
	//     was goldmark's largest row (19 functions) and first contact measured
	//     it delivering ~nothing anyway, because neither operand resolves at
	//     the call sites. Worth revisiting WITH argument-size resolution, not
	//     before.

	// --- math/bits, added 2026-08-11 ---
	//
	// Found by running `bigo trust init` against repositories the survey has
	// never analysed. It was the largest STDLIB blocker in that population — 71
	// functions across them, 60 in jaeger alone — and bigo priced none of it.
	//
	// ONE argument covers all 49, and it is about the OPERANDS rather than the
	// bodies: every exported function here takes and returns FIXED-WIDTH
	// integers, so no cost in this package can depend on a program size
	// variable. Two apparent loops are inside COMMENTS (OnesCount64's Hacker's
	// Delight derivation, Sub64's pointer to Sub32); Div64 has two real
	// correction loops, and they iterate a number of times bounded by the word
	// width — a machine constant, not an input size, exactly as GOMAXPROCS is
	// for (*sync.Pool).Put.
	//
	// Enumerated from the package's own type information rather than typed out,
	// and pinned against it by TestMathBitsMatchesStdlib, so a Go release adding
	// a function fails the build instead of going silently unpriced.
	"math/bits.Add":             constCost,
	"math/bits.Add32":           constCost,
	"math/bits.Add64":           constCost,
	"math/bits.Div":             constCost,
	"math/bits.Div32":           constCost,
	"math/bits.Div64":           constCost,
	"math/bits.LeadingZeros":    constCost,
	"math/bits.LeadingZeros16":  constCost,
	"math/bits.LeadingZeros32":  constCost,
	"math/bits.LeadingZeros64":  constCost,
	"math/bits.LeadingZeros8":   constCost,
	"math/bits.Len":             constCost,
	"math/bits.Len16":           constCost,
	"math/bits.Len32":           constCost,
	"math/bits.Len64":           constCost,
	"math/bits.Len8":            constCost,
	"math/bits.Mul":             constCost,
	"math/bits.Mul32":           constCost,
	"math/bits.Mul64":           constCost,
	"math/bits.OnesCount":       constCost,
	"math/bits.OnesCount16":     constCost,
	"math/bits.OnesCount32":     constCost,
	"math/bits.OnesCount64":     constCost,
	"math/bits.OnesCount8":      constCost,
	"math/bits.Rem":             constCost,
	"math/bits.Rem32":           constCost,
	"math/bits.Rem64":           constCost,
	"math/bits.Reverse":         constCost,
	"math/bits.Reverse16":       constCost,
	"math/bits.Reverse32":       constCost,
	"math/bits.Reverse64":       constCost,
	"math/bits.Reverse8":        constCost,
	"math/bits.ReverseBytes":    constCost,
	"math/bits.ReverseBytes16":  constCost,
	"math/bits.ReverseBytes32":  constCost,
	"math/bits.ReverseBytes64":  constCost,
	"math/bits.RotateLeft":      constCost,
	"math/bits.RotateLeft16":    constCost,
	"math/bits.RotateLeft32":    constCost,
	"math/bits.RotateLeft64":    constCost,
	"math/bits.RotateLeft8":     constCost,
	"math/bits.Sub":             constCost,
	"math/bits.Sub32":           constCost,
	"math/bits.Sub64":           constCost,
	"math/bits.TrailingZeros":   constCost,
	"math/bits.TrailingZeros16": constCost,
	"math/bits.TrailingZeros32": constCost,
	"math/bits.TrailingZeros64": constCost,
	"math/bits.TrailingZeros8":  constCost,

	// --- Other stdlib gaps the blind run surfaced, each read on its own ---
	//
	// container/list.New is new(List).Init(), which sets three fields.
	"container/list.New": constCost,
	// (time.Time).UTC sets the location pointer and returns the value; Before,
	// After and Equal compare two fixed-width fields.
	"(time.Time).UTC":    constCost,
	"(time.Time).Before": constCost,
	"(time.Time).After":  constCost,
	"(time.Time).Equal":  constCost,
	// Duration accessors are integer conversions on a single int64.
	"(time.Duration).Nanoseconds":  constCost,
	"(time.Duration).Microseconds": constCost,
	"(time.Duration).Milliseconds": constCost,
	"(time.Duration).Seconds":      constCost,
	// maps.Copy ranges over SRC — argument 1, not the destination.
	"maps.Copy": func(a []ssa.Value) bound.Bound { return linear(a, 1) },
	// maps.Clone copies every entry of m.
	"maps.Clone": func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	// strings.Compare stops at the shorter operand, so O(min) ≤ O(len(a)) —
	// the slices.Equal precedent, and checked against the body rather than
	// assumed from it.
	"strings.Compare": func(a []ssa.Value) bound.Bound { return linear(a, 0) },

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
	"strings.LastIndex": searchCost,
	"strings.IndexByte": func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"strings.SplitN":    searchCost,
	"strings.Title":     func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"bytes.Equal":       func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"bytes.HasPrefix":   func(a []ssa.Value) bound.Bound { return linear(a, 0) },
	"bytes.HasSuffix":   func(a []ssa.Value) bound.Bound { return linear(a, 0) },
}
