package costtable

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
	"golang.org/x/tools/go/ssa"
)

func costOf(t *testing.T, src string) (string, bool) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	var call *ssa.CallCommon
	for _, b := range fn.Blocks {
		for _, in := range b.Instrs {
			if c, ok := in.(*ssa.Call); ok {
				call = &c.Call
			}
		}
	}
	if call == nil {
		t.Fatal("no call found in f")
	}
	b, ok := Lookup(call)
	return b.String(), ok
}

func TestLookup(t *testing.T) {
	tests := []struct {
		name, src, want string
		ok              bool
	}{
		{"len is O(1)", `package input
func f(xs []int) int { return len(xs) }`, "O(1)", true},
		{"append is amortized O(1)", `package input
func f(xs []int) []int { return append(xs, 1) }`, "O(1)", true},
		{"sort.Ints is n log n", `package input
import "sort"
func f(xs []int) { sort.Ints(xs) }`, "O(len(xs) log(len(xs)))", true},
		{"slices.Contains is linear", `package input
import "slices"
func f(xs []int, v int) bool { return slices.Contains(xs, v) }`, "O(len(xs))", true},
		{"strings.Contains linear in s", `package input
import "strings"
func f(s string) bool { return strings.Contains(s, "x") }`, "O(len(s))", true},
		// This fixture needs a stdlib function that will NEVER be priced, and
		// picking one casually does not achieve that: it used math.Sqrt until
		// the 2026-08-12 blind-repo lane priced the whole math package, and the
		// test then failed for an entirely good reason. net.Dial is a network
		// round trip — unbounded in work and in wall-clock, and bounded by
		// nothing in the program — so no future lane can reach it.
		{"unknown stdlib not in table", `package input
import "net"
func f(a string) (net.Conn, error) { return net.Dial("tcp", a) }`, "O(1)", false},
		{"slices.Max is linear", `package input
import "slices"
func f(xs []int) int { return slices.Max(xs) }`, "O(len(xs))", true},
		{"slices.BinarySearch is log", `package input
import "slices"
func f(xs []int, v int) bool { _, ok := slices.BinarySearch(xs, v); return ok }`, "O(log(len(xs)))", true},
		{"strings.Repeat is s times count", `package input
import "strings"
func f(s string, count int) string { return strings.Repeat(s, count) }`, "O(count len(s))", true},
		{"strings.ToLower is linear", `package input
import "strings"
func f(s string) string { return strings.ToLower(s) }`, "O(len(s))", true},
		{"maps iterator construction is O(1)", `package input
import "maps"
func f(m map[string]int) { _ = maps.Keys(m) }`, "O(1)", true},
		// sync: O(1) work per operation. Blocking under contention is
		// wall-clock, not work, and bigo models work (issue #46).
		{"sync.Mutex.Lock is O(1)", `package input
import "sync"
func f(mu *sync.Mutex) { mu.Lock() }`, "O(1)", true},
		{"sync.Mutex.Unlock is O(1)", `package input
import "sync"
func f(mu *sync.Mutex) { mu.Unlock() }`, "O(1)", true},
		{"sync.RWMutex.RLock is O(1)", `package input
import "sync"
func f(mu *sync.RWMutex) { mu.RLock() }`, "O(1)", true},
		{"sync.RWMutex.RUnlock is O(1)", `package input
import "sync"
func f(mu *sync.RWMutex) { mu.RUnlock() }`, "O(1)", true},
		{"sync.WaitGroup.Wait is O(1) work", `package input
import "sync"
func f(wg *sync.WaitGroup) { wg.Wait() }`, "O(1)", true},
		// sync.Once.Do(f) costs cost(f), NOT O(1). Costing it O(1) would
		// under-approximate a call into a false Within — a wrong bound. It
		// stays out of the table (⊤) until the parametric path models it.
		{"sync.Once.Do is not in the table", `package input
import "sync"
func f(once *sync.Once, g func()) { once.Do(g) }`, "O(1)", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := costOf(t, tt.src)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("cost = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCurated covers both curated tables and the negative case. It matters
// because `bigo trust init` filters on it: a false negative offers the user a
// line that is silently shadowed, and a false positive hides a key they need.
func TestCurated(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"strconv.Atoi", true},              // plain table
		{"bytes.Equal", true},               // plain table
		{"(*sync/atomic.Value).Load", true}, // plain table, method key
		{"sort.Slice", true},                // PARAMETRIC table — also shadows
		{"os.Getenv", false},                // genuinely unpriced
		{"bytes.Repeat", false},             // genuinely unpriced
		{"example.com/x.Whatever", false},
	}
	for _, c := range cases {
		if got := Curated(c.key); got != c.want {
			t.Errorf("Curated(%q) = %v, want %v", c.key, got, c.want)
		}
	}
}
