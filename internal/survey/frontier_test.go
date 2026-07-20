package survey

import (
	"strings"
	"testing"
	"time"

	"github.com/RomanAgaltsev/bigo/internal/report"
)

// callTo builds the cause the engine emits for an unresolved call, so these
// fixtures exercise the same string the frontier walk parses in production.
func callTo(callee string) report.CauseJSON {
	return cause("call", costPrefix+callee)
}

func TestDistanceCountsDistinctLeaves(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			// Two distinct leaf blockers ⇒ distance 2.
			fn("example.com/m", "A", true, callTo("fmt.Errorf"), callTo("time.Now")),
			// The same leaf twice ⇒ distance 1: distance counts DISTINCT blockers.
			fn("example.com/m", "B", true, callTo("fmt.Errorf"), callTo("fmt.Errorf")),
		},
	}
	fr := frontierOf(doc)
	if got := fr.Hist["2"]; got != 1 {
		t.Errorf("A should sit at distance 2, hist=%v", fr.Hist)
	}
	if got := fr.Hist["1"]; got != 1 {
		t.Errorf("B should sit at distance 1, hist=%v", fr.Hist)
	}
}

// TestDistanceRecursesThroughPropagation is the point of the metric: a call to
// another ⊤ function is not a blocker, it is a hop toward one.
func TestDistanceRecursesThroughPropagation(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			fn("example.com/m", "A", true, callTo("example.com/m.B")),
			fn("example.com/m", "B", true, callTo("example.com/m.C")),
			fn("example.com/m", "C", true, callTo("fmt.Errorf")),
		},
	}
	fr := frontierOf(doc)
	// All three inherit C's single leaf: distance 1, not 1/2/3.
	if fr.Hist["1"] != 3 {
		t.Errorf("propagation should not count as distance, hist=%v", fr.Hist)
	}
	if fr.SoleBlocker[costPrefix+"fmt.Errorf"] != 3 {
		t.Errorf("all three are sole-blocked by fmt.Errorf, got %v", fr.SoleBlocker)
	}
}

// TestPropagationCycleTerminates pins the seen-set. Mutual recursion between ⊤
// functions is ordinary in real Go; a depth cap would silently truncate exactly
// the deepest chains this metric exists to measure, and no cap at all hangs.
func TestPropagationCycleTerminates(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			fn("example.com/m", "A", true, callTo("example.com/m.B")),
			fn("example.com/m", "B", true, callTo("example.com/m.A"), callTo("fmt.Errorf")),
		},
	}
	done := make(chan frontier, 1)
	go func() { done <- frontierOf(doc) }()
	select {
	case fr := <-done:
		if fr.Hist["1"] != 2 {
			t.Errorf("both cycle members reach one leaf, hist=%v", fr.Hist)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("frontier walk did not terminate on a propagation cycle")
	}
}

// TestAmbiguousCalleeIsALeaf: several same-named functions in a package is legal
// Go (the `bigo diff` F1 shape). The walk must not pick one arbitrarily — an
// unresolvable key is a leaf, not a guess.
func TestAmbiguousCalleeIsALeaf(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			fn("example.com/m", "A", true, callTo("example.com/m.init")),
			fn("example.com/m", "init", true, callTo("fmt.Errorf")),
			fn("example.com/m", "init", true, callTo("time.Now")),
		},
	}
	fr := frontierOf(doc)
	// A must NOT inherit either init's leaf; the ambiguous call is its own leaf.
	if fr.SoleBlocker[costPrefix+"example.com/m.init"] != 1 {
		t.Errorf("ambiguous callee should be a leaf, got %v", fr.SoleBlocker)
	}
}

func TestFrontierExcludesDependencies(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			fn("example.com/m", "A", true, callTo("fmt.Errorf")),
			fn("other.com/dep", "D", true, callTo("fmt.Sprintf")),
		},
	}
	fr := frontierOf(doc)
	if fr.Top != 1 {
		t.Errorf("dependency ⊤ leaked into the count: Top=%d", fr.Top)
	}
	if _, ok := fr.SoleBlocker[costPrefix+"fmt.Sprintf"]; ok {
		t.Errorf("dependency blocker leaked into sole-blocker: %v", fr.SoleBlocker)
	}
}

// TestSoleBlockerExcludesMultiBlockerFunctions pins the spec's §3 decision: a
// function blocked by two different fmt calls counts toward NEITHER, because
// counts key on the detail verbatim rather than a collapsed package class.
// These counts are a LOWER bound per class, deliberately — the class-collapsing
// parser is what got the fmt probe's first ranking wrong.
func TestSoleBlockerExcludesMultiBlockerFunctions(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			fn("example.com/m", "One", true, callTo("fmt.Errorf")),
			fn("example.com/m", "Two", true, callTo("fmt.Errorf"), callTo("fmt.Sprintf")),
		},
	}
	fr := frontierOf(doc)
	if got := fr.SoleBlocker[costPrefix+"fmt.Errorf"]; got != 1 {
		t.Errorf("only the single-blocker function counts, got %d", got)
	}
	if got := fr.SoleBlocker[costPrefix+"fmt.Sprintf"]; got != 0 {
		t.Errorf("a two-blocker function must count toward neither, got %d", got)
	}
}

func TestNearFrontierAndCeiling(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			fn("example.com/m", "Bounded", false),
			fn("example.com/m", "Near1", true, callTo("fmt.Errorf")),
			fn("example.com/m", "Near2", true, callTo("fmt.Errorf"), callTo("time.Now")),
			fn("example.com/m", "Far", true,
				callTo("a.A"), callTo("b.B"), callTo("c.C")),
		},
	}
	fr := frontierOf(doc)
	if fr.Near != 2 {
		t.Errorf("distance ≤2 should be 2 functions, got %d", fr.Near)
	}
	// Ceiling = (bounded + near) / functions = (1+2)/4 = 75.0
	if got := ceilingPct(1, fr.Near, 4); got != "75.0" {
		t.Errorf("ceiling_pct = %s, want 75.0", got)
	}
}

func TestCeilingPctZeroFunctionsIsNotADivideByZero(t *testing.T) {
	if got := ceilingPct(0, 0, 0); got != "0.0" {
		t.Errorf("empty target should render 0.0, got %s", got)
	}
}

// TestDistanceBucketsCapAtTen keeps the histogram readable: the deep tail is one
// bucket, since its exact depth is not actionable.
func TestDistanceBucketsCapAtTen(t *testing.T) {
	var causes []report.CauseJSON
	for _, c := range strings.Split("a b c d e f g h i j k l", " ") {
		causes = append(causes, callTo(c+".F"))
	}
	doc := report.Document{
		Module:    "example.com/m",
		Functions: []report.Function{fn("example.com/m", "Deep", true, causes...)},
	}
	fr := frontierOf(doc)
	if fr.Hist["10+"] != 1 {
		t.Errorf("12 blockers should land in 10+, hist=%v", fr.Hist)
	}
}

// TestDistanceOrderIsNumericNotRanked: the distance histogram's x-axis is
// ordinal. `ranked` sorts by count, and a plain string sort puts "10+" before
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
