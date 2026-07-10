package callsummary

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func TestGenericAtInstantiation(t *testing.T) {
	const src = `package input
func sum[T int | float64](xs []T) T {
	var s T
	for i := 0; i < len(xs); i++ { s += xs[i] }
	return s
}
func f(xs []int) int { return sum(xs) }`
	pkg, _, err := ssasupport.BuildGeneric(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if fn == nil {
		t.Fatal("f not found")
	}
	if got, want := engine.Infer(fn, New(nil)).String(), "O(len(xs))"; got != want {
		t.Errorf("Infer = %q, want %q (generic sum should resolve at instantiation)", got, want)
	}
}
