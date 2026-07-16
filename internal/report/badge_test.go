package report

import (
	"reflect"
	"testing"
)

// fnBudget / fnSpaceBudget build a Function carrying a single time or space
// budget with the given verdict — the only fields Badge reads.
func fnBudget(v string) Function { return Function{Budget: &BudgetJSON{Verdict: v}} }
func fnSpaceBudget(v string) Function {
	return Function{Space: &SpaceJSON{Budget: &BudgetJSON{Verdict: v}}}
}

func TestBadge(t *testing.T) {
	cases := []struct {
		name string
		fns  []Function
		want Endpoint
	}{
		{
			"zero budgets is lightgrey",
			[]Function{{Func: "Plain"}}, // a function with no budget contributes nothing
			Endpoint{SchemaVersion: 1, Label: "bigo", Message: "no budgets", Color: "lightgrey"},
		},
		{
			"single within is singular noun",
			[]Function{fnBudget("within")},
			Endpoint{SchemaVersion: 1, Label: "bigo", Message: "1 budget · all within", Color: "brightgreen"},
		},
		{
			"all within pools time and space",
			[]Function{fnBudget("within"), fnBudget("within"), fnSpaceBudget("within")},
			Endpoint{SchemaVersion: 1, Label: "bigo", Message: "3 budgets · all within", Color: "brightgreen"},
		},
		{
			"unverifiable only is yellow",
			[]Function{fnBudget("within"), fnBudget("unverifiable")},
			Endpoint{SchemaVersion: 1, Label: "bigo", Message: "2 budgets · 1 unverifiable", Color: "yellow"},
		},
		{
			"invalid outranks unverifiable, orange, severity-ordered",
			[]Function{fnBudget("invalid"), fnBudget("unverifiable")},
			Endpoint{SchemaVersion: 1, Label: "bigo", Message: "2 budgets · 1 invalid, 1 unverifiable", Color: "orange"},
		},
		{
			"exceeds outranks all, red, severity-ordered message",
			[]Function{fnBudget("exceeds"), fnBudget("invalid"), fnBudget("unverifiable"), fnBudget("within")},
			Endpoint{SchemaVersion: 1, Label: "bigo", Message: "4 budgets · 1 exceeds, 1 invalid, 1 unverifiable", Color: "red"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Badge(Document{Functions: tc.fns})
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Badge = %+v, want %+v", got, tc.want)
			}
		})
	}
}
