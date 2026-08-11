package costtable

import (
	"go/types"
	"sort"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// atomicValueMethods is the deliberate refusal, now down to one. Load, Store
// and Swap were priced in the F3 fix; CompareAndSwap stays out because it
// compares the stored value with a runtime interface equality check, which
// costs O(len) on a string- or array-bearing dynamic type — not a contention
// question and not soundly priceable. See the entry comment in costtable.go.
//
// Listing it HERE rather than omitting it silently is the point: the parity
// check below then has no unexplained gap, and shrinking this list is how a
// change of mind about a refusal announces itself.
var atomicValueMethods = map[string]bool{
	"(*sync/atomic.Value).CompareAndSwap": true,
}

// TestAtomicTableMatchesStdlib pins the cost table's sync/atomic key set
// against the real package.
//
// Both directions matter and neither is visible any other way. A key in the
// table that no longer exists is DEAD: it never fires, and Priced simply
// answers false, so no test and no golden would notice. A stdlib function
// missing from the table is a silent capability gap — which is precisely how
// the package-level API drifted to 22 of 39 behind the typed one (review F4).
//
// TRADEOFF, deliberate: this couples the test to the toolchain's stdlib, so a
// Go release that adds an atomic method fails this test until the entry is
// priced. That is the alarm working. Do NOT weaken it to a subset check —
// a subset check would have caught neither F4 nor a typo.
func TestAtomicTableMatchesStdlib(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedTypes | packages.NeedName,
	}, "sync/atomic")
	if err != nil {
		t.Fatalf("load sync/atomic: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Types == nil {
		t.Fatalf("want exactly one typed package, got %d", len(pkgs))
	}

	want := map[string]bool{}
	scope := pkgs[0].Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		switch o := obj.(type) {
		case *types.Func:
			want["sync/atomic."+o.Name()] = true
		case *types.TypeName:
			nt, ok := o.Type().(*types.Named)
			if !ok {
				continue
			}
			for i := 0; i < nt.NumMethods(); i++ {
				if m := nt.Method(i); m.Exported() && !atomicValueMethods[m.FullName()] {
					want[m.FullName()] = true
				}
			}
		}
	}

	got := map[string]bool{}
	for key := range stdlib {
		if isAtomicKey(key) {
			got[key] = true
		}
	}

	for _, k := range sortedKeys(want) {
		if !got[k] {
			t.Errorf("stdlib exports %s and the table does not price it", k)
		}
	}
	for _, k := range sortedKeys(got) {
		if !want[k] {
			t.Errorf("table prices %s, which the stdlib does not export — a dead key never fires", k)
		}
	}
}

// isAtomicKey reports whether a cost-table key names a sync/atomic function or
// method, in either of the two forms FuncKey produces.
func isAtomicKey(key string) bool {
	return strings.HasPrefix(key, "sync/atomic.") || strings.HasPrefix(key, "(*sync/atomic.")
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
