package report

import (
	"strings"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// doc builds a minimal compatible document carrying fns.
func doc(fns ...Function) Document {
	return Document{
		SchemaVersion: SchemaVersion,
		BigoVersion:   "1.19.0",
		Module:        "example.com/m",
		Functions:     fns,
	}
}

func TestCompatSameModuleAndVersionIsClean(t *testing.T) {
	warn, err := Compat(doc(), doc())
	if err != nil {
		t.Fatalf("Compat = %v, want nil", err)
	}
	if warn != "" {
		t.Errorf("warning = %q, want empty", warn)
	}
}

func TestCompatDifferentModuleIsError(t *testing.T) {
	head := doc()
	head.Module = "example.com/other"
	if _, err := Compat(doc(), head); err == nil {
		t.Fatal("Compat across modules = nil, want error (apples-to-oranges)")
	}
}

func TestCompatMajorSchemaMismatchIsError(t *testing.T) {
	head := doc()
	head.SchemaVersion = "2.0.0"
	if _, err := Compat(doc(), head); err == nil {
		t.Fatal("Compat across schema majors = nil, want error")
	}
}

func TestCompatMinorSchemaMismatchIsAllowed(t *testing.T) {
	head := doc()
	// A minor far ahead of any real release: within a major the format is
	// additive-only, so this must be allowed. Deliberately not "1.1.0" — the
	// contribution-program plan bumps SchemaVersion to 1.1.0, which would make
	// this case compare a version against itself and silently stop testing a
	// mismatch while still passing.
	head.SchemaVersion = "1.99.0"
	if _, err := Compat(doc(), head); err != nil {
		t.Errorf("Compat across schema minors = %v, want nil", err)
	}
}

func TestCompatBigoVersionMismatchWarns(t *testing.T) {
	head := doc()
	head.BigoVersion = "1.20.0"
	warn, err := Compat(doc(), head)
	if err != nil {
		t.Fatalf("Compat = %v, want nil (a version bump is not fatal)", err)
	}
	if !strings.Contains(warn, "1.19.0") || !strings.Contains(warn, "1.20.0") {
		t.Errorf("warning = %q, want both versions named", warn)
	}
}

func TestBoundOfTop(t *testing.T) {
	b, ok := boundOf(BoundJSON{Top: true})
	if !ok || !b.IsTop() {
		t.Errorf("boundOf(top) = %v/%v, want Top/true", b, ok)
	}
}

func TestBoundOfConstant(t *testing.T) {
	// O(1) serializes as one empty term (document_test.go:19).
	b, ok := boundOf(BoundJSON{Str: "O(1)", Terms: []map[string]FactorJSON{{}}})
	if !ok {
		t.Fatal("boundOf(O(1)) not ok")
	}
	if !b.Equal(bound.Constant()) {
		t.Errorf("boundOf(O(1)) = %v, want O(1)", b)
	}
}

func TestBoundOfPolyLogRoundTrips(t *testing.T) {
	want := bound.Of(bound.Mono("n", 1, 1)) // n log n
	b, ok := boundOf(boundJSON(want))
	if !ok {
		t.Fatal("boundOf not ok")
	}
	if !b.Equal(want) {
		t.Errorf("round trip = %v, want %v", b, want)
	}
}

func TestBoundOfAntichainRoundTrips(t *testing.T) {
	want := bound.Of(bound.Term("n"), bound.Term("m"))
	b, ok := boundOf(boundJSON(want))
	if !ok || !b.Equal(want) {
		t.Errorf("round trip = %v/%v, want %v", b, ok, want)
	}
}

func TestBoundOfAbsentIsNotOk(t *testing.T) {
	// A zero BoundJSON means "no bound recorded" — never O(1).
	if _, ok := boundOf(BoundJSON{}); ok {
		t.Error("boundOf(zero) ok = true, want false: absent must not read as O(1)")
	}
}

// fn builds a Function with an inferred time bound and optional budget.
func fn(name string, time BoundJSON, budget *BudgetJSON) Function {
	return Function{Package: "example.com/m/p", Func: name, File: "p/x.go", Line: 10, Time: time, Budget: budget}
}

func bj(b bound.Bound) BoundJSON { return boundJSON(b) }

var (
	oN     = bound.Of(bound.Term("n"))
	oN2    = bound.Of(bound.Mono("n", 2, 0))
	oNLogN = bound.Of(bound.Mono("n", 1, 1))
	oTop   = bound.Top()
)

func budget(b bound.Bound, verdict string) *BudgetJSON {
	x := boundJSON(b)
	return &BudgetJSON{Raw: "//bigo:max " + b.String(), Bound: &x, Verdict: verdict}
}

// classesOf extracts the class sequence for terse assertions.
func classesOf(fs []Finding) []Class {
	out := make([]Class, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.Class)
	}
	return out
}

