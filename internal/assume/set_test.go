package assume_test

import (
	"strings"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/assume"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func setOf(t *testing.T, text string) *assume.Set {
	t.Helper()
	es, err := assume.ParseText(text)
	if err != nil {
		t.Fatal(err)
	}
	return assume.NewSet(es)
}

func TestValidateMatchesAndCompiles(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
func Work(xs []int) int { return len(xs) }
func f() {}`)
	if err != nil {
		t.Fatal(err)
	}
	s := setOf(t, "input.Work O(n) where n=len(xs)\n")
	if err := s.Validate(pkg.Prog); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	fn := ssasupport.Func(pkg, "Work")
	b, names, ok := s.For("input.Work", fn.Signature)
	if !ok || b.String() != "O(len(xs))" {
		t.Fatalf("For = %v %v %v", b, names, ok)
	}
	if len(names) != 1 || names[0] != "xs" {
		t.Fatalf("names = %v", names)
	}
}

func TestValidateUnmatchedKeyIsHardError(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
func f() {}`)
	if err != nil {
		t.Fatal(err)
	}
	s := setOf(t, "input.NoSuch O(1)\n")
	err = s.Validate(pkg.Prog)
	if err == nil || !strings.Contains(err.Error(), "input.NoSuch") {
		t.Fatalf("err = %v, want unmatched-key error naming input.NoSuch", err)
	}
}

func TestValidateBadVarIsHardError(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
func Work(xs []int) int { return len(xs) }`)
	if err != nil {
		t.Fatal(err)
	}
	s := setOf(t, "input.Work O(k) where k=nosuchparam\n")
	if err := s.Validate(pkg.Prog); err == nil {
		t.Fatal("want compile error for a var naming no parameter")
	}
}
