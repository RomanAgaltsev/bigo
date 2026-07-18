package engine

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/ssasupport"
)

type topModel struct{}

func (topModel) CallCost(*ssa.CallCommon) bound.Bound { return bound.Top() }

func TestCauseTextPricedVsUnpriced(t *testing.T) {
	pkg, _, err := ssasupport.Build(`package input
func g() {}
func f(d, s []int) { copy(d, s); g() }`)
	if err != nil {
		t.Fatal(err)
	}
	_, causes := InferDetailed(ssasupport.Func(pkg, "f"), topModel{})
	var texts []string
	for _, c := range causes {
		if c.Kind == CauseCall {
			texts = append(texts, c.What)
		}
	}
	joined := strings.Join(texts, "; ")
	if !strings.Contains(joined, "unresolved argument size at call to copy") {
		t.Errorf("priced callee cause wrong: %q", joined)
	}
	if !strings.Contains(joined, "unresolved cost at call to") ||
		strings.Contains(joined, "unresolved cost at call to copy") {
		t.Errorf("un-priced callee cause wrong: %q", joined)
	}
}
