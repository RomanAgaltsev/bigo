package survey

import (
	"strings"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/report"
)

func fn(pkg, name string, top bool, causes ...report.CauseJSON) report.Function {
	f := report.Function{Package: pkg, Func: name, File: name + ".go", Line: 1}
	f.Time = report.BoundJSON{Top: top}
	if top {
		f.Causes = causes
	}
	return f
}

func cause(kind, detail string) report.CauseJSON {
	return report.CauseJSON{Kind: kind, Detail: detail}
}

// TestSummarizeExcludesDependencies is the harness's correctness crux. An
// ad-hoc run over prometheus counted pb33f/libopenapi symbols — somebody else's
// code — and any coverage number computed over those measures the wrong thing.
func TestSummarizeExcludesDependencies(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			fn("example.com/m", "A", false),
			fn("example.com/m/inner", "B", false),
			fn("example.com/m", "C", true, cause("call", "unresolved cost at call to fmt.Errorf")),
			// dependencies — must not be counted
			fn("github.com/other/dep", "D", false),
			fn("github.com/other/dep/x", "E", true, cause("loop", "loop with unrecognized trip count")),
			// the boundary case: a DIFFERENT module sharing a prefix
			fn("example.com/mtools", "F", false),
		},
	}
	got, byCause, byDetail, _ := Summarize(doc)

	if got.Functions != 3 {
		t.Errorf("Functions = %d, want 3 (only example.com/m and example.com/m/inner)", got.Functions)
	}
	if got.Seen != 6 {
		t.Errorf("Seen = %d, want 6 — the excluded total must stay visible", got.Seen)
	}
	if got.Bounded != 2 {
		t.Errorf("Bounded = %d, want 2", got.Bounded)
	}
	if got.CoveragePct != "66.7" {
		t.Errorf("CoveragePct = %q, want \"66.7\"", got.CoveragePct)
	}
	if byCause["loop"] != 0 {
		t.Errorf("dependency causes leaked into the histogram: %v", byCause)
	}
	if byDetail["unresolved cost at call to fmt.Errorf"] != 1 {
		t.Errorf("first-party detail missing: %v", byDetail)
	}
}

// TestFirstPartyBoundary pins the prefix boundary on its own: a sibling module
// whose path merely starts with ours is not ours.
func TestFirstPartyBoundary(t *testing.T) {
	cases := []struct {
		pkg, module string
		want        bool
	}{
		{"example.com/m", "example.com/m", true},
		{"example.com/m/sub", "example.com/m", true},
		{"example.com/mtools", "example.com/m", false},
		{"example.com/other", "example.com/m", false},
		{"anything", "", true}, // no module recorded: cannot filter
	}
	for _, c := range cases {
		if got := firstParty(c.pkg, c.module); got != c.want {
			t.Errorf("firstParty(%q, %q) = %v, want %v", c.pkg, c.module, got, c.want)
		}
	}
}

// TestSummarizeEmptyIsNotADivideByZero: a target whose functions are all
// filtered out is a legitimate outcome, not a crash.
func TestSummarizeEmptyIsNotADivideByZero(t *testing.T) {
	doc := report.Document{
		Module:    "example.com/m",
		Functions: []report.Function{fn("github.com/other/dep", "D", false)},
	}
	got, _, _, _ := Summarize(doc)
	if got.Functions != 0 || got.CoveragePct != "0.0" {
		t.Errorf("empty target = %+v, want 0 functions and \"0.0\"", got)
	}
	if got.Seen != 1 {
		t.Errorf("Seen = %d, want 1", got.Seen)
	}
}

// TestRunSkipsMissingPathWithAReason: a silently dropped target would quietly
// bias every aggregate, so a skip must be visible and must not contribute.
func TestRunSkipsMissingPathWithAReason(t *testing.T) {
	r := Run(Config{Targets: []TargetConfig{
		{Name: "absent", Path: filepathNeverExists},
	}}, "test", nil)

	if len(r.Targets) != 1 {
		t.Fatalf("targets = %d, want 1 — a skip is still a row", len(r.Targets))
	}
	if r.Targets[0].Skipped == "" {
		t.Error("missing path produced no Skipped reason")
	}
	if r.Aggregate.Functions != 0 {
		t.Errorf("skipped target contributed %d functions to the aggregate", r.Aggregate.Functions)
	}
	if !strings.Contains(string(r.Markdown()), "skipped:") {
		t.Error("SURVEY.md does not surface the skip")
	}
}

const filepathNeverExists = "Z:/no/such/path/for/bigo/survey"

// TestMarkdownIsDeterministic: the rendered file is committed, so identical
// input must produce identical bytes — map iteration order must never reach it.
func TestMarkdownIsDeterministic(t *testing.T) {
	r := Report{
		Generated: "2026-07-20", BigoVersion: "test",
		Targets: []Target{{
			Name: "t", Module: "example.com/m", Commit: "abc1234",
			Totals: Totals{Functions: 10, Bounded: 4, Seen: 20, CoveragePct: "40.0"},
		}},
		Aggregate:   Totals{Functions: 10, Bounded: 4, Seen: 20, CoveragePct: "40.0"},
		AggByCause:  map[string]int{"call": 5, "loop": 5, "go": 2, "defer": 1},
		AggByDetail: map[string]int{"a": 3, "b": 3, "c": 1, "d": 9},
	}
	first := string(r.Markdown())
	for i := 0; i < 12; i++ {
		if got := string(r.Markdown()); got != first {
			t.Fatal("Markdown is not deterministic across renders")
		}
	}
	// equal counts must break ties by name, so "a" precedes "b"
	if strings.Index(first, "| a | 3 |") > strings.Index(first, "| b | 3 |") {
		t.Error("equal-count details are not name-ordered")
	}
	// and the highest count leads
	if strings.Index(first, "| d | 9 |") > strings.Index(first, "| a | 3 |") {
		t.Error("detail histogram is not count-ordered")
	}
}

// TestMarkdownEscapesPipes: cause details are engine prose and can contain
// generic type arguments; an unescaped pipe would break the table.
func TestMarkdownEscapesPipes(t *testing.T) {
	r := Report{
		Generated: "2026-07-20", BigoVersion: "test",
		AggByCause:  map[string]int{},
		AggByDetail: map[string]int{"call to Map[string|int]": 2},
	}
	if !strings.Contains(string(r.Markdown()), `Map[string\|int]`) {
		t.Error("pipe in a cause detail was not escaped")
	}
}
