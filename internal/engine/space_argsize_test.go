package engine

import "testing"

func TestAppendCopyChargesHeap(t *testing.T) {
	// The B3 shape: through v1.28.1 this charged O(1) and, once recurrence
	// solved, produced an oracle-confirmed wrong space bound (probe 2026-07-18).
	got := heapOf(t, `package input
func f(s []int) []int { mid := len(s) / 2; return append([]int(nil), s[:mid]...) }`)
	if got != "O(len(s))" {
		t.Errorf("heap = %s, want O(len(s))", got)
	}
}

func TestMakeDerivedLengthChargesHeap(t *testing.T) {
	got := heapOf(t, `package input
func f(s []int) []int { return make([]int, len(s)/2) }`)
	if got != "O(len(s))" {
		t.Errorf("heap = %s, want O(len(s))", got)
	}
}

func TestScalarAppendStaysConstant(t *testing.T) {
	got := heapOf(t, `package input
func f(s []int, x int) []int { return append(s, x) }`)
	if got != "O(1)" {
		t.Errorf("heap = %s, want O(1)", got)
	}
}
