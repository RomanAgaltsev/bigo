package recurrence

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// ciMergeBody is the merge + two tail loops shared by the staged shapes below.
const ciMergeBody = `
	i, j, k := 0, 0, 0
	for i < len(left) && j < len(right) {
		if left[i] <= right[j] {
			s[k] = left[i]
			i++
		} else {
			s[k] = right[j]
			j++
			inv += len(left) - i
		}
		k++
	}
	for i < len(left) {
		s[k] = left[i]
		i++
		k++
	}
	for j < len(right) {
		s[k] = right[j]
		j++
		k++
	}
	return inv
}`

// ciVerbatim is corpus dandc.CountInversions: recursion on append-copy locals.
const ciVerbatim = `package input
func f(s []int) int {
	if len(s) <= 1 {
		return 0
	}
	mid := len(s) / 2
	left := append([]int(nil), s[:mid]...)
	right := append([]int(nil), s[mid:]...)
	inv := f(left) + f(right)` + ciMergeBody

// ciDirect recurses directly on the slice expressions (isolates B2 from B1).
const ciDirect = `package input
func f(s []int) int {
	if len(s) <= 1 {
		return 0
	}
	mid := len(s) / 2
	left := append([]int(nil), s[:mid]...)
	right := append([]int(nil), s[mid:]...)
	inv := f(s[:mid]) + f(s[mid:])` + ciMergeBody

// ciNoTails drops the two tail loops — the probe's isolating experiment that
// named B2. Kept so the isolate stays green independently of the tails.
const ciNoTails = `package input
func f(s []int) int {
	if len(s) <= 1 {
		return 0
	}
	mid := len(s) / 2
	left := append([]int(nil), s[:mid]...)
	right := append([]int(nil), s[mid:]...)
	inv := f(s[:mid]) + f(s[mid:])
	i, j, k := 0, 0, 0
	for i < len(left) && j < len(right) {
		if left[i] <= right[j] {
			s[k] = left[i]
			i++
		} else {
			s[k] = right[j]
			j++
			inv += len(left) - i
		}
		k++
	}
	return inv
}`

func solveOf(t *testing.T, src string) (string, bool) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	b, _, ok := Solve(ssasupport.Func(pkg, "f"), stubModel{})
	return b.String(), ok
}

// TestSolveCountInversionsFamily is the 2026-07-18 probe's staged diagnosis,
// promoted to a permanent regression suite: each shape isolates one blocker.
func TestSolveCountInversionsFamily(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"reduced", `package input
func f(s []int) int {
	if len(s) <= 1 { return 0 }
	mid := len(s) / 2
	return f(s[:mid]) + f(s[mid:])
}`, "O(len(s))"},
		{"noTails", ciNoTails, "O(len(s) log(len(s)))"},
		{"direct", ciDirect, "O(len(s) log(len(s)))"},
		{"verbatim", ciVerbatim, "O(len(s) log(len(s)))"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := solveOf(t, c.src)
			if !ok || got != c.want {
				t.Errorf("Solve = (%q, %v), want (%q, true)", got, ok, c.want)
			}
		})
	}
}

func TestExtractAppendCopySteps(t *testing.T) {
	// The unwrap classifies the copy by its spread operand.
	r, ok := extractOf(t, `package input
func f(s []int) int {
	if len(s) <= 1 { return 0 }
	mid := len(s) / 2
	left := append([]int(nil), s[:mid]...)
	right := append([]int(nil), s[mid:]...)
	return f(left) + f(right)
}`)
	if !ok || len(r.terms) != 2 {
		t.Fatalf("extract = %+v, %v; want ok with 2 terms", r, ok)
	}
	for _, tm := range r.terms {
		if tm.kind != stepDiv || tm.div != 2 {
			t.Errorf("term = %+v, want Div/2", tm)
		}
	}
}

func TestExtractAppendCopyNonZeroDestRejected(t *testing.T) {
	// A non-zero destination breaks the len equality: must stay unextracted.
	_, ok := extractOf(t, `package input
func f(s []int) int {
	if len(s) <= 1 { return 0 }
	mid := len(s) / 2
	left := append(make([]int, 1), s[:mid]...)
	return f(left)
}`)
	if ok {
		t.Error("non-zero-length destination must not classify as a decrease")
	}
}
