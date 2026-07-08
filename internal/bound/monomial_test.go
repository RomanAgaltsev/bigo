package bound

import "testing"

func TestMonomialConstructorsAndStrings(t *testing.T) {
	tests := []struct {
		name string
		m    Monomial
		want string
	}{
		{"one", One(), "1"},
		{"n", Term("n"), "n"},
		{"n squared", Mono("n", 2, 0), "n^2"},
		{"log n", LogOf("n"), "log(n)"},
		{"n log n", Term("n").Mul(LogOf("n")), "n log(n)"},
		{"log squared", Mono("n", 0, 2), "log(n)^2"},
		{"n^2 m", Mono("n", 2, 0).Mul(Term("m")), "m n^2"}, // vars sorted: m before n
	}
	for _, tt := range tests {
		if got := tt.m.String(); got != tt.want {
			t.Errorf("%s: String() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestMonomialMulAddsExponents(t *testing.T) {
	got := Mono("n", 1, 1).Mul(Mono("n", 2, 0)) // n·log n · n^2 = n^3 log n
	want := Mono("n", 3, 1)
	if !got.Equal(want) {
		t.Errorf("Mul = %v, want %v", got, want)
	}
}

func TestMonomialEqualIgnoresZeroFactors(t *testing.T) {
	a := Term("n").Mul(Mono("m", 0, 0)) // m^0 contributes nothing
	if !a.Equal(Term("n")) {
		t.Errorf("expected %v == n", a)
	}
}
