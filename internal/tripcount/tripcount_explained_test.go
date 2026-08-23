package tripcount

import (
	"testing"
)

// Every rule that can fire must be able to say its own name. A rule with a
// name but no fixture here is untested presentation.
func TestOfExplainedNamesTheRule(t *testing.T) {
	tests := []struct {
		name string
		src  string
		rule string
	}{
		{
			name: "increasing",
			src: `package input
func f(xs []int) int {
	n := 0
	for i := 0; i < len(xs); i++ {
		n += xs[i]
	}
	return n
}`,
			rule: "increasing (unit step)",
		},
		{
			name: "decreasing",
			src: `package input
func f(k int) int {
	n := 0
	for i := k; i > 0; i-- {
		n++
	}
	return n
}`,
			rule: "decreasing",
		},
		{
			name: "geometric-down",
			src: `package input
func f(k int) int {
	n := 0
	for i := k; i > 0; i /= 2 {
		n++
	}
	return n
}`,
			rule: "geometric-down (halving)",
		},
		{
			name: "geometric-up",
			src: `package input
func f(k int) int {
	n := 0
	for i := 1; i < k; i *= 2 {
		n++
	}
	return n
}`,
			rule: "geometric-up (doubling)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loop, stab := firstLoop(t, tt.src)
			b, rule := OfExplained(loop, stab)
			if b.IsTop() {
				t.Fatalf("bound is top; the fixture must be bounded for the rule name to mean anything")
			}
			if rule != tt.rule {
				t.Errorf("rule = %q, want %q", rule, tt.rule)
			}
		})
	}
}

// The most important negative in the feature: a loop no rule bounded must not
// be handed a plausible-looking rule name (spec 2.3).
func TestOfExplainedNamesNoRuleWhenNoneMatched(t *testing.T) {
	loop, stab := firstLoop(t, `package input
func f(xs []int) int {
	n := 0
	for len(xs) > 0 {
		xs = xs[n:]
		n++
	}
	return n
}`)
	b, rule := OfExplained(loop, stab)
	if !b.IsTop() {
		t.Skip("fixture became bounded; replace it with one that is not")
	}
	if rule != "" {
		t.Errorf("rule = %q, want empty when no rule matched", rule)
	}
}

// Of and OfExplained must never disagree: OfExplained IS the derivation and Of
// delegates to it, so a divergence means a second implementation appeared.
func TestOfDelegatesToOfExplained(t *testing.T) {
	for _, src := range []string{
		`package input
func f(xs []int) int { n := 0; for i := 0; i < len(xs); i++ { n++ }; return n }`,
		`package input
func f(xs []int) int { n := 0; for len(xs) > 0 { xs = xs[n:]; n++ }; return n }`,
	} {
		loop, stab := firstLoop(t, src)
		want := Of(loop, stab)
		got, _ := OfExplained(loop, stab)
		if want.String() != got.String() {
			t.Errorf("Of = %s, OfExplained = %s", want.String(), got.String())
		}
	}
}
