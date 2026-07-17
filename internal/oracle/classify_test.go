package oracle

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

func TestClassify(t *testing.T) {
	n := bound.Of(bound.Term("n"))
	n2 := bound.Of(bound.Mono("n", 2, 0))
	nlogn := bound.Of(bound.Mono("n", 1, 1))
	nm := bound.Of(bound.Term("n").Mul(bound.Term("m")))
	for _, tc := range []struct {
		name         string
		emitted, pin bound.Bound
		want         Status
	}{
		{"equal is exact", n, n, Exact},
		{"dominating is loose", n2, n, Loose},
		{"n log n dominates n", nlogn, n, Loose},
		{"below is wrong", n, n2, Wrong},
		{"below nlogn is wrong", n, nlogn, Wrong},
		{"incomparable is wrong", n2, nm, Wrong},
		{"top is top", bound.Top(), n, Top},
	} {
		if got := Classify(tc.emitted, tc.pin); got != tc.want {
			t.Errorf("%s: Classify = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestStatusString(t *testing.T) {
	for s, want := range map[Status]string{Wrong: "wrong", Exact: "exact", Loose: "loose", Top: "top"} {
		if s.String() != want {
			t.Errorf("%v.String() = %q, want %q", s, s.String(), want)
		}
	}
}
