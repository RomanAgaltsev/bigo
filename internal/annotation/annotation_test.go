package annotation

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

func TestParseDirectives(t *testing.T) {
	t.Run("max with bound", func(t *testing.T) {
		d, err := Parse("//bigo:max O(n log n)")
		if err != nil {
			t.Fatal(err)
		}
		if d.Verb != Max {
			t.Errorf("Verb = %v, want Max", d.Verb)
		}
		if got, want := d.Budget.String(), "O(n log(n))"; got != want {
			t.Errorf("Budget = %q, want %q", got, want)
		}
		if len(d.Bindings) != 0 {
			t.Errorf("expected no bindings, got %v", d.Bindings)
		}
	})

	t.Run("cost verb", func(t *testing.T) {
		d, err := Parse("//bigo:cost O(log n)")
		if err != nil {
			t.Fatal(err)
		}
		if d.Verb != Cost {
			t.Errorf("Verb = %v, want Cost", d.Verb)
		}
	})

	t.Run("ignore takes no argument", func(t *testing.T) {
		d, err := Parse("//bigo:ignore")
		if err != nil {
			t.Fatal(err)
		}
		if d.Verb != Ignore {
			t.Errorf("Verb = %v, want Ignore", d.Verb)
		}
		if _, err := Parse("//bigo:ignore O(n)"); err == nil {
			t.Errorf("expected error for ignore with argument")
		}
	})

	t.Run("leading spaces tolerated", func(t *testing.T) {
		if _, err := Parse("//  bigo:max O(n)"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("non-bigo comment rejected", func(t *testing.T) {
		if _, err := Parse("// just a comment"); err == nil {
			t.Errorf("expected error for non-bigo comment")
		}
	})

	t.Run("unknown verb rejected", func(t *testing.T) {
		if _, err := Parse("//bigo:frobnicate O(n)"); err == nil {
			t.Errorf("expected error for unknown verb")
		}
	})

	var _ = bound.Var("n")
}