func TestDiffBudgetBreak(t *testing.T) {
	base := doc(fn("SortUsers", bj(oN), budget(oN, "within")))
	head := doc(fn("SortUsers", bj(oN2), budget(oN, "exceeds")))
	fs, _, err := Diff(base, head)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].Class != BudgetBreak {
		t.Fatalf("findings = %+v, want one BudgetBreak", fs)
	}
	if !strings.Contains(fs[0].Message, "SortUsers") || !strings.Contains(fs[0].Message, "O(n^2)") {
		t.Errorf("message = %q, want the function and its inferred bound named", fs[0].Message)
	}
}

func TestDiffProvenRegressionWithoutBudget(t *testing.T) {
	base := doc(fn("Scan", bj(oN), nil))
	head := doc(fn("Scan", bj(oN2), nil))
	fs, _, err := Diff(base, head)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].Class != ProvenRegression {
		t.Fatalf("findings = %+v, want one ProvenRegression", fs)
	}
}

func TestDiffBudgetBreakOutranksRegression(t *testing.T) {
	// Both predicates hold; the budget break is the single reported finding.
	base := doc(fn("F", bj(oN), budget(oN, "within")))
	head := doc(fn("F", bj(oN2), budget(oN, "exceeds")))
	fs, _, _ := Diff(base, head)
	if got := classesOf(fs); len(got) != 1 || got[0] != BudgetBreak {
		t.Errorf("classes = %v, want [BudgetBreak] only", got)
	}
}

func TestDiffNewTopNamesCause(t *testing.T) {
	head := fn("Parse", BoundJSON{Top: true}, nil)
	head.Causes = []CauseJSON{{Kind: "call", Detail: "unresolved cost at call to f", File: "p/parser.go", Line: 40}}
	fs, _, err := Diff(doc(fn("Parse", bj(oN), nil)), doc(head))
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].Class != NewTop {
		t.Fatalf("findings = %+v, want one NewTop", fs)
	}
	if !strings.Contains(fs[0].Message, "parser.go:40") {
		t.Errorf("message = %q, want the blocking position named", fs[0].Message)
	}
}

func TestDiffNewFunctionAlreadyExceeding(t *testing.T) {
	head := doc(fn("Added", bj(oN2), budget(oN, "exceeds")))
	fs, _, err := Diff(doc(), head)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].Class != NewFuncBreak {
		t.Fatalf("findings = %+v, want one NewFuncBreak", fs)
	}
}

func TestDiffImprovements(t *testing.T) {
	cases := []struct {
		name       string
		base, head Function
	}{
		{"exceeds to within", fn("F", bj(oN2), budget(oN, "exceeds")), fn("F", bj(oN), budget(oN, "within"))},
		{"bound tightened", fn("F", bj(oN2), nil), fn("F", bj(oNLogN), nil)},
		{"top to proven", fn("F", bj(oTop), nil), fn("F", bj(oN), nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs, _, _ := Diff(doc(tc.base), doc(tc.head))
			if len(fs) != 1 || fs[0].Class != Improvement {
				t.Fatalf("findings = %+v, want one Improvement", fs)
			}
		})
	}
}

