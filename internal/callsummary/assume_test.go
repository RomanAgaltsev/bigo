package callsummary

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/assume"
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/normalize"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// inferWith is inferF plus an assumption set on the resolver.
func inferWith(t *testing.T, src, assumptions string) (string, *Resolver) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if fn == nil {
		t.Fatal("f not found")
	}
	r := New(nil)
	if assumptions != "" {
		es, err := assume.ParseText(assumptions)
		if err != nil {
			t.Fatal(err)
		}
		r.UseAssumptions(assume.NewSet(es))
	}
	return engine.Infer(fn, r).String(), r
}

func mustBound(t *testing.T, expr string, fn *ssa.Function) bound.Bound {
	t.Helper()
	dir, err := annotation.Parse("//bigo:cost " + expr)
	if err != nil {
		t.Fatal(err)
	}
	b, err := normalize.Budget(dir, fn)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAssumptionBoundsUncuratedStdlib(t *testing.T) {
	src := `package input
import "os"
func f(k string) string { return os.Getenv(k) }`
	got, _ := inferWith(t, src, "") // control: unpriced stdlib is unverifiable
	if got != "unverifiable" {
		t.Fatalf("control = %q, want unverifiable", got)
	}
	got, _ = inferWith(t, src, "os.Getenv O(1)\n")
	if got != "O(1)" {
		t.Fatalf("assumed = %q, want O(1)", got)
	}
}

func TestAssumptionBeatsInference(t *testing.T) {
	src := `package input
import "os"
func blocked(xs []int) int { os.Getenv("x"); return len(xs) }
func f(xs []int) int { return blocked(xs) }`
	got, _ := inferWith(t, src, "input.blocked O(n) where n=len(xs)\n")
	if got != "O(len(xs))" {
		t.Fatalf("got %q, want the assumed parametric bound in caller vars", got)
	}
}

func TestDirectiveBeatsAssumptionAndWarns(t *testing.T) {
	src := `package input
func helper(ys []int) int { s := 0; for i := 0; i < len(ys); i++ { s += ys[i] }; return s }
func f(xs []int) int { return helper(xs) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	helper := ssasupport.Func(pkg, "helper")
	es, err := assume.ParseText("input.helper O(1)\n")
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a //bigo:cost directive: an override on helper.
	r := New(map[*ssa.Function]bound.Bound{helper: mustBound(t, "O(n) where n=len(ys)", helper)})
	r.UseAssumptions(assume.NewSet(es))
	if got := engine.Infer(fn, r).String(); got != "O(len(xs))" {
		t.Fatalf("got %q — the directive must win over the O(1) assumption", got)
	}
	warns := r.AssumeWarnings()
	if len(warns) != 1 || !strings.Contains(warns[0], "input.helper") || !strings.Contains(warns[0], "directive") {
		t.Fatalf("warnings = %v, want one directive-shadow warning naming input.helper", warns)
	}
}

func TestCuratedTableBeatsAssumptionAndWarns(t *testing.T) {
	src := `package input
import "sort"
func f(xs []int) { sort.Ints(xs) }`
	got, r := inferWith(t, src, "sort.Ints O(1)\n")
	if got != "O(len(xs) log(len(xs)))" {
		t.Fatalf("got %q — the curated entry must win", got)
	}
	warns := r.AssumeWarnings()
	if len(warns) != 1 || !strings.Contains(warns[0], "sort.Ints") {
		t.Fatalf("warnings = %v, want one curated-shadow warning", warns)
	}
}
