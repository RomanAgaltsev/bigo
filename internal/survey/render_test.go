package survey

import (
	"strings"
	"testing"
)

func TestMarkdownRendersBothBlockerTables(t *testing.T) {
	r := Report{
		Generated: "2026-07-20", BigoVersion: "1.36.0",
		AggByCause:     map[string]int{"call": 3},
		AggByDetail:    map[string]int{"unresolved cost at call to fmt.Errorf": 9},
		AggSoleBlocker: map[string]int{"unresolved cost at call to fmt.Errorf": 2},
	}
	r.Aggregate = Totals{
		Functions: 10, Bounded: 4, Seen: 10, CoveragePct: "40.0",
		Top: 6, NearFrontier: 3, CeilingPct: "70.0",
		DistanceHist: map[string]int{"1": 2, "2": 1, "10+": 3},
	}
	md := string(r.Markdown())

	for _, want := range []string{
		"Near frontier: 3 of 6",
		"UPPER BOUND, not a forecast",
		"blockers by GRADUATION count",
		"**This table is the deliverable.**",
		"blockers by SITES",
		"A concentration measure, not a work queue.",
		"## Distance to bound",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("SURVEY.md missing %q", want)
		}
	}
	// The deliverable label must sit on the graduation table, not the sites one.
	if strings.Index(md, "**This table is the deliverable.**") > strings.Index(md, "blockers by SITES") {
		t.Error("the deliverable label is attached to the sites table")
	}
}

// TestFrontierExcludingFiltersPopulationNotWalk is the load-bearing
// distinction of the generated-code split. A HAND-WRITTEN function whose only
// blocker sits behind a GENERATED callee still has a genuine blocker: the
// generated code stands between real user code and a bound. Truncating the
// walk there would erase that blocker from the work queue, which is the
// opposite of the split's purpose.

func TestMarkdownReportsBothPopulations(t *testing.T) {
	r := Report{
		Generated:   "2026-07-21",
		BigoVersion: "test",
		Aggregate: Totals{
			Functions: 100, Bounded: 40, CoveragePct: "40.0",
			Seen: 100, Top: 60, NearFrontier: 30, CeilingPct: "70.0",
			Generated: 20,
			Hand: HandTotals{
				Functions: 80, Bounded: 36, CoveragePct: "45.0",
				Top: 44, NearFrontier: 22, CeilingPct: "72.5",
			},
			DistanceHist: map[string]int{"1": 30, "2": 30},
		},
		AggByCause:     map[string]int{"call": 1},
		AggByDetail:    map[string]int{"unresolved cost at call to fmt.Errorf": 1},
		AggSoleBlocker: map[string]int{"unresolved cost at call to fmt.Errorf": 1},
	}
	md := string(r.Markdown())

	if !strings.Contains(md, "**Aggregate: 40.0%**") {
		t.Error("the all-first-party headline must survive unchanged")
	}
	if !strings.Contains(md, "**Hand-written: 45.0%**") {
		t.Error("the hand-written headline is missing")
	}
	if !strings.Contains(md, "20 generated") {
		t.Error("the generated count must be visible, never silent")
	}
	if !strings.Contains(md, "hand-written code only") {
		t.Error("the tables must state their population")
	}
}

// "2" — either would render a scrambled histogram.
func TestDistanceOrderIsNumericNotRanked(t *testing.T) {
	got := distanceOrder(map[string]int{"10+": 99, "2": 1, "1": 50, "9": 2})
	want := []string{"1", "2", "9", "10+"}
	if len(got) != len(want) {
		t.Fatalf("distanceOrder = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("distanceOrder = %v, want %v", got, want)
		}
	}
}

// TestMarkdownRendersBothBlockerTables pins that the graduation table is the one
// labelled as the deliverable, and that the sites table survives as an
// explicitly-labelled concentration measure rather than being deleted.
