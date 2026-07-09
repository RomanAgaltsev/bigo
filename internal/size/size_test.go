package size

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/bound"
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
