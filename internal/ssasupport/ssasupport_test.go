package ssasupport

import "testing"

func TestBuildAndFunc(t *testing.T) {
	const src = `package input
func Sum(xs []int) int {
	total := 0
	for i := 0; i < len(xs); i++ {
		total += xs[i]
	}
	return total
}`
	pkg, _, err := Build(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := Func(pkg, "Sum")
	if fn == nil {
		t.Fatal("Sum function not found")
	}
	if len(fn.Blocks) == 0 {
		t.Fatal("Sum has no SSA blocks")
	}
}
