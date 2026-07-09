package engine

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

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
	return Infer(fn).String()
}

func TestInfer(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			"constant",
			`package input
func f(x int) int { return x + 1 }`,
			"O(1)",
		},
		{
			"linear scan",
			`package input
func f(xs []int, t int) int { for i := 0; i < len(xs); i++ { if xs[i] == t { return i } }; return -1 }`,
			"O(len(xs))",
		},
		{
			"nested loops are quadratic",
			`package input
func f(xs []int) int { s := 0; for i := 0; i < len(xs); i++ { for j := 0; j < len(xs); j++ { s += xs[i]*xs[j] } }; return s }`,
			"O(len(xs)^2)",
		},
		{
			"call makes it unverifiable",
			`package input
func g(int) int
func f(xs []int) int { s := 0; for i := 0; i < len(xs); i++ { s += g(xs[i]) }; return s }`,
			"unverifiable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := infer(t, tt.src); got != tt.want {
				t.Errorf("Infer = %q, want %q", got, tt.want)
			}
		})
	}
}
