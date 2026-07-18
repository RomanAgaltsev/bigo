package costtable

import (
	"sync"
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// copyCall builds src and returns the builtin copy call inside f.
func copyCall(t *testing.T, src string) *ssa.CallCommon {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			c, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			if bi, ok := c.Call.Value.(*ssa.Builtin); ok && bi.Name() == "copy" {
				return &c.Call
			}
		}
	}
	t.Fatal("no copy call in f")
	return nil
}

// copyCost returns the cost-table bound of the builtin copy call in f.
func copyCost(t *testing.T, src string) bound.Bound {
	t.Helper()
	got, ok := Lookup(copyCall(t, src))
	if !ok {
		t.Fatal("copy must be priced")
	}
	return got
}

func TestCopyCostLocalDest(t *testing.T) {
	// Locally-derived destination: the #88 shape. Was ⊤ through v1.28.1.
	got := copyCost(t, `package input
func f(s []int) { d := make([]int, len(s)); copy(d, s) }`)
	if got.String() != "O(len(s))" {
		t.Errorf("cost = %s, want O(len(s))", got.String())
	}
}

func TestCopyCostParamDestUnchanged(t *testing.T) {
	got := copyCost(t, `package input
func f(d, s []int) { copy(d, s) }`)
	if got.String() != "O(len(d))" {
		t.Errorf("cost = %s, want O(len(d))", got.String())
	}
}

func TestCopyCostUnresolvableStaysTop(t *testing.T) {
	// A call-result destination has no derivable length: ⊤, and no panic on
	// values whose Parent() chain is unusual.
	got := copyCost(t, `package input
func g() []int { return nil }
func f(s []int) { copy(g(), s) }`)
	if !got.IsTop() {
		t.Errorf("cost = %s, want unverifiable", got.String())
	}
}

func TestArgExtentConcurrent(t *testing.T) {
	// The Stability memo is shared package state: hammer one call from many
	// goroutines. Run under CI's -race to be meaningful; locally it still
	// exercises LoadOrStore agreement.
	call := copyCall(t, `package input
func f(s []int) { d := make([]int, len(s)); copy(d, s) }`)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				if got, _ := Lookup(call); got.String() != "O(len(s))" {
					panic("concurrent Lookup diverged: " + got.String())
				}
			}
		}()
	}
	wg.Wait()
}