// The silence cases. These are the reason diffing is immune to top-noise and
// are as load-bearing as any positive case.
func TestDiffSilence(t *testing.T) {
	cases := []struct {
		name       string
		base, head Function
	}{
		{"top to top", fn("F", bj(oTop), nil), fn("F", bj(oTop), nil)},
		{"unchanged bound", fn("F", bj(oN), nil), fn("F", bj(oN), nil)},
		{"unchanged within budget", fn("F", bj(oN), budget(oN, "within")), fn("F", bj(oN), budget(oN, "within"))},
		{"removed function", fn("F", bj(oN2), budget(oN, "exceeds")), Function{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var head Document
			if tc.head.Func != "" {
				head = doc(tc.head)
			} else {
				head = doc()
			}
			fs, _, _ := Diff(doc(tc.base), head)
			if len(fs) != 0 {
				t.Errorf("findings = %+v, want silence", fs)
			}
		})
	}
}

func TestDiffStillExceedingIsSilent(t *testing.T) {
	// Already broken in base and still broken: not a regression this PR caused.
	base := doc(fn("F", bj(oN2), budget(oN, "exceeds")))
	head := doc(fn("F", bj(oN2), budget(oN, "exceeds")))
	if fs, _, _ := Diff(base, head); len(fs) != 0 {
		t.Errorf("findings = %+v, want silence: the break predates this diff", fs)
	}
}

func TestDiffJoinsOnReceiver(t *testing.T) {
	// Same name, different receiver: two distinct functions, not a regression.
	a := fn("Len", bj(oN), nil)
	a.Receiver = "*Tree"
	b := fn("Len", bj(oN2), nil)
	b.Receiver = "*List"
	if fs, _, _ := Diff(doc(a), doc(a, b)); len(fs) != 0 {
		t.Errorf("findings = %+v, want silence: *List.Len is a new unbudgeted function", fs)
	}
}

func TestDiffOrdersBySeverityThenKey(t *testing.T) {
	base := doc(
		fn("A", bj(oN), budget(oN, "within")),
		fn("B", bj(oN), nil),
		fn("C", bj(oN2), nil),
	)
	head := doc(
		fn("A", bj(oN2), budget(oN, "exceeds")), // BudgetBreak
		fn("B", bj(oN2), nil),                   // ProvenRegression
		fn("C", bj(oN), nil),                    // Improvement
	)
	fs, _, _ := Diff(base, head)
	want := []Class{BudgetBreak, ProvenRegression, Improvement}
	got := classesOf(fs)
	if len(got) != len(want) {
		t.Fatalf("classes = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("classes = %v, want %v", got, want)
		}
	}
}

func TestDiffIncompatibleDocumentsError(t *testing.T) {
	head := doc()
	head.Module = "example.com/other"
	if _, _, err := Diff(doc(), head); err == nil {
		t.Error("Diff across modules = nil error, want error")
	}
}

// trustedFn is a document function with a provenance marking, built with the
// same bound serialization the rest of these tests use — a bare Str with no
// Terms is deliberately NOT a bound (see boundOf).
func trustedFn(name string, b bound.Bound, provenance string) Function {
	return Function{
		Package: "example.com/m", Func: name, File: "a.go", Line: 1,
		Time: bj(b), Provenance: provenance,
	}
}

