package fieldpath

import (
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// lenOnField returns the first len(...) call in f whose argument is not a
// bare parameter — i.e. the field-read len the test src contains exactly one of.
func lenOnField(t *testing.T, src string) (ssa.Value, *Stability) {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if fn == nil {
		t.Fatal("f not found")
	}
	stab := Analyze(fn)
	for _, b := range fn.Blocks {
		for _, in := range b.Instrs {
			c, ok := in.(*ssa.Call)
			if !ok {
				continue
			}
			bi, ok := c.Call.Value.(*ssa.Builtin)
			if !ok || bi.Name() != "len" {
				continue
			}
			if _, isParam := c.Call.Args[0].(*ssa.Parameter); isParam {
				continue
			}
			return c, stab
		}
	}
	t.Fatal("no len-on-field call found")
	return nil, nil
}

func TestVarForAccepts(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"rule V: value param, calls everywhere are harmless",
			`package input
type S struct{ items []int }
func cb()
func f(s S) int { cb(); t := 0; for i := 0; i < len(s.items); i++ { cb(); t++ }; return t }`,
			"len(s.items)",
		},
		{
			"rule P: hoisted length, call-free preamble",
			`package input
type S struct{ items []int }
func f(s *S) int { n := len(s.items); t := 0; for i := 0; i < n; i++ { t++ }; return t }`,
			"len(s.items)",
		},
		{
			"rule P: inline re-read, call-free store-free body",
			`package input
type S struct{ items []int }
func f(s *S) int { t := 0; for i := 0; i < len(s.items); i++ { t += s.items[i] }; return t }`,
			"len(s.items)",
		},
		{
			"depth 2 through an embedded struct field",
			`package input
type C struct{ items []int }
type S struct{ cfg C }
func f(s *S) int { t := 0; for i := 0; i < len(s.cfg.items); i++ { t++ }; return t }`,
			"len(s.cfg.items)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, stab := lenOnField(t, tt.src)
			got, ok := stab.VarFor(v)
			if !ok || string(got) != tt.want {
				t.Errorf("VarFor = (%q, %v), want (%q, true)", got, ok, tt.want)
			}
		})
	}
}

func TestVarForRejects(t *testing.T) {
	tests := []struct{ name, src string }{
		{
			"rule P: call before the load",
			`package input
type S struct{ items []int }
func cb()
func f(s *S) int { cb(); n := len(s.items); t := 0; for i := 0; i < n; i++ { t++ }; return t }`,
		},
		{
			"rule P: call inside the loop body reaches the header re-read",
			`package input
type S struct{ items []int }
func cb()
func f(s *S) int { t := 0; for i := 0; i < len(s.items); i++ { cb(); t++ }; return t }`,
		},
		{
			"store to the path before the loop",
			`package input
type S struct{ items []int }
func f(s *S, v int) int { s.items = append(s.items, v); t := 0; for i := 0; i < len(s.items); i++ { t++ }; return t }`,
		},
		{
			"escaped &s.items forces a spill; alloc-rooted loads are rejected",
			`package input
type S struct{ items []int }
func mutate(p *[]int)
func f(s S) int { mutate(&s.items); t := 0; for i := 0; i < len(s.items); i++ { t++ }; return t }`,
		},
		{
			"depth 3 rejected",
			`package input
type A struct{ items []int }
type B struct{ a A }
type S struct{ b B }
func f(s *S) int { t := 0; for i := 0; i < len(s.b.a.items); i++ { t++ }; return t }`,
		},
		{
			"channel field rejected (sync legalizes concurrent mutation)",
			`package input
type S struct{ ch chan int }
func f(s *S) int { t := 0; for i := 0; i < len(s.ch); i++ { t++ }; return t }`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, stab := lenOnField(t, tt.src)
			if got, ok := stab.VarFor(v); ok {
				t.Errorf("VarFor = (%q, true), want rejection — accepting here is a wrong-bound bug", got)
			}
		})
	}
}

func TestVarForNilSafe(t *testing.T) {
	var s *Stability
	if _, ok := s.VarFor(nil); ok {
		t.Error("nil Stability must reject everything")
	}
}
