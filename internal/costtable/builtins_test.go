package costtable

import (
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// builtinCallCost returns the cost of the first call to the named builtin in f.
func builtinCallCost(t *testing.T, name, src string) (string, bool) {
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
			if bi, ok := c.Call.Value.(*ssa.Builtin); ok && bi.Name() == name {
				got, priced := Lookup(&c.Call)
				if priced != Priced(&c.Call) {
					t.Errorf("Lookup priced=%v but Priced=%v — the tables have drifted", priced, Priced(&c.Call))
				}
				return got.String(), priced
			}
		}
	}
	t.Fatalf("no call to %s in f", name)
	return "", false
}

func TestModernBuiltins(t *testing.T) {
	cases := []struct {
		name, builtin, src, want string
		priced                   bool
	}{
		{"min numeric is O(1)", "min", `package input
func f(a, b int) int { return min(a, b) }`, "O(1)", true},
		{"max numeric is O(1)", "max", `package input
func f(a, b float64) float64 { return max(a, b) }`, "O(1)", true},
		// A string comparison is O(min(len)); a chain of them is not bounded by
		// any single argument, so strings must stay unpriced.
		{"min string stays unpriced", "min", `package input
func f(a, b string) string { return min(a, b) }`, "", false},
		// clear(slice) zeroes exactly len(s) elements.
		{"clear slice is linear", "clear", `package input
func f(s []int) { clear(s) }`, "O(len(s))", true},
		// clear(map) walks the bucket array, whose size tracks historical
		// capacity, not len — unpriceable without a cap(map) variable.
		{"clear map stays unpriced", "clear", `package input
func f(m map[int]int) { clear(m) }`, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, priced := builtinCallCost(t, c.builtin, c.src)
			if priced != c.priced {
				t.Fatalf("priced = %v, want %v (cost %q)", priced, c.priced, got)
			}
			if priced && got != c.want {
				t.Errorf("cost = %q, want %q", got, c.want)
			}
		})
	}
}
