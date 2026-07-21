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
