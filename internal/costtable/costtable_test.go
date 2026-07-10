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
		{"unknown stdlib not in table", `package input
import "math"
func f(x float64) float64 { return math.Sqrt(x) }`, "O(1)", false},
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
