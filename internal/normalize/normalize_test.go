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

func TestBudgetFieldPathBinding(t *testing.T) {
	d, err := annotation.Parse("//bigo:max O(n) where n=len(s.items)")
	if err != nil {
		t.Fatal(err)
	}
	f := fn(t, `package input
type S struct{ items []int }
func f(s *S) int { return 0 }`)
	b, err := Budget(d, f.fn)
	if err != nil {
		t.Fatal(err)
	}
	if b.String() != "O(len(s.items))" {
		t.Errorf("Budget = %q, want O(len(s.items))", b.String())
	}
}

// A bound printed by -report must be pastable back as a budget, so a canonical
// size variable is accepted with no 'where' clause.
func TestBudgetAcceptsCanonicalSizeVar(t *testing.T) {
	src := `package input
func f(xs []int, s string) int { return len(xs) + len(s) }`
	for _, tt := range []struct{ dir, want string }{
		{"//bigo:max O(len(xs))", "O(len(xs))"},
		{"//bigo:max O(cap(xs))", "O(cap(xs))"},
		{"//bigo:max O(len(s))", "O(len(s))"},
		{"//bigo:max O(log(len(xs)))", "O(log(len(xs)))"},
		{"//bigo:max O(len(xs) log(len(xs)))", "O(len(xs) log(len(xs)))"},
	} {
		d, err := annotation.Parse(tt.dir)
		if err != nil {
			t.Errorf("Parse(%q): %v", tt.dir, err)
			continue
		}
		got, err := Budget(d, fn(t, src).fn)
		if err != nil {
			t.Errorf("Budget(%q): %v", tt.dir, err)
			continue
		}
		if got.String() != tt.want {
			t.Errorf("Budget(%q) = %q, want %q", tt.dir, got.String(), tt.want)
		}
	}
}

// The accept path is deliberately narrow: a size variable naming something
// that is not a length-bearing parameter would let a budget mention a variable
// no inferred bound can contain, making the comparison meaningless.
func TestBudgetRejectsNonCanonicalSizeVar(t *testing.T) {
	src := `package input
func f(xs []int, k int) int { return len(xs) + k }`
	for _, dir := range []string{
		"//bigo:max O(len(nope))", // not a parameter at all
		"//bigo:max O(len(k))",    // an int has no length
		"//bigo:max O(cap(k))",
	} {
		d, err := annotation.Parse(dir)
		if err != nil {
			continue // rejected earlier, at parse time, which is also fine
		}
		if _, err := Budget(d, fn(t, src).fn); err == nil {
			t.Errorf("Budget(%q): expected an error, got nil", dir)
		}
	}
}
