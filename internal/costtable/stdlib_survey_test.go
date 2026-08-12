package costtable

import "testing"

// TestSurveyRankedEntries covers the cost-table lane — entries added because
// the v1.34.0 real-world survey measured them blocking real Go, each with a
// documented bound. See the spec's §3 table for the per-entry argument.
func TestSurveyRankedEntries(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		// --- O(1): constant work ---
		{"errors.New", `package input
import "errors"
func f() error { return errors.New("boom") }`, "O(1)"},

		{"time.Now", `package input
import "time"
func f() time.Time { return time.Now() }`, "O(1)"},

		{"time.Since", `package input
import "time"
func f(t0 time.Time) time.Duration { return time.Since(t0) }`, "O(1)"},

		{"context.Background", `package input
import "context"
func f() context.Context { return context.Background() }`, "O(1)"},

		// At most 20 digits for any int — constant-bounded, not linear in the
		// value.
		{"strconv.Itoa", `package input
import "strconv"
func f(n int) string { return strconv.Itoa(n) }`, "O(1)"},

		{"strconv.FormatInt", `package input
import "strconv"
func f(n int64) string { return strconv.FormatInt(n, 10) }`, "O(1)"},

		{"math.Float64bits", `package input
import "math"
func f(x float64) uint64 { return math.Float64bits(x) }`, "O(1)"},

		{"atomic.LoadUint64", `package input
import "sync/atomic"
func f(p *uint64) uint64 { return atomic.LoadUint64(p) }`, "O(1)"},

		// reflect: constant work on the interface header. This prices the
		// call's own cost and claims NOTHING about what the program does with
		// reflection — see the spec's §3 note.
		{"reflect.TypeOf", `package input
import "reflect"
func f(v any) reflect.Type { return reflect.TypeOf(v) }`, "O(1)"},

		{"reflect.Value.Kind", `package input
import "reflect"
func f(rv reflect.Value) reflect.Kind { return rv.Kind() }`, "O(1)"},

		// --- Linear in argument 0 ---
		{"strconv.Atoi", `package input
import "strconv"
func f(s string) (int, error) { return strconv.Atoi(s) }`, "O(len(s))"},

		// O(len(suffix)) <= O(len(s)) — the strings.HasPrefix precedent.
		{"strings.HasSuffix", `package input
import "strings"
func f(s, x string) bool { return strings.HasSuffix(s, x) }`, "O(len(s))"},

		{"strings.TrimPrefix", `package input
import "strings"
func f(s, x string) string { return strings.TrimPrefix(s, x) }`, "O(len(s))"},

		// O(min) <= O(len(a)) — the slices.Equal precedent.
		{"bytes.Equal", `package input
import "bytes"
func f(a, b []byte) bool { return bytes.Equal(a, b) }`, "O(len(a))"},

		// --- The Trim family: the PRODUCT, not the first argument ---
		//
		// These do not compare two sequences, they test MEMBERSHIP of every
		// rune of s in cutset, and both of Go's paths carry a cutset term:
		// makeASCIISet walks the whole cutset before any trimming, and a
		// cutset holding any non-ASCII byte falls back to trimLeftUnicode,
		// which calls ContainsRune — a scan of cutset — once per rune of s.
		//
		// Priced O(len(s)) from v1.35.0 to v1.38.0 by inheriting the
		// HasPrefix/slices.Equal precedent, which does not apply to them.
		// That was the SEVENTH wrong bound; see the 2026-07-21 review, F1.
		{"strings.Trim", `package input
import "strings"
func f(s, cutset string) string { return strings.Trim(s, cutset) }`, "O(len(cutset) len(s))"},

		{"strings.TrimLeft", `package input
import "strings"
func f(s, cutset string) string { return strings.TrimLeft(s, cutset) }`, "O(len(cutset) len(s))"},

		{"strings.TrimRight", `package input
import "strings"
func f(s, cutset string) string { return strings.TrimRight(s, cutset) }`, "O(len(cutset) len(s))"},

		// A CONSTANT cutset keeps the linear bound: a compile-time string has a
		// fixed length and contributes only a constant factor. This is the
		// common real-world shape, and it is a POSITIVE CONTROL — the first fix
		// for F1 priced the product unconditionally and turned
		// strings.Trim(s, " \t\n") into ⊤, a silent capability loss of exactly
		// the kind C5 already cost this project once.
		{"strings.Trim const cutset", `package input
import "strings"
func f(s string) string { return strings.Trim(s, " \t\n") }`, "O(len(s))"},

		{"strings.TrimLeft const cutset", `package input
import "strings"
func f(s string) string { return strings.TrimLeft(s, "x") }`, "O(len(s))"},

		// --- The OUTPUT-AMPLIFIED family: also a product (2026-08-12 review F1) ---
		//
		// The EIGHTH wrong bound, and it is the seventh's class with the sweep
		// that was owed after it. Trim multiplies because it READS its second
		// argument once per element of its first; these multiply because they
		// WRITE their second argument once per element of the first. Different
		// mechanism, identical arithmetic, and the same repair.
		//
		// Join writes sep exactly len(elems)-1 times (strings.go's
		// b.WriteString(sep) loop), so the cost carries a len(sep) factor no
		// matter what the elements hold. Priced O(len(elems)) since the entry
		// was added, which is not an upper bound whenever sep is a variable.
		{"strings.Join", `package input
import "strings"
func f(elems []string, sep string) string { return strings.Join(elems, sep) }`, "O(len(elems) len(sep))"},

		// Replace writes `new` once per replacement, and the replacement count
		// is Count(s, old), which is bounded only by len(s) — so the cost
		// carries a len(new) factor. Note the arg index: `new` is argument 2,
		// not 1. Sizing it by `old` would be a different wrong bound.
		{"strings.Replace", `package input
import "strings"
func f(s, old, nw string, n int) string { return strings.Replace(s, old, nw, n) }`, "O(len(nw) len(s))"},

		{"strings.ReplaceAll", `package input
import "strings"
func f(s, old, nw string) string { return strings.ReplaceAll(s, old, nw) }`, "O(len(nw) len(s))"},

		// POSITIVE CONTROLS. A constant separator or replacement collapses the
		// product exactly as a constant cutset does, and these are the common
		// real shapes — strings.Join(parts, ", ") is everywhere. Pinning only
		// the product cases would pass throughout the defect's life AND would
		// pass again if someone later priced the product unconditionally, which
		// is the C5 capability loss this project has now paid for twice.
		{"strings.Join const sep", `package input
import "strings"
func f(elems []string) string { return strings.Join(elems, ", ") }`, "O(len(elems))"},

		{"strings.ReplaceAll const new", `package input
import "strings"
func f(s, old string) string { return strings.ReplaceAll(s, old, "-") }`, "O(len(s))"},

		// --- The substring-search family: sweep item C, decided 2026-08-12 ---
		//
		// A needle longer than bytealg.MaxLen falls through to IndexRabinKarp,
		// which verifies every hash match with a full compare against a
		// compile-time PrimeRK, so collisions are constructible and the worst
		// case is the product. Below that length the comparison is bounded by a
		// machine constant and the linear bound is sound.
		//
		// The 2026-08-12 review declined to call this a break because the worst
		// case is adversarial. Priced as the product anyway: the directive is a
		// WORST-CASE bound, and "holds for typical inputs" is the reasoning
		// that produced Trim.
		{"strings.Index", `package input
import "strings"
func f(s, sub string) int { return strings.Index(s, sub) }`, "O(len(s) len(sub))"},

		{"strings.Split", `package input
import "strings"
func f(s, sep string) []string { return strings.Split(s, sep) }`, "O(len(s) len(sep))"},

		{"bytes.Index", `package input
import "bytes"
func f(b, sub []byte) int { return bytes.Index(b, sub) }`, "O(len(b) len(sub))"},

		// POSITIVE CONTROLS. A literal needle is the overwhelming majority of
		// real calls and keeps the linear bound; without these the decision
		// would be a coverage loss wearing a soundness argument.
		{"strings.Contains const needle", `package input
import "strings"
func f(s string) bool { return strings.Contains(s, "://") }`, "O(len(s))"},

		{"strings.Split const sep", `package input
import "strings"
func f(s string) []string { return strings.Split(s, ",") }`, "O(len(s))"},

		// IndexByte is NOT in the family: a single byte has no length to
		// multiply, so widening it would be a pure capability loss.
		{"strings.IndexByte stays linear", `package input
import "strings"
func f(s string, c byte) int { return strings.IndexByte(s, c) }`, "O(len(s))"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := costOf(t, tt.src)
			if !ok {
				t.Fatalf("Lookup = not found, want %s", tt.want)
			}
			if got != tt.want {
				t.Errorf("cost = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSurveyExcludedStayTop is the prime-directive pin of this lane, and it
// matters more than the positive cases above.
//
// Every callee here is high-volume in the survey — fmt.Errorf alone is 5,431
// sites — which makes pricing them permanently tempting. They cannot be priced:
//
//   - the fmt family's cost depends on the VALUES, not the format string. `%v`
//     on a slice or map is O(n), and on any type with a String() method it is
//     arbitrary user code. Neither O(1) nor O(len(format)) is an upper bound.
//   - json.Marshal/Unmarshal recurse over arbitrary value graphs.
//   - errors.Is walks an unwrap chain of unbounded depth, calling user-defined
//     Is methods on the way.
//   - reflect.Value.Call invokes arbitrary code; Interface copies a value.
//
// A naive entry for any of these turns this test red, which is the point.
func TestSurveyExcludedStayTop(t *testing.T) {
	tests := []struct{ name, src string }{
		{"fmt.Errorf", `package input
import "fmt"
func f(x any) error { return fmt.Errorf("%v", x) }`},

		{"fmt.Sprintf", `package input
import "fmt"
func f(x any) string { return fmt.Sprintf("%v", x) }`},

		{"json.Marshal", `package input
import "encoding/json"
func f(x any) ([]byte, error) { return json.Marshal(x) }`},

		{"errors.Is", `package input
import "errors"
func f(a, b error) bool { return errors.Is(a, b) }`},

		{"reflect.Value.Call", `package input
import "reflect"
func f(rv reflect.Value, in []reflect.Value) []reflect.Value { return rv.Call(in) }`},

		{"reflect.Value.Interface", `package input
import "reflect"
func f(rv reflect.Value) any { return rv.Interface() }`},

		// Excluded for lack of an expressible bound rather than for danger:
		// the cost is the SUM of variadic element lengths.
		{"filepath.Join", `package input
import "path/filepath"
func f(a, b string) string { return filepath.Join(a, b) }`},

		// Cost is the file's size, which is not a program size variable.
		{"os.ReadFile", `package input
import "os"
func f(p string) ([]byte, error) { return os.ReadFile(p) }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := costOf(t, tt.src)
			if ok && got != "unverifiable" {
				t.Errorf("cost = %q, want unverifiable — this callee has no sound bound; pricing it would be a wrong bound", got)
			}
		})
	}
}

// TestSurveyLinearUnresolvedArgStaysTop: a priced-linear entry whose argument
// size does not resolve must yield ⊤, never a fabricated constant.
func TestSurveyLinearUnresolvedArgStaysTop(t *testing.T) {
	got, ok := costOf(t, `package input
import "strconv"
func g() string { return "x" }
func f() (int, error) { return strconv.Atoi(g()) }`)
	if ok && got != "unverifiable" {
		t.Errorf("cost = %q, want unverifiable — the argument's size is unknown", got)
	}
}

// TestTypedAtomicPriced pins the typed sync/atomic API added 2026-07-24 after
// the S1 truthfulness probe. Positive controls: every one of these compiles to
// a single hardware instruction, so a regression to ⊤ is a silent capability
// loss (the C5 lesson) and a regression to anything else is a wrong bound.
func TestTypedAtomicPriced(t *testing.T) {
	tests := []struct{ name, src string }{
		{"Bool.Load", `package input
import "sync/atomic"
func f(b *atomic.Bool) bool { return b.Load() }`},

		{"Int64.Add", `package input
import "sync/atomic"
func f(n *atomic.Int64) int64 { return n.Add(1) }`},

		{"Uint32.CompareAndSwap", `package input
import "sync/atomic"
func f(n *atomic.Uint32) bool { return n.CompareAndSwap(1, 2) }`},

		{"Int32.Or", `package input
import "sync/atomic"
func f(n *atomic.Int32) int32 { return n.Or(4) }`},

		// The generic type: its key must resolve through Origin(), which is the
		// one member of the family whose keying is not obvious.
		{"Pointer.Load", `package input
import "sync/atomic"
type T struct{ x int }
func f(p *atomic.Pointer[T]) *T { return p.Load() }`},

		{"Pool.Put", `package input
import "sync"
func f(p *sync.Pool, x any) { p.Put(x) }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := costOf(t, tt.src)
			if !ok || got != "O(1)" {
				t.Errorf("cost = %q (priced=%v), want O(1)", got, ok)
			}
		})
	}
}

// TestAtomicAndPoolRefusalsStayTop is the no-fire half, and it is the one that
// matters: each of these was refused for a written reason during the S1 probe,
// and pricing any of them O(1) would be a wrong bound.
func TestAtomicAndPoolRefusalsStayTop(t *testing.T) {
	tests := []struct{ name, src string }{
		// An empty pool calls p.New — a func-typed struct field, i.e. arbitrary
		// user code — and getSlow loops over every P.
		{"Pool.Get", `package input
import "sync"
func f(p *sync.Pool) any { return p.Get() }`},

		// (*atomic.Value).CompareAndSwap compares the stored value with `i !=
		// old`, a RUNTIME INTERFACE EQUALITY check, and the stdlib comment says
		// that is deliberate: "This allows value types to be compared". If the
		// dynamic type contains a string, that comparison is O(len(string)) and
		// no length is nameable at the call site — the same shape as min/max on
		// strings (v1.30.1). This is NOT a contention question, and it is the
		// one member of Value that must never be priced (2026-08-11 review F3).
		{"atomic.Value.CompareAndSwap", `package input
import "sync/atomic"
func f(v *atomic.Value, a, b any) bool { return v.CompareAndSwap(a, b) }`},

		// Cost IS the callback's; it needs the parametric table, which cannot
		// key a method today.
		{"Once.Do", `package input
import "sync"
func f(o *sync.Once, g func()) { o.Do(g) }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := costOf(t, tt.src)
			if ok && got != "unverifiable" {
				t.Errorf("cost = %q, want unverifiable — pricing this would be a wrong bound", got)
			}
		})
	}
}

// TestPackageLevelAtomicPriced closes the mirror image of the hole v1.40.0
// existed to fix (2026-08-11 review F4): the typed API was priced completely
// while the package-level API it delegates to was not, so
// (*atomic.Pointer[T]).Store was O(1) and atomic.StorePointer was ⊤.
//
// All 39 are assembly-implemented intrinsics declared with no Go body in
// doc.go / doc_32.go / doc_64.go, which is the whole soundness argument — read
// from those files, not inherited from the 22 entries added in v1.35.0.
func TestPackageLevelAtomicPriced(t *testing.T) {
	tests := []struct{ name, src string }{
		{"AndInt32", `package input
import "sync/atomic"
func f(p *int32) int32 { return atomic.AndInt32(p, 3) }`},

		{"OrUint64", `package input
import "sync/atomic"
func f(p *uint64) uint64 { return atomic.OrUint64(p, 3) }`},

		{"AddUintptr", `package input
import "sync/atomic"
func f(p *uintptr) uintptr { return atomic.AddUintptr(p, 1) }`},

		{"LoadUintptr", `package input
import "sync/atomic"
func f(p *uintptr) uintptr { return atomic.LoadUintptr(p) }`},

		{"CompareAndSwapUintptr", `package input
import "sync/atomic"
func f(p *uintptr) bool { return atomic.CompareAndSwapUintptr(p, 1, 2) }`},

		{"StorePointer", `package input
import (
	"sync/atomic"
	"unsafe"
)
func f(p *unsafe.Pointer, v unsafe.Pointer) { atomic.StorePointer(p, v) }`},

		{"SwapPointer", `package input
import (
	"sync/atomic"
	"unsafe"
)
func f(p *unsafe.Pointer, v unsafe.Pointer) unsafe.Pointer { return atomic.SwapPointer(p, v) }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := costOf(t, tt.src)
			if !ok || got != "O(1)" {
				t.Errorf("cost = %q (priced=%v), want O(1)", got, ok)
			}
		})
	}
}

// TestAtomicValuePriced is the positive-control half of review F3. Load, Store
// and Swap were refused with the WHOLE TYPE for a reason that describes only
// Store's loop, and Load has no loop at all — a precondition stricter than
// soundness requires is a silent capability loss (the C5 lesson), and this one
// had been pinned, so the pin preserved the loss.
//
// Store and Swap are priced under the SAME doctrine the sync block above
// already applies to (*sync.Mutex).Lock, whose lockSlow also spins: their loops
// continue only when another goroutine won the first-store CAS or is mid first
// store, so no iteration count depends on any input size.
//
// CompareAndSwap is deliberately NOT here — see the refusal test above.
func TestAtomicValuePriced(t *testing.T) {
	tests := []struct{ name, src string }{
		{"Value.Load", `package input
import "sync/atomic"
func f(v *atomic.Value) any { return v.Load() }`},

		{"Value.Store", `package input
import "sync/atomic"
func f(v *atomic.Value, x any) { v.Store(x) }`},

		{"Value.Swap", `package input
import "sync/atomic"
func f(v *atomic.Value, x any) any { return v.Swap(x) }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := costOf(t, tt.src)
			if !ok || got != "O(1)" {
				t.Errorf("cost = %q (priced=%v), want O(1)", got, ok)
			}
		})
	}
}

// TestFirstContactEntries covers the lane added 2026-08-11, after
// `bigo trust init` was run against four real repositories and most of what it
// offered users turned out to be bigo's own unfilled table.
//
// The lane is deliberately weighted to CONSTANTS. First contact measured the
// split: constant entries delivered ~93% of their graduation count (pgx, 27 of
// 29), while a size-dependent one delivered 6% (goldmark, 2 of 32), because the
// argument size does not resolve at most real call sites.
func TestFirstContactEntries(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{"context.WithValue", `package input
import "context"
func f(ctx context.Context, k, v any) context.Context { return context.WithValue(ctx, k, v) }`, "O(1)"},

		{"time.Time.IsZero", `package input
import "time"
func f(t time.Time) bool { return t.IsZero() }`, "O(1)"},

		{"log.New", `package input
import (
	"io"
	"log"
)
func f(w io.Writer) *log.Logger { return log.New(w, "p", 0) }`, "O(1)"},

		{"netip.Prefix.Contains", `package input
import "net/netip"
func f(p netip.Prefix, a netip.Addr) bool { return p.Contains(a) }`, "O(1)"},

		{"binary.BigEndian.Uint32", `package input
import "encoding/binary"
func f(b []byte) uint32 { return binary.BigEndian.Uint32(b) }`, "O(1)"},

		{"binary.LittleEndian.Uint64", `package input
import "encoding/binary"
func f(b []byte) uint64 { return binary.LittleEndian.Uint64(b) }`, "O(1)"},

		{"binary.BigEndian.PutUint16", `package input
import "encoding/binary"
func f(b []byte) { binary.BigEndian.PutUint16(b, 7) }`, "O(1)"},

		// AMENDED 2026-08-12 (sweep item C). This pinned O(len(s)) when the
		// entry was added, on the stated ground that Cut is exactly as sound as
		// strings.Index — which it is, and Index was not sound for a VARIABLE
		// separator. Both are searchCost now, so a variable separator is the
		// product and a literal one keeps the linear bound. The pin moved
		// because the answer was wrong, not because the test was inconvenient.
		{"strings.Cut", `package input
import "strings"
func f(s, sep string) (string, string, bool) { return strings.Cut(s, sep) }`, "O(len(s) len(sep))"},

		{"strings.Cut const sep", `package input
import "strings"
func f(s string) (string, string, bool) { return strings.Cut(s, "=") }`, "O(len(s))"},

		{"bytes.IndexByte", `package input
import "bytes"
func f(b []byte) int { return bytes.IndexByte(b, 'x') }`, "O(len(b))"},

		{"http.CanonicalHeaderKey", `package input
import "net/http"
func f(s string) string { return http.CanonicalHeaderKey(s) }`, "O(len(s))"},

		{"netip.MustParsePrefix", `package input
import "net/netip"
func f(s string) netip.Prefix { return netip.MustParsePrefix(s) }`, "O(len(s))"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := costOf(t, tt.src)
			if !ok || got != tt.want {
				t.Errorf("cost = %q (priced=%v), want %q", got, ok, tt.want)
			}
		})
	}
}

// TestUnicodeIsStaysTop is the refusal this lane earned by reading rather than
// assuming. The first-contact note listed unicode.Is as bigo-priceable; it is
// not. is16/is32 scan the RANGE TABLE linearly for small tables and binary
// search it otherwise, so the cost depends on rangeTab — an argument other than
// the one an O(1) entry would be sized by.
//
// Fifth instance of that shape, after clear on maps, min/max on strings, the
// Trim cutset, and (*sync/atomic.Value).CompareAndSwap.
func TestUnicodeIsStaysTop(t *testing.T) {
	got, ok := costOf(t, `package input
import "unicode"
func f(r rune) bool { return unicode.Is(unicode.Latin, r) }`)
	if ok && got != "unverifiable" {
		t.Errorf("cost = %q, want unverifiable — the cost depends on the range table", got)
	}
}

// TestBlindRepoEntries covers the lane found by running `bigo trust init`
// against repositories the survey has never analysed — the same discovery
// method that produced the first-contact lane, applied to an unseen population.
func TestBlindRepoEntries(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{"bits.Len64", `package input
import "math/bits"
func f(x uint64) int { return bits.Len64(x) }`, "O(1)"},

		{"bits.OnesCount64", `package input
import "math/bits"
func f(x uint64) int { return bits.OnesCount64(x) }`, "O(1)"},

		// Div64 has two real correction loops, bounded by the word width.
		{"bits.Div64", `package input
import "math/bits"
func f(hi, lo, y uint64) (uint64, uint64) { return bits.Div64(hi, lo, y) }`, "O(1)"},

		{"list.New", `package input
import "container/list"
func f() *list.List { return list.New() }`, "O(1)"},

		{"time.Time.UTC", `package input
import "time"
func f(t time.Time) time.Time { return t.UTC() }`, "O(1)"},

		{"time.Time.Before", `package input
import "time"
func f(a, b time.Time) bool { return a.Before(b) }`, "O(1)"},

		{"time.Duration.Seconds", `package input
import "time"
func f(d time.Duration) float64 { return d.Seconds() }`, "O(1)"},

		// maps.Copy ranges over SRC, so the bound names argument 1.
		{"maps.Copy", `package input
import "maps"
func f(dst, src map[string]int) { maps.Copy(dst, src) }`, "O(len(src))"},

		{"maps.Clone", `package input
import "maps"
func f(m map[string]int) map[string]int { return maps.Clone(m) }`, "O(len(m))"},

		{"strings.Compare", `package input
import "strings"
func f(a, b string) int { return strings.Compare(a, b) }`, "O(len(a))"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := costOf(t, tt.src)
			if !ok || got != tt.want {
				t.Errorf("cost = %q (priced=%v), want %q", got, ok, tt.want)
			}
		})
	}
}
