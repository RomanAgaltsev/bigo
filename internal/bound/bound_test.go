package bound

import "testing"

func TestOfReducesToAntichain(t *testing.T) {
	// O(n + n^2 + 1) collapses to O(n^2): n and 1 are dominated by n^2.
	b := Of(Term("n"), Mono("n", 2, 0), One())
	if got, want := b.String(), "O(n^2)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if len(b.Terms()) != 1 {
		t.Errorf("expected 1 term, got %d", len(b.Terms()))
	}
}

func TestOfKeepsIncomparableTerms(t *testing.T) {
	// O(n·m + n^2): neither dominates the other, both kept, sorted in String.
	b := Of(Term("n").Mul(Term("m")), Mono("n", 2, 0))
	if got, want := b.String(), "O(m n + n^2)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestConstantAndOfEmpty(t *testing.T) {
	if got, want := Constant().String(), "O(1)"; got != want {
		t.Errorf("Constant() = %q, want %q", got, want)
	}
	if !Of().Equal(Constant()) {
		t.Errorf("Of() should equal Constant()")
	}
}

func TestJoinTakesDominant(t *testing.T) {
	b := Of(Term("n")).Join(Of(Mono("n", 2, 0)))
	if got, want := b.String(), "O(n^2)"; got != want {
		t.Errorf("Join = %q, want %q", got, want)
	}
}

func TestTopIsAbsorbingUnderJoin(t *testing.T) {
	if !Top().Join(Of(Term("n"))).IsTop() {
		t.Errorf("Top().Join(x) should be Top")
	}
	if !Of(Term("n")).Join(Top()).IsTop() {
		t.Errorf("x.Join(Top()) should be Top")
	}
	if got, want := Top().String(), "unverifiable"; got != want {
		t.Errorf("Top().String() = %q, want %q", got, want)
	}
}
