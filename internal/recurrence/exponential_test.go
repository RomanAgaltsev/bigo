package recurrence

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func expCheck(t *testing.T, src, name string) (int, bool) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	return ProvablyExponential(ssasupport.Func(pkg, name))
}

func TestProvablyExponential(t *testing.T) {
	// Naive Fibonacci: a=2 subtractive, guarded — the showcase.
	fib := `package input
func Fib(n int) int {
	if n < 2 { return n }
	return Fib(n-1) + Fib(n-2)
}`
	if a, ok := expCheck(t, fib, "Fib"); !ok || a != 2 {
		t.Errorf("Fib: ProvablyExponential = (%d, %v), want (2, true)", a, ok)
	}

	// Guarded linear countdown: a=1 subtractive — not exponential.
	countdown := `package input
func Countdown(n int) int {
	if n <= 0 { return 0 }
	return 1 + Countdown(n-1)
}`
	if _, ok := expCheck(t, countdown, "Countdown"); ok {
		t.Error("Countdown: a=1 must not be exponential")
	}

	// Divisive binary-search shape: solves, not exponential.
	binary := `package input
func Bin(n int) int {
	if n > 0 { return Bin(n / 2) }
	return 0
}`
	if _, ok := expCheck(t, binary, "Bin"); ok {
		t.Error("Bin: divisive must not be exponential")
	}

	// Unguarded f(n-1)+f(n-2): no termination proof — must NOT fire.
	unguarded := `package input
func U(n int) int {
	return U(n-1) + U(n-2)
}`
	if _, ok := expCheck(t, unguarded, "U"); ok {
		t.Error("U: unguarded recursion must not be exponential (no termination proof)")
	}
}

func TestProvablyExponentialMutualPair(t *testing.T) {
	// Mutual pair: not directly self-recursive — stays out.
	src := `package input
func A(n int) int { if n <= 0 { return 0 }; return B(n-1) + B(n-2) }
func B(n int) int { if n <= 0 { return 0 }; return A(n-1) + A(n-2) }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ProvablyExponential(ssasupport.Func(pkg, "A")); ok {
		t.Error("A: mutual recursion must not be exponential (not directly self-recursive)")
	}
}
