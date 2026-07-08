package annotation

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

func TestParseWhereAndBindings(t *testing.T) {
	d, err := Parse("//bigo:max O(n*m) where n=len(a), m=len(b)")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := d.Budget.String(), "O(m n)"; got != want {
		t.Errorf("Budget = %q, want %q", got, want)
	}
	want := map[bound.Var]SizeRef{
		"n": {Kind: Len, Param: "a"},
		"m": {Kind: Len, Param: "b"},
	}
	if len(d.Bindings) != len(want) {
		t.Fatalf("bindings = %v, want %v", d.Bindings, want)
	}
	for k, v := range want {
		if d.Bindings[k] != v {
			t.Errorf("binding %q = %v, want %v", k, d.Bindings[k], v)
		}
	}
}

func TestParseWhereKinds(t *testing.T) {
	got, err := parseWhere("n=len(a), m=cap(b), k=count")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[bound.Var]SizeRef{
		"n": {Kind: Len, Param: "a"},
		"m": {Kind: Cap, Param: "b"},
		"k": {Kind: Num, Param: "count"},
	}
	for v, want := range cases {
		if got[v] != want {
			t.Errorf("binding %q = %v, want %v", v, got[v], want)
		}
	}
}

func TestParseWhereErrors(t *testing.T) {
	bad := []string{"n", "n=", "=len(a)", "n=len()", "n=len(a", "n=1a"}
	for _, in := range bad {
		if _, err := parseWhere(in); err == nil {
			t.Errorf("parseWhere(%q) expected error, got nil", in)
		}
	}
}
