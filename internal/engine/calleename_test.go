package engine

import (
	"slices"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
	"golang.org/x/tools/go/ssa"
)

// Issue #47: cause text rendered callees bare ("call to Now", "call to Verify"),
// so delegation to a same-named callee read as self-recursion — a user grepping
// the report for recursion by matching funcname to callee got a wall of false
// positives. Every static callee is now qualified, and every method names its
// receiver type.
func TestCalleeNameIsQualified(t *testing.T) {
	src := `package input
import ("time"; "sync")
type Verifier interface{ Verify() error }
func local(x int) int { return x }
func f(v Verifier, mu *sync.Mutex, g func(int) int) {
	_ = time.Now()
	mu.Lock()
	_ = v.Verify()
	_ = local(1)
	_ = g(1)
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, b := range ssasupport.Func(pkg, "f").Blocks {
		for _, in := range b.Instrs {
			if c, ok := in.(*ssa.Call); ok {
				got = append(got, calleeName(&c.Call))
			}
		}
	}
	want := []string{
		"time.Now",                // stdlib function: package-qualified
		"(*sync.Mutex).Lock",      // method: receiver-qualified
		"(input.Verifier).Verify", // interface dispatch: interface-qualified
		"input.local",             // same-package function: still qualified
		"g",                       // func value: no static target to qualify
	}
	if !slices.Equal(got, want) {
		t.Errorf("calleeName:\n got %q\nwant %q", got, want)
	}
}
