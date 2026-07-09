package bound

import "testing"

func TestSubst(t *testing.T) {
	b := Of(Term("n").Mul(LogOf("n"))) // O(n log n)
	got := b.Subst(map[Var]Var{"n": "len(xs)"})
	if want := "O(len(xs) log(len(xs)))"; got.String() != want {
		t.Errorf("Subst = %q, want %q", got.String(), want)
	}
}

func TestSubstTopStaysTop(t *testing.T) {
	if !Top().Subst(map[Var]Var{"n": "m"}).IsTop() {
		t.Errorf("Top().Subst should stay Top")
	}
}

func TestSubstMissingKeyUnchanged(t *testing.T) {
	b := Of(Term("m"))
	if got := b.Subst(map[Var]Var{"n": "x"}); got.String() != "O(m)" {
		t.Errorf("Subst = %q, want O(m)", got.String())
	}
}
