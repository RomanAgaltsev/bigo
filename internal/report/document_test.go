package report

import (
	"reflect"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

func TestBoundJSONTop(t *testing.T) {
	got := boundJSON(bound.Top())
	if !got.Top || got.Str != "" || got.Terms != nil {
		t.Errorf("boundJSON(Top) = %+v, want {Top:true}", got)
	}
}

func TestBoundJSONConstant(t *testing.T) {
	got := boundJSON(bound.Constant())
	want := BoundJSON{Str: "O(1)", Terms: []map[string]FactorJSON{{}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("boundJSON(O(1)) = %+v, want %+v", got, want)
	}
}

func TestBoundJSONPolyLog(t *testing.T) {
	// n · log n
	got := boundJSON(bound.Of(bound.Mono("n", 1, 1)))
	want := BoundJSON{
		Str:   bound.Of(bound.Mono("n", 1, 1)).String(),
		Terms: []map[string]FactorJSON{{"n": {Pow: 1, Log: 1}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("boundJSON(n log n) = %+v, want %+v", got, want)
	}
}

func TestBoundJSONAntichainSorted(t *testing.T) {
	// Incomparable antichain {m, n}: terms must come out sorted by canonical string.
	b := bound.Of(bound.Term("n"), bound.Term("m"))
	got := boundJSON(b)
	if len(got.Terms) != 2 {
		t.Fatalf("want 2 terms, got %+v", got)
	}
	want := []map[string]FactorJSON{{"m": {Pow: 1}}, {"n": {Pow: 1}}}
	if !reflect.DeepEqual(got.Terms, want) {
		t.Errorf("terms = %+v, want sorted %+v", got.Terms, want)
	}
}

func TestVerdictString(t *testing.T) {
	cases := map[bound.Verdict]string{
		bound.Within:  "within",
		bound.Exceeds: "exceeds",
		bound.Unknown: "unverifiable", // schema vocabulary, not Verdict.String()'s "unknown"
	}
	for v, want := range cases {
		if got := verdictString(v); got != want {
			t.Errorf("verdictString(%v) = %q, want %q", v, got, want)
		}
	}
}
