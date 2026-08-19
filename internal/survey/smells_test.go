package survey

import (
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/report"
)

func sm(pkg, rule, file string, line int) report.SmellJSON {
	return report.SmellJSON{
		Package: pkg,
		Rule:    rule,
		File:    file,
		Line:    line,
		Message: rule + " at " + file,
	}
}

// TestSummarizeSmellsExcludesDependencies mirrors TestSummarizeExcludesDependencies:
// a finding in somebody else's code is not a contribution, and before schema
// 1.4.0 there was no field that could express the difference.
func TestSummarizeSmellsExcludesDependencies(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Smells: []report.SmellJSON{
			sm("example.com/m", "SM1", "a.go", 10),
			sm("example.com/m/inner", "SM3", "inner/b.go", 20),
			sm("github.com/other/dep", "SM1", "dep.go", 30),
			// the boundary case: a DIFFERENT module sharing a prefix
			sm("example.com/mtools", "SM1", "tools.go", 40),
		},
	}
	got := SummarizeSmells(doc, nil)

	if len(got.Findings) != 2 {
		t.Fatalf("Findings = %d, want 2 (only example.com/m and example.com/m/inner): %+v",
			len(got.Findings), got.Findings)
	}
	if got.ByRule["SM1"] != 1 {
		t.Errorf("ByRule[SM1] = %d, want 1 — dependency findings leaked", got.ByRule["SM1"])
	}
	if got.ByRule["SM3"] != 1 {
		t.Errorf("ByRule[SM3] = %d, want 1", got.ByRule["SM3"])
	}
}

// TestSummarizeSmellsExcludesGenerated: nobody hand-tunes generated code, so a
// finding there is not a contribution. The 2026-07-21 (*sync.Once).Do probe
// measured 239 of 326 sole-blocker functions in one class as generated protobuf.
func TestSummarizeSmellsExcludesGenerated(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Smells: []report.SmellJSON{
			sm("example.com/m", "SM1", "hand.go", 1),
			sm("example.com/m", "SM1", "api.pb.go", 2),
		},
	}
	isGen := func(f string) bool { return f == "api.pb.go" }
	got := SummarizeSmells(doc, isGen)

	if len(got.Findings) != 1 {
		t.Fatalf("Findings = %d, want 1: %+v", len(got.Findings), got.Findings)
	}
	if got.Findings[0].File != "hand.go" {
		t.Errorf("kept %q, want hand.go", got.Findings[0].File)
	}
	if got.ByRule["SM1"] != 1 {
		t.Errorf("ByRule[SM1] = %d, want 1", got.ByRule["SM1"])
	}
}

// TestSummarizeSmellsNoModule matches frontier.FirstParty's documented
// behaviour: with no module recorded there is nothing to filter on, and
// counting nothing would silently empty the queue.
func TestSummarizeSmellsNoModule(t *testing.T) {
	doc := report.Document{
		Smells: []report.SmellJSON{sm("whatever/pkg", "SM2", "x.go", 1)},
	}
	if got := SummarizeSmells(doc, nil); len(got.Findings) != 1 {
		t.Errorf("Findings = %d, want 1 when the document records no module", len(got.Findings))
	}
}

// TestSummarizeSmellsPreservesOrder: report.Collect already sorts by file, line,
// rule. Re-sorting here would create a second source of truth for the order the
// probe's sampling rule is pinned to.
func TestSummarizeSmellsPreservesOrder(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Smells: []report.SmellJSON{
			sm("example.com/m", "SM1", "a.go", 5),
			sm("example.com/m", "SM7", "a.go", 9),
			sm("example.com/m", "SM3", "b.go", 1),
		},
	}
	got := SummarizeSmells(doc, nil)
	want := []struct {
		file string
		line int
	}{{"a.go", 5}, {"a.go", 9}, {"b.go", 1}}
	for i, w := range want {
		if got.Findings[i].File != w.file || got.Findings[i].Line != w.line {
			t.Errorf("Findings[%d] = %s:%d, want %s:%d",
				i, got.Findings[i].File, got.Findings[i].Line, w.file, w.line)
		}
	}
}

