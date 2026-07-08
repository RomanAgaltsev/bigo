package bound

import "testing"

func TestDominates(t *testing.T) {
	tests := []struct {
		name string
		a, b Monomial
		want bool
	}{
		{"n^2 dominates n", Mono("n", 2, 0), Term("n"), true},
		{"n does not dominate n^2", Term("n"), Mono("n", 2, 0), false},
		{"n dominates n log n? no", Term("n"), Term("n").Mul(LogOf("n")), false},
		{"n log n dominates n", Term("n").Mul(LogOf("n")), Term("n"), true},
		{"n log n dominates n^2? no", Term("n").Mul(LogOf("n")), Mono("n", 2, 0), false},
		{"anything dominates one", Term("n"), One(), true},
		{"one does not dominate n", One(), Term("n"), false},
		{"equal monomials dominate each other", Mono("n", 2, 1), Mono("n", 2, 1), true},
		{"n and m are incomparable (n does not dominate m)", Term("n"), Term("m"), false},
		{"n and m are incomparable (m does not dominate n)", Term("m"), Term("n"), false},
		{"n^2 dominates n m? no (m exponent 0 < 1)", Mono("n", 2, 0), Term("n").Mul(Term("m")), false},
	}
	for _, tt := range tests {
		if got := Dominates(tt.a, tt.b); got != tt.want {
			t.Errorf("%s: Dominates(%v,%v) = %v, want %v", tt.name, tt.a, tt.b, got, tt.want)
		}
	}
}
