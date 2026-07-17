package smell

import (
	"slices"
	"testing"
)

// Issue #73: findings anchored at the accumulator's phi carry the *declaration*
// position, so two loops accumulating into one variable collapse to a single
// (rule, file, line) — indistinguishable from a duplicate, and double-counted by
// any consumer tallying findings. A finding must point at the accumulation site,
// which is also where the fix goes.

func TestSM1TwoLoopsAnchorAtAccumulation(t *testing.T) {
	// Issue #73's repro, reduced from nats-server server/errors.go:317.
	src := `package input
func TwoLoops(warnings, errs []string) string {
	var msg string
	for _, w := range warnings {
		msg += w
	}
	for _, e := range errs {
		msg += e
	}
	return msg
}
`
	got := detectLines(t, src, "TwoLoops", "SM1")
	slices.Sort(got)
	want := []int{5, 8} // the two `msg +=` sites, not the line-3 declaration
	if !slices.Equal(got, want) {
		t.Errorf("SM1 anchors: got lines %v, want %v", got, want)
	}
}

func TestSM1SprintfAnchorsAtCall(t *testing.T) {
	src := `package input
import "fmt"
func Sprintf(xs []string) string {
	var s string
	for _, x := range xs {
		s = fmt.Sprintf("%s%s", s, x)
	}
	return s
}
`
	got := detectLines(t, src, "Sprintf", "SM1")
	want := []int{6} // the Sprintf call, not the line-4 declaration
	if !slices.Equal(got, want) {
		t.Errorf("SM1 anchors: got lines %v, want %v", got, want)
	}
}

func TestSM3TwoAccumulatorsAnchorAtAppend(t *testing.T) {
	// Two slices declared on one line: phi-anchored findings collapse to line 3.
	src := `package input
func TwoAppends(a, b []string) ([]string, []string) {
	var xs, ys []string
	for _, x := range a {
		xs = append(xs, x)
	}
	for _, y := range b {
		ys = append(ys, y)
	}
	return xs, ys
}
`
	got := detectLines(t, src, "TwoAppends", "SM3")
	slices.Sort(got) // SM3 iterates a loop-keyed map; order is not fixed here
	want := []int{5, 8}
	if !slices.Equal(got, want) {
		t.Errorf("SM3 anchors: got lines %v, want %v", got, want)
	}
}