// TestDiffIdenticalTrustBehavesAsBefore is the gate on the whole feature: when
// the trust surfaces match, provenance plays NO role and diff classifies
// exactly as it did before trust existed. The v1.33.0 review found this command
// silently masking regressions through its join key; a fix for one hazard must
// not introduce another.
func TestDiffIdenticalTrustBehavesAsBefore(t *testing.T) {
	trust := []AssumptionJSON{{Key: "os.Getenv", Bound: "O(1)"}}
	base := doc(trustedFn("A", bound.Constant(), ProvenanceTainted))
	base.Assumptions = trust
	head := doc(trustedFn("A", bound.Top(), ProvenanceTainted))
	head.Assumptions = trust

	findings, _, err := Diff(base, head)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Class != NewTop {
		t.Fatalf("findings = %+v, want one NewTop — identical trust must not change classification", findings)
	}
}

// TestDiffTrustChangeIsItsOwnClassAndCannotMaskARegression is a DIFFERENTIAL
// PAIR and must stay one. In a single run where the trust surface changed: the
// tainted function's move is TrustChanged, and the UNTAINTED function's
// regression is still reported. Asserting only the first would pass even if a
// trust edit swallowed every real finding.
func TestDiffTrustChangeIsItsOwnClassAndCannotMaskARegression(t *testing.T) {
	base := doc(
		trustedFn("Trusted", bound.Top(), ""),
		trustedFn("Plain", bound.Constant(), ""),
	)
	head := doc(
		trustedFn("Trusted", bound.Constant(), ProvenanceTainted),
		trustedFn("Plain", bound.Top(), ""),
	)
	head.Assumptions = []AssumptionJSON{{Key: "os.Getenv", Bound: "O(1)"}}

	findings, _, err := Diff(base, head)
	if err != nil {
		t.Fatal(err)
	}
	byFunc := map[string]Class{}
	for _, f := range findings {
		byFunc[f.Key] = f.Class
	}
	if got, ok := byFunc["example.com/m.Trusted"]; !ok || got != TrustChanged {
		t.Errorf("Trusted = %v (present=%v), want TrustChanged — a trust-driven move is not an improvement", got, ok)
	}
	if got, ok := byFunc["example.com/m.Plain"]; !ok || got != NewTop {
		t.Errorf("Plain = %v (present=%v), want NewTop — a trust edit must never mask a regression in untainted code", got, ok)
	}
}

// TestDiffWarnsWhenTrustSurfaceMoves: a reader must learn the INPUT changed
// before reading any verdict, and the warning names the entries rather than
// merely asserting that something moved.
func TestDiffWarnsWhenTrustSurfaceMoves(t *testing.T) {
	base := doc(trustedFn("A", bound.Top(), ""))
	head := doc(trustedFn("A", bound.Constant(), ProvenanceTainted))
	head.Assumptions = []AssumptionJSON{{Key: "os.Getenv", Bound: "O(1)"}}

	_, warn, err := Diff(base, head)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warn, "trust surface differs") || !strings.Contains(warn, "added os.Getenv") {
		t.Errorf("warning = %q, want it to name the added entry", warn)
	}
}

// A recognition is advisory, so a change in pattern matching must never
// surface as a complexity regression. Without this, adding a recognizer would
// break the shipped CI gate's contract: a run that newly recognizes a shape
// would report a finding even though no verdict moved.
//
// This passes because diff.go was NOT modified. The exclusion is obtained by
// construction -- Diff walks Functions, and recognitions live in a top-level
// array it never reads -- and this test is what pins that property so a later
// edit cannot quietly undo it.
func TestDiffIgnoresRecognitions(t *testing.T) {
	base := doc(Function{
		Package: "p", Func: "F", File: "p.go", Line: 1,
		Time: BoundJSON{Str: "O(1)"},
	})
	head := base
	head.Recognitions = []RecognitionJSON{{
		Package: "p", Func: "F", Pattern: "amortized two-pointer scan",
		Kind: "amortized", Bound: "O(n)", Assumption: "stated here",
		File: "p.go", Line: 1,
	}}

	fs, _, err := Diff(base, head)
	if err != nil {
		t.Fatalf("Diff = %v, want nil", err)
	}
	if len(fs) != 0 {
		t.Errorf("recognitions must not produce diff findings, got %+v", fs)
	}
}
