package bound

import "testing"

func TestVerdictString(t *testing.T) {
	for v, want := range map[Verdict]string{Within: "within", Exceeds: "exceeds", Unknown: "unknown"} {
		if got := v.String(); got != want {
			t.Errorf("Verdict(%d).String() = %q, want %q", v, got, want)
		}
	}
}

func TestCheck(t *testing.T) {
	nlogn := Of(Term("n").Mul(LogOf("n")))
	tests := []struct {
		name             string
		inferred, budget Bound
		want             Verdict
	}{
		{"n within n log n", Of(Term("n")), nlogn, Within},
		{"n log n within n log n", nlogn, nlogn, Within},
		{"n^2 exceeds n log n", Of(Mono("n", 2, 0)), nlogn, Exceeds},
		{"n^2 exceeds n", Of(Mono("n", 2, 0)), Of(Term("n")), Exceeds},
		{"constant within anything", Constant(), Of(Term("n")), Within},
		{"unverifiable inferred is unknown", Top(), Of(Term("n")), Unknown},
		{"unverifiable budget is unknown", Of(Term("n")), Top(), Unknown},
		// Multi-variable, sound-conservative: n·m vs budget n^2+m^2 is truly within
		// (AM-GM) but not dominated by a single term -> conservatively Unknown.
		{"n*m vs n^2+m^2 is conservatively unknown", Of(Term("n").Mul(Term("m"))), Of(Mono("n", 2, 0), Mono("m", 2, 0)), Unknown},
		// n^3 strictly dominates every term of n^2+m^2? no (incomparable to m^2) -> Unknown, not Exceeds.
		{"n^3 vs n^2+m^2 is unknown not exceeds", Of(Mono("n", 3, 0)), Of(Mono("n", 2, 0), Mono("m", 2, 0)), Unknown},
		// A sum where one term is a clear violation: O(n + n^3) vs O(n^2) -> Exceeds.
		{"sum with a violating term exceeds", Of(Term("n"), Mono("n", 3, 0)), Of(Mono("n", 2, 0)), Exceeds},
	}
	for _, tt := range tests {
		if got := Check(tt.inferred, tt.budget); got != tt.want {
			t.Errorf("%s: Check(%v,%v) = %v, want %v", tt.name, tt.inferred, tt.budget, got, tt.want)
		}
	}
}