// TestSampleCapsPerRule is the anti-cherry-picking control. SM3 is predicted to
// dominate by volume; without the cap it would fill the sample and the probe
// would measure one rule instead of eight.
func TestSampleCapsPerRule(t *testing.T) {
	var findings []SmellFinding
	for i := 0; i < 20; i++ {
		findings = append(findings, SmellFinding{Rule: "SM3", File: "a.go", Line: i})
	}
	for i := 0; i < 5; i++ {
		findings = append(findings, SmellFinding{Rule: "SM8", File: "b.go", Line: i})
	}

	got := Sample(findings, 40, 8, 0)

	if len(got) != 13 {
		t.Fatalf("len = %d, want 13 (8 SM3 capped + 5 SM8)", len(got))
	}
	counts := map[string]int{}
	for _, f := range got {
		counts[f.Rule]++
	}
	if counts["SM3"] != 8 {
		t.Errorf("SM3 = %d, want 8 (capped)", counts["SM3"])
	}
	if counts["SM8"] != 5 {
		t.Errorf("SM8 = %d, want 5 (under the cap, all admitted)", counts["SM8"])
	}
}

// TestSampleStopsAtN: the bar is stated for a sample of 40, so the sample must
// not exceed it even when many rules are under their cap.
func TestSampleStopsAtN(t *testing.T) {
	var findings []SmellFinding
	for r := 1; r <= 8; r++ {
		for i := 0; i < 8; i++ {
			findings = append(findings, SmellFinding{
				Rule: "SM" + string(rune('0'+r)), File: "a.go", Line: i,
			})
		}
	}
	if got := Sample(findings, 40, 8, 0); len(got) != 40 {
		t.Errorf("len = %d, want exactly 40 from a 64-finding pool", len(got))
	}
}

// TestSampleIsDeterministic: two draws from the same input must be identical, or
// the thresholds file's committed sampling rule means nothing.
func TestSampleIsDeterministic(t *testing.T) {
	var findings []SmellFinding
	for i := 0; i < 30; i++ {
		findings = append(findings, SmellFinding{
			Rule: []string{"SM1", "SM3", "SM8"}[i%3], File: "a.go", Line: i,
		})
	}
	a, b := Sample(findings, 20, 8, 0), Sample(findings, 20, 8, 0)
	if len(a) != len(b) {
		t.Fatalf("lengths differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("draw %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

// TestSampleDoesNotMutateInput: Run keeps the full finding list for the JSON
// artifact after sampling for the Markdown one.
func TestSampleDoesNotMutateInput(t *testing.T) {
	findings := []SmellFinding{
		{Rule: "SM1", File: "a.go", Line: 1},
		{Rule: "SM1", File: "a.go", Line: 2},
		{Rule: "SM3", File: "b.go", Line: 3},
	}
	before := append([]SmellFinding(nil), findings...)
	Sample(findings, 2, 1, 0)
	for i := range before {
		if findings[i] != before[i] {
			t.Errorf("input mutated at %d: %+v became %+v", i, before[i], findings[i])
		}
	}
}

// TestSampleCapsPerTarget is Amendment 1 to the probe thresholds. Volume is
// uneven along two independent axes: the per-rule cap stops one prolific RULE
// from filling the sample, and this one stops one prolific TARGET. The first
// scan drew 28 of 40 rows from caddy alone without it.
func TestSampleCapsPerTarget(t *testing.T) {
	var findings []SmellFinding
	// One dominant target with plenty of every rule.
	for i := 0; i < 30; i++ {
		findings = append(findings, SmellFinding{
			Target: "big", Rule: []string{"SM1", "SM3", "SM5"}[i%3], File: "a.go", Line: i,
		})
	}
	for i := 0; i < 10; i++ {
		findings = append(findings, SmellFinding{
			Target: "small", Rule: "SM6", File: "b.go", Line: i,
		})
	}

	got := Sample(findings, 40, 8, 6)

	byTarget := map[string]int{}
	for _, f := range got {
		byTarget[f.Target]++
	}
	if byTarget["big"] != 6 {
		t.Errorf("big = %d, want 6 (capped)", byTarget["big"])
	}
	if byTarget["small"] != 6 {
		t.Errorf("small = %d, want 6 (capped)", byTarget["small"])
	}
	if len(got) != 12 {
		t.Errorf("len = %d, want 12; both targets capped at 6", len(got))
	}
}

// TestSampleBothCapsBind: whichever cap is reached first must exclude the
// finding, so neither axis can be starved by the other.
func TestSampleBothCapsBind(t *testing.T) {
	var findings []SmellFinding
	// One target, one rule: the RULE cap is the tighter of the two here.
	for i := 0; i < 20; i++ {
		findings = append(findings, SmellFinding{
			Target: "t", Rule: "SM3", File: "a.go", Line: i,
		})
	}
	if got := Sample(findings, 40, 3, 6); len(got) != 3 {
		t.Errorf("len = %d, want 3 — the per-rule cap is tighter and must bind", len(got))
	}
	if got := Sample(findings, 40, 8, 2); len(got) != 2 {
		t.Errorf("len = %d, want 2 — the per-target cap is tighter and must bind", len(got))
	}
}
