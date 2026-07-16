package report

import (
	"strings"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// doc builds a minimal compatible document carrying fns.
func doc(fns ...Function) Document {
	return Document{
		SchemaVersion: SchemaVersion,
		BigoVersion:   "1.19.0",
		Module:        "example.com/m",
		Functions:     fns,
	}
}

func TestCompatSameModuleAndVersionIsClean(t *testing.T) {
	warn, err := Compat(doc(), doc())
	if err != nil {
		t.Fatalf("Compat = %v, want nil", err)
	}
	if warn != "" {
		t.Errorf("warning = %q, want empty", warn)
	}
}

func TestCompatDifferentModuleIsError(t *testing.T) {
	head := doc()
	head.Module = "example.com/other"
	if _, err := Compat(doc(), head); err == nil {
		t.Fatal("Compat across modules = nil, want error (apples-to-oranges)")
	}
}

func TestCompatMajorSchemaMismatchIsError(t *testing.T) {
	head := doc()
	head.SchemaVersion = "2.0.0"
	if _, err := Compat(doc(), head); err == nil {
		t.Fatal("Compat across schema majors = nil, want error")
	}
}

func TestCompatMinorSchemaMismatchIsAllowed(t *testing.T) {
	head := doc()
	// A minor far ahead of any real release: within a major the format is
	// additive-only, so this must be allowed. Deliberately not "1.1.0" — the
	// contribution-program plan bumps SchemaVersion to 1.1.0, which would make
	// this case compare a version against itself and silently stop testing a
	// mismatch while still passing.
	head.SchemaVersion = "1.99.0"
	if _, err := Compat(doc(), head); err != nil {
		t.Errorf("Compat across schema minors = %v, want nil", err)
	}
}

func TestCompatBigoVersionMismatchWarns(t *testing.T) {
	head := doc()
	head.BigoVersion = "1.20.0"
	warn, err := Compat(doc(), head)
	if err != nil {
		t.Fatalf("Compat = %v, want nil (a version bump is not fatal)", err)
	}
	if !strings.Contains(warn, "1.19.0") || !strings.Contains(warn, "1.20.0") {
		t.Errorf("warning = %q, want both versions named", warn)
	}
}

func TestBoundOfTop(t *testing.T) {
	b, ok := boundOf(BoundJSON{Top: true})
	if !ok || !b.IsTop() {
		t.Errorf("boundOf(top) = %v/%v, want Top/true", b, ok)
	}
}

func TestBoundOfConstant(t *testing.T) {
	// O(1) serializes as one empty term (document_test.go:19).
	b, ok := boundOf(BoundJSON{Str: "O(1)", Terms: []map[string]FactorJSON{{}}})
	if !ok {
		t.Fatal("boundOf(O(1)) not ok")
	}
	if !b.Equal(bound.Constant()) {
		t.Errorf("boundOf(O(1)) = %v, want O(1)", b)
	}
}

func TestBoundOfPolyLogRoundTrips(t *testing.T) {
	want := bound.Of(bound.Mono("n", 1, 1)) // n log n
	b, ok := boundOf(boundJSON(want))
	if !ok {
		t.Fatal("boundOf not ok")
	}
	if !b.Equal(want) {
		t.Errorf("round trip = %v, want %v", b, want)
	}
}

func TestBoundOfAntichainRoundTrips(t *testing.T) {
	want := bound.Of(bound.Term("n"), bound.Term("m"))
	b, ok := boundOf(boundJSON(want))
	if !ok || !b.Equal(want) {
		t.Errorf("round trip = %v/%v, want %v", b, ok, want)
	}
}

func TestBoundOfAbsentIsNotOk(t *testing.T) {
	// A zero BoundJSON means "no bound recorded" — never O(1).
	if _, ok := boundOf(BoundJSON{}); ok {
		t.Error("boundOf(zero) ok = true, want false: absent must not read as O(1)")
	}
}
