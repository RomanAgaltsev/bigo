package bound

import "testing"

func TestMulLoopMultipliesBounds(t *testing.T) {
	// A loop of O(n) iterations with an O(n) body is O(n^2).
	b := Of(Term("n")).Mul(Of(Term("n")))
	if got, want := b.String(), "O(n^2)"; got != want {
		t.Errorf("Mul = %q, want %q", got, want)
	}
}

func TestMulDistributesOverSum(t *testing.T) {
	// O(n) × O(m + 1) = O(n·m + n) -> reduces to O(n·m) (n dominated by n·m).
	b := Of(Term("n")).Mul(Of(Term("m"), One()))
	if got, want := b.String(), "O(m n)"; got != want {
		t.Errorf("Mul = %q, want %q", got, want)
	}
}

func TestMulByConstantIsIdentity(t *testing.T) {
	b := Of(Mono("n", 2, 0)).Mul(Constant())
	if got, want := b.String(), "O(n^2)"; got != want {
		t.Errorf("Mul by O(1) = %q, want %q", got, want)
	}
}

func TestMulTopIsAbsorbing(t *testing.T) {
	if !Of(Term("n")).Mul(Top()).IsTop() {
		t.Errorf("x.Mul(Top()) should be Top")
	}
	if !Top().Mul(Of(Term("n"))).IsTop() {
		t.Errorf("Top().Mul(x) should be Top")
	}
}
