package callsummary

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

func inferF(t *testing.T, src string) string {
	t.Helper()
	pkg, _, err := ssasupport.Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssasupport.Func(pkg, "f")
	if fn == nil {
		t.Fatal("f not found")
	}
	return engine.Infer(fn, New()).String()
}

func TestInterprocedural(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{
			"call resolves to callee bound",
			`package input
func helper(ys []int) int { s := 0; for i := 0; i < len(ys); i++ { s += ys[i] }; return s }
func f(xs []int) int { return helper(xs) }`,
			"O(len(xs))",
		},
		{
			"linear caller over linear callee is quadratic",
			`package input
func helper(ys []int) int { s := 0; for i := 0; i < len(ys); i++ { s += ys[i] }; return s }
func f(xs []int) int { s := 0; for i := 0; i < len(xs); i++ { s += helper(xs) }; return s }`,
			"O(len(xs)^2)",
		},
		{
			"stdlib sort resolves",
			`package input
import "sort"
func f(xs []int) { sort.Ints(xs) }`,
			"O(len(xs) log(len(xs)))",
		},
		{
			"recursion is unverifiable",
			`package input
func f(xs []int) int { if len(xs) == 0 { return 0 }; return f(xs[1:]) }`,
			"unverifiable",
		},
		{
			"interface call is unverifiable",
			`package input
type D interface{ Do(int) int }
func f(xs []int, d D) int { s := 0; for i := 0; i < len(xs); i++ { s += d.Do(xs[i]) }; return s }`,
			"unverifiable",
		},
		{
			"external call not in the table is unverifiable",
			`package input
import "os"
func f(k string) string { return os.Getenv(k) }`,
			"unverifiable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inferF(t, tt.src); got != tt.want {
				t.Errorf("Infer = %q, want %q", got, tt.want)
			}
		})
	}
}
