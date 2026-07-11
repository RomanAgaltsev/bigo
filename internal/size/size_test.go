package size

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func TestCanonicalSizeVars(t *testing.T) {
	if Len("xs") != bound.Var("len(xs)") {
		t.Errorf("Len = %q", Len("xs"))
	}
	if Cap("b") != bound.Var("cap(b)") {
		t.Errorf("Cap = %q", Cap("b"))
	}
	if Num("n") != bound.Var("n") {
		t.Errorf("Num = %q", Num("n"))
	}
}

func TestFromRef(t *testing.T) {
	cases := []struct {
		ref  annotation.SizeRef
		want bound.Var
	}{
		{annotation.SizeRef{Kind: annotation.Len, Param: "a"}, "len(a)"},
		{annotation.SizeRef{Kind: annotation.Cap, Param: "b"}, "cap(b)"},
		{annotation.SizeRef{Kind: annotation.Num, Param: "k"}, "k"},
	}
	for _, c := range cases {
		if got := FromRef(c.ref); got != c.want {
			t.Errorf("FromRef(%v) = %q, want %q", c.ref, got, c.want)
		}
	}
}

func TestValue(t *testing.T) {
	const src = `package input
func f(xs []int, n int, s string, x float64) {}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	params := fn.Params
	cases := []struct {
		i    int
		want bound.Var
		ok   bool
	}{
		{0, "len(xs)", true}, // slice
		{1, "n", true},       // int
		{2, "len(s)", true},  // string
		{3, "", false},       // float64: not a size
	}
	for _, c := range cases {
		got, ok := Value(params[c.i])
		if ok != c.ok || got != c.want {
			t.Errorf("Value(param %d) = (%q,%v), want (%q,%v)", c.i, got, ok, c.want, c.ok)
		}
	}
}

func TestValueClass(t *testing.T) {
	const src = `package input
func f(xs []int, n int, s string) {}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if v, c, ok := ValueClass(fn.Params[0]); !ok || v != "len(xs)" || c != Length {
		t.Errorf("slice: got (%q,%v,%v), want (len(xs), Length, true)", v, c, ok)
	}
	if v, c, ok := ValueClass(fn.Params[1]); !ok || v != "n" || c != Numeric {
		t.Errorf("int: got (%q,%v,%v), want (n, Numeric, true)", v, c, ok)
	}
	if v, c, ok := ValueClass(fn.Params[2]); !ok || v != "len(s)" || c != Length {
		t.Errorf("string: got (%q,%v,%v), want (len(s), Length, true)", v, c, ok)
	}
}

func TestIsFieldPath(t *testing.T) {
	for v, want := range map[bound.Var]bool{
		"len(s.items)":     true,
		"cap(s.buf)":       true,
		"s.limit":          true,
		"len(s.cfg.items)": true,
		"len(xs)":          false,
		"cap(xs)":          false,
		"n":                false,
	} {
		if got := IsFieldPath(v); got != want {
			t.Errorf("IsFieldPath(%q) = %v, want %v", v, got, want)
		}
	}
}
