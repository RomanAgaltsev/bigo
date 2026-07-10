package normalize

import (
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

type ssaFn struct{ fn *ssa.Function }

func fn(t *testing.T, src string) *ssaFn {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	f := ssasupport.Func(pkg, "f")
	if f == nil {
		t.Fatal("f not found")
	}
	return &ssaFn{f}
}

func TestBudgetDefaultPrimarySize(t *testing.T) {
	d, err := annotation.Parse("//bigo:max O(n)")
	if err != nil {
		t.Fatal(err)
	}
	f := fn(t, `package input
func f(xs []int) int { return len(xs) }`)
	b, err := Budget(d, f.fn)
	if err != nil {
		t.Fatal(err)
	}
	if b.String() != "O(len(xs))" {
		t.Errorf("Budget = %q, want O(len(xs))", b.String())
	}
}

func TestBudgetWithBindings(t *testing.T) {
	d, err := annotation.Parse("//bigo:max O(n*m) where n=len(a), m=len(b)")
	if err != nil {
		t.Fatal(err)
	}
	f := fn(t, `package input
func f(a, b []int) int { return len(a) + len(b) }`)
	got, err := Budget(d, f.fn)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "O(len(a) len(b))" {
		t.Errorf("Budget = %q, want O(len(a) len(b))", got.String())
	}
}

func TestBudgetUnboundVarErrors(t *testing.T) {
	d, err := annotation.Parse("//bigo:max O(m)")
	if err != nil {
		t.Fatal(err)
	}
	f := fn(t, `package input
func f(xs []int) int { return len(xs) }`)
	if _, err := Budget(d, f.fn); err == nil {
		t.Errorf("expected error for unbound var m")
	}
}

func TestBudgetSigDefaultsPrimarySize(t *testing.T) {
	d, err := annotation.Parse("//bigo:cost O(n)")
	if err != nil {
		t.Fatal(err)
	}
	f := fn(t, `package input
func f(keys []string) int { return len(keys) }`)
	sig := f.fn.Signature
	b, err := BudgetSig(d, sig)
	if err != nil {
		t.Fatal(err)
	}
	if b.String() != "O(len(keys))" {
		t.Errorf("BudgetSig = %q, want O(len(keys))", b.String())
	}
}
