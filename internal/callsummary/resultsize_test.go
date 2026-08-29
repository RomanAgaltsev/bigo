package callsummary

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// Ranging over the result of a call is the D8 "range-over-call-result" shape:
// `for _, w := range strings.Split(s, " ")`. go/ssa lowers a slice range to an
// index loop whose limit is len(<the call result>), and a call result is not a
// nameable size root, so the loop was ⊤ with no rule matched.
//
// len(strings.Split(s, sep)) <= len(s)+1, and a constant vanishes
// asymptotically, so the sound answer is O(len(s)).
//
// Measured 2026-08-29 on the kata corpus: this shape blocks searchengine's two
// rows and rpncalc.Calculate, and graduates exactly ONE of them
// (searchengine.UpdateIndex). The other two have second blockers — a map range
// and a func value from a map.
func TestRangeOverCallResult(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{
			"range over strings.Split directly",
			`package input
import "strings"
func f(s string) int {
	n := 0
	for range strings.Split(s, " ") {
		n++
	}
	return n
}`,
			"O(len(s))",
		},
		{
			"result stored in a local first",
			`package input
import "strings"
func f(s string) int {
	parts := strings.Split(s, " ")
	n := 0
	for range parts {
		n++
	}
	return n
}`,
			"O(len(s))",
		},
		{
			"strings.Fields",
			`package input
import "strings"
func f(s string) int {
	n := 0
	for range strings.Fields(s) {
		n++
	}
	return n
}`,
			"O(len(s))",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg, _, err := ssasupport.Build(tc.src)
			if err != nil {
				t.Fatal(err)
			}
			fn := ssasupport.Func(pkg, "f")
			if got := engine.Infer(fn, New(nil)).String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
