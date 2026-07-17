package oracle

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// pinsOf parses src as a file and returns ExtractPins' result keyed by func name.
func pinsOf(t *testing.T, src string) (map[string]Pin, error) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pins, err := ExtractPins(file)
	if err != nil {
		return nil, err
	}
	out := map[string]Pin{}
	for decl, p := range pins {
		out[decl.Name.Name] = p
	}
	return out, nil
}

func TestExtractPinsFull(t *testing.T) {
	src := `package p

//oracle:time O(n^2) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source CLRS §2.1
func F(s []int) {}

func Unpinned() {}
`
	pins, err := pinsOf(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 1 {
		t.Fatalf("got %d pins, want 1", len(pins))
	}
	p := pins["F"]
	if p.Source != "CLRS §2.1" {
		t.Errorf("Source = %q", p.Source)
	}
	if p.Time.Budget.String() != "O(n^2)" {
		t.Errorf("Time = %s", p.Time.Budget)
	}
	if p.Space == nil || p.Space.Budget.String() != "O(1)" {
		t.Errorf("Space = %v", p.Space)
	}
	if _, ok := p.Time.Bindings["n"]; !ok {
		t.Error("time where-binding for n missing")
	}
}

func TestExtractPinsSpaceOptional(t *testing.T) {
	src := `package p

//oracle:time O(n) where n=len(s)
//oracle:source somewhere
func F(s []int) {}
`
	pins, err := pinsOf(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if pins["F"].Space != nil {
		t.Error("Space should be nil when unpinned")
	}
}

func TestExtractPinsErrors(t *testing.T) {
	for name, src := range map[string]string{
		"missing source": "package p\n\n//oracle:time O(n) where n=len(s)\nfunc F(s []int) {}\n",
		"missing time":   "package p\n\n//oracle:source x\nfunc F(s []int) {}\n",
		"unknown verb":   "package p\n\n//oracle:tame O(n)\n//oracle:source x\nfunc F(s []int) {}\n",
		"bad expr":       "package p\n\n//oracle:time O(2^n) where n=n\n//oracle:source x\nfunc F(n int) {}\n",
		"duplicate time": "package p\n\n//oracle:time O(n)\n//oracle:time O(n)\n//oracle:source x\nfunc F(s []int) {}\n",
		"empty source":   "package p\n\n//oracle:time O(n) where n=len(s)\n//oracle:source \nfunc F(s []int) {}\n",
	} {
		if _, err := pinsOf(t, src); err == nil {
			t.Errorf("%s: no error", name)
		} else if !strings.Contains(err.Error(), "F") {
			t.Errorf("%s: error %q does not name the function", name, err)
		}
	}
}
