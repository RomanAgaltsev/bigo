package engine

import (
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

// builtinModel: len/cap are O(1); every other call is unverifiable.
type builtinModel struct{}

func (builtinModel) CallCost(c *ssa.CallCommon) bound.Bound {
	if b, ok := c.Value.(*ssa.Builtin); ok {
		switch b.Name() {
		case "len", "cap":
			return bound.Constant()
		}
	}
	return bound.Top()
}

func infer(t *testing.T, src string) string {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if fn == nil {
		t.Fatal("f not found")
	}
	return Infer(fn, builtinModel{}).String()
}

func TestInfer(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{"constant", `package input
func f(x int) int { return x + 1 }`, "O(1)"},
		{"linear scan", `package input
func f(xs []int, t int) int { for i := 0; i < len(xs); i++ { if xs[i] == t { return i } }; return -1 }`, "O(len(xs))"},
		{"nested loops are quadratic", `package input
func f(xs []int) int { s := 0; for i := 0; i < len(xs); i++ { for j := 0; j < len(xs); j++ { s += xs[i]*xs[j] } }; return s }`, "O(len(xs)^2)"},
		{"call is unverifiable under builtin model", `package input
func g(int) int
func f(xs []int) int { s := 0; for i := 0; i < len(xs); i++ { s += g(xs[i]) }; return s }`, "unverifiable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := infer(t, tt.src); got != tt.want {
				t.Errorf("Infer = %q, want %q", got, tt.want)
			}
		})
	}
}

// userLinearModel: len/cap O(1); every other call costs O(k) — a stand-in for
// a resolvable user callee, so defer multiplication is observable.
type userLinearModel struct{}

func (userLinearModel) CallCost(c *ssa.CallCommon) bound.Bound {
	if b, ok := c.Value.(*ssa.Builtin); ok {
		switch b.Name() {
		case "len", "cap":
			return bound.Constant()
		}
	}
	return bound.Of(bound.Term("k"))
}

func inferWith(t *testing.T, model CostModel, src string) string {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if fn == nil {
		t.Fatal("f not found")
	}
	return Infer(fn, model).String()
}

func TestDeferInLoopIsMultiplied(t *testing.T) {
	const src = `package input
func g(int) int
func f(xs []int) {
	for i := 0; i < len(xs); i++ {
		defer g(i)
	}
}`
	// n deferred O(k) calls all run at return: the loop factor must apply.
	if got, want := inferWith(t, userLinearModel{}, src), "O(k len(xs))"; got != want {
		t.Errorf("Infer = %q, want %q", got, want)
	}
}

func TestGoStatementIsUnverifiable(t *testing.T) {
	const src = `package input
func g(int) int
func f(xs []int) {
	for i := 0; i < len(xs); i++ {
		go g(i)
	}
}`
	// concurrency-dependent bounds are unverifiable in v1 — even if
	// the spawned callee itself is resolvable.
	if got := inferWith(t, userLinearModel{}, src); got != "unverifiable" {
		t.Errorf("Infer = %q, want unverifiable", got)
	}
}

func TestBodylessFunctionIsUnverifiable(t *testing.T) {
	const src = `package input
func g(n int) int
func f() int { return 0 }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	g := ssasupport.Func(pkg, "g")
	if g == nil {
		t.Fatal("g not found")
	}
	if got := Infer(g, builtinModel{}); !got.IsTop() {
		t.Errorf("Infer(bodyless) = %q, want Top — no body means nothing is known", got.String())
	}
	if got := Infer(nil, builtinModel{}); !got.IsTop() {
		t.Errorf("Infer(nil) = %q, want Top", got.String())
	}
}

func TestInferDetailedNamesTheBlocker(t *testing.T) {
	const src = `package input
func g(int) int
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		s += g(xs[i])
	}
	return s
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	b, causes := InferDetailed(fn, builtinModel{})
	if !b.IsTop() {
		t.Fatalf("bound = %q, want Top", b.String())
	}
	if len(causes) == 0 {
		t.Fatal("expected at least one cause for a Top bound")
	}
	if want := "unresolved cost at call to input.g"; causes[0].What != want {
		t.Errorf("cause = %q, want %q", causes[0].What, want)
	}
	if !causes[0].Pos.IsValid() {
		t.Error("cause position must be valid")
	}
}

func TestInferDetailedNoCausesWhenBounded(t *testing.T) {
	const src = `package input
func f(xs []int) int { s := 0; for i := 0; i < len(xs); i++ { s += xs[i] }; return s }`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	b, causes := InferDetailed(fn, builtinModel{})
	if b.IsTop() || causes != nil {
		t.Errorf("got (%q, %v), want bounded with nil causes", b.String(), causes)
	}
}

func TestCauseKinds(t *testing.T) {
	const src = `package input
func g(int) int
func f(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		s += g(xs[i])
	}
	return s
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	_, causes := InferDetailed(ssasupport.Func(pkg, "f"), builtinModel{})
	if len(causes) == 0 || causes[0].Kind != CauseCall {
		t.Errorf("causes[0].Kind = %v, want CauseCall", causes)
	}
	_, causes = InferDetailed(ssasupport.Func(pkg, "g"), builtinModel{})
	if len(causes) == 0 || causes[0].Kind != CauseNoBody {
		t.Errorf("bodyless causes[0].Kind = %v, want CauseNoBody", causes)
	}
	if got := CauseCall.String(); got != "call" {
		t.Errorf("CauseCall.String() = %q, want call", got)
	}
}

func TestInferFieldBoundedFunction(t *testing.T) {
	const src = `package input
type S struct{ items []int }
func f(s *S) int {
	t := 0
	for i := 0; i < len(s.items); i++ {
		t += s.items[i]
	}
	return t
}`
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := Infer(ssasupport.Func(pkg, "f"), builtinModel{}).String(), "O(len(s.items))"; got != want {
		t.Errorf("Infer = %q, want %q", got, want)
	}
}
