package frontier

import (
	"strings"
	"testing"
	"time"

	"github.com/RomanAgaltsev/bigo/internal/report"
)

// callTo builds the cause the engine emits for an unresolved call, so these
// fixtures exercise the same string the frontier walk parses in production.
func callTo(callee string) report.CauseJSON {
	return cause("call", CostPrefix+callee)
}

// keyedCallTo is callTo with the cost-table key the engine attaches alongside
// the sentence. The two DIFFER on generic code — the sentence carries the
// instantiation, the key is resolved to the origin — which is the whole subject
// of the 2026-08-12 review's F2.
func keyedCallTo(callee, key string) report.CauseJSON {
	c := callTo(callee)
	c.Callee = key
	return c
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
	fr := Of(doc)
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
	fr := Of(doc)
	// All three inherit C's single leaf: distance 1, not 1/2/3.
	if fr.Hist["1"] != 3 {
		t.Errorf("propagation should not count as distance, hist=%v", fr.Hist)
	}
	if fr.SoleBlocker[CostPrefix+"fmt.Errorf"] != 3 {
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
	done := make(chan Frontier, 1)
	go func() { done <- Of(doc) }()
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
	fr := Of(doc)
	// A must NOT inherit either init's leaf; the ambiguous call is its own leaf.
	if fr.SoleBlocker[CostPrefix+"example.com/m.init"] != 1 {
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
	fr := Of(doc)
	if fr.Top != 1 {
		t.Errorf("dependency ⊤ leaked into the count: Top=%d", fr.Top)
	}
	if _, ok := fr.SoleBlocker[CostPrefix+"fmt.Sprintf"]; ok {
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
	fr := Of(doc)
	if got := fr.SoleBlocker[CostPrefix+"fmt.Errorf"]; got != 1 {
		t.Errorf("only the single-blocker function counts, got %d", got)
	}
	if got := fr.SoleBlocker[CostPrefix+"fmt.Sprintf"]; got != 0 {
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
	fr := Of(doc)
	if fr.Near != 2 {
		t.Errorf("distance ≤2 should be 2 functions, got %d", fr.Near)
	}
	// Ceiling = (bounded + near) / functions = (1+2)/4 = 75.0
	if got := CeilingPct(1, fr.Near, 4); got != "75.0" {
		t.Errorf("ceiling_pct = %s, want 75.0", got)
	}
}

func TestCeilingPctZeroFunctionsIsNotADivideByZero(t *testing.T) {
	if got := CeilingPct(0, 0, 0); got != "0.0" {
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
	fr := Of(doc)
	if fr.Hist["10+"] != 1 {
		t.Errorf("12 blockers should land in 10+, hist=%v", fr.Hist)
	}
}

// TestDistanceOrderIsNumericNotRanked: the distance histogram's x-axis is
// ordinal. `ranked` sorts by count, and a plain string sort puts "10+" before
func TestFrontierExcludingFiltersPopulationNotWalk(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			// Hand-written caller of a generated ⊤ function.
			fn("example.com/m", "Caller", true, callTo("example.com/m.Gen")),
			// Generated, ⊤, and itself blocked by a real leaf.
			genFn("example.com/m", "Gen", true, callTo("sync.Once.Do")),
		},
	}
	skip := func(f report.Function) bool { return f.Func == "Gen" }

	fr := Excluding(doc, skip)

	if fr.Top != 1 {
		t.Errorf("only the hand-written function counts: Top = %d, want 1", fr.Top)
	}
	if got := fr.SoleBlocker[CostPrefix+"sync.Once.Do"]; got != 1 {
		t.Errorf("the caller must KEEP the leaf behind the generated callee, got %d want 1", got)
	}
	if fr.Hist["1"] != 1 {
		t.Errorf("caller sits at distance 1, hist=%v", fr.Hist)
	}
}

// TestFrontierOfIsUnfiltered pins that the existing entry point did not change
// meaning: with no skip, both functions are counted.
func TestFrontierOfIsUnfiltered(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			fn("example.com/m", "Caller", true, callTo("example.com/m.Gen")),
			genFn("example.com/m", "Gen", true, callTo("sync.Once.Do")),
		},
	}
	if fr := Of(doc); fr.Top != 2 {
		t.Errorf("frontierOf must count both: Top = %d, want 2", fr.Top)
	}
}

// TestMarkdownReportsBothPopulations pins that the split is visible in the
// rendered file: the unrebased aggregate, the hand-written headline, the
// generated count, and each ranking table's population.

// fn and cause build report fixtures. Duplicated from internal/survey rather
// than exported from it: these are test scaffolding, and a package that had to
// import the survey to test the frontier would recreate the dependency this
// extraction removed.
func fn(pkg, name string, top bool, causes ...report.CauseJSON) report.Function {
	return report.Function{
		Package: pkg, Func: name,
		Time:   report.BoundJSON{Top: top},
		Causes: causes,
	}
}

func cause(kind, detail string) report.CauseJSON {
	return report.CauseJSON{Kind: kind, Detail: detail}
}

// genFn is fn for a file the generated-code split would classify as machine
// written. Duplicated from internal/survey for the same reason as fn and cause.
func genFn(pkg, name string, top bool, causes ...report.CauseJSON) report.Function {
	f := fn(pkg, name, top, causes...)
	f.File = name + ".pb.go"
	return f
}

// TestGenericCalleePropagates is the DIFFERENTIAL PAIR for the 2026-08-12
// review's F2. Two callers of two otherwise identical ⊤ functions; the only
// difference is that one callee is generic.
//
// The cause sentence renders the INSTANTIATION — RelString gives
// "example.com/m.Drain[string]" — while the document records the DECLARATION,
// "Drain". The join missed, so the hop was scored as a LEAF: CallGeneric came
// out at distance 1 with a cost-table key, which made `bigo trust init` offer
// the user their own generic function and inflated Near, CeilingPct and the
// sole-blocker ranking the survey ranks work by.
//
// Both halves must be asserted. Checking only the generic caller's distance
// would pass under a fix that broke the non-generic one, and checking only that
// nothing is offered would pass if the walk stopped recursing entirely.
// The callees below carry TWO distinct leaves, so a missed hop is visible as a
// distance of 1 where the answer is 2. A single-leaf callee would score 1 both
// ways and the test would pass throughout the defect's life.
func TestGenericCalleePropagates(t *testing.T) {
	twoLeaves := []report.CauseJSON{callTo("fmt.Errorf"), callTo("time.Now")}
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			// The cause sentence renders the instantiation; Callee carries the
			// origin-resolved key, exactly as engine.go emits them.
			fn("example.com/m", "CallGeneric", true,
				keyedCallTo("example.com/m.Drain[string]", "example.com/m.Drain")),
			fn("example.com/m", "CallPlain", true,
				keyedCallTo("example.com/m.DrainInt", "example.com/m.DrainInt")),
			fn("example.com/m", "Drain", true, twoLeaves...),
			fn("example.com/m", "DrainInt", true, twoLeaves...),
		},
	}
	fr := Of(doc)

	// Every function bottoms out at the same two leaves, so all four sit at
	// distance 2. Before the fix CallGeneric sat at 1, because its hop was
	// mistaken for a blocker.
	if got := fr.Hist["2"]; got != 4 {
		t.Errorf("all four should sit at distance 2, hist=%v", fr.Hist)
	}
	if got := fr.Hist["1"]; got != 0 {
		t.Errorf("nothing here is one blocker from a bound, hist=%v", fr.Hist)
	}
	// A first-party generic must never be offerable: it is code the user can
	// edit, and the right tool for it is //bigo:cost, not a trust entry.
	if got := fr.SoleBlockerCallee["example.com/m.Drain"]; got != 0 {
		t.Errorf("bigo offered the user their own generic function %d time(s)", got)
	}
	// Same property at the per-function level, which is what the trust
	// generator attributes graduations with. The two must agree.
	for pos, key := range SoleBlockerIndex(doc) {
		if key == "example.com/m.Drain" {
			t.Errorf("SoleBlockerIndex disagrees with SoleBlockerCallee at %q", pos)
		}
	}
}

// TestGenericMethodCalleePropagates is the same defect on a method of a generic
// type, where the type arguments sit inside the RECEIVER rather than after the
// function name — and where the two sides disagree on BOTH sides of the join:
// the document renders the declared parameters "[K, V]" and the cause renders
// the arguments "[string,int]". Normalising only the cause would not fix it.
func TestGenericMethodCalleePropagates(t *testing.T) {
	declared := report.Function{
		Package: "example.com/m", Func: "Get", Receiver: "*Pair[K, V]",
		Time:   report.BoundJSON{Top: true},
		Causes: []report.CauseJSON{callTo("fmt.Errorf"), callTo("time.Now")},
	}
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			fn("example.com/m", "Caller", true, keyedCallTo(
				"(*example.com/m.Pair[string,int]).Get", "(*example.com/m.Pair[K, V]).Get")),
			declared,
		},
	}
	fr := Of(doc)
	if got := fr.Hist["2"]; got != 2 {
		t.Errorf("both should sit at distance 2, hist=%v", fr.Hist)
	}
	if len(fr.SoleBlockerCallee) != 0 {
		t.Errorf("the caller's hop is not a blocker, got %v", fr.SoleBlockerCallee)
	}
}

// TestNonGenericJoinUnaffected is the negative half: stripping type arguments
// must not disturb a name that has none, and must not merge two genuinely
// different callees.
func TestNonGenericJoinUnaffected(t *testing.T) {
	doc := report.Document{
		Module: "example.com/m",
		Functions: []report.Function{
			fn("example.com/m", "A", true, callTo("fmt.Errorf")),
			fn("example.com/m", "B", true, callTo("example.com/m.A")),
		},
	}
	fr := Of(doc)
	if got := fr.SoleBlocker[CostPrefix+"fmt.Errorf"]; got != 2 {
		t.Errorf("both should bottom out at fmt.Errorf, got %d", got)
	}
}

// TestSeenSetNotDepthCap pins that the walk terminates on mutual ⊤ recursion by
// remembering what it has visited, not by capping depth. A cap would silently
// truncate exactly the deepest chains this measure exists to report.
func TestSeenSetNotDepthCap(t *testing.T) {
	doc := report.Document{
		Module: "m",
		Functions: []report.Function{
			fn("m", "A", true, cause("call", CostPrefix+"m.B")),
			fn("m", "B", true, cause("call", CostPrefix+"m.A")),
		},
	}
	done := make(chan Frontier, 1)
	go func() { done <- Of(doc) }()
	select {
	case fr := <-done:
		if fr.Top != 2 {
			t.Errorf("Top = %d, want 2", fr.Top)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("frontier walk did not terminate on mutual ⊤ recursion")
	}
}

// TestSkipFiltersPopulationNotWalk pins the rule the generated-code split rests
// on: a skipped function leaves the SCORING POPULATION but the walk still
// recurses through it, because a counted function blocked BEHIND skipped code
// has a genuine blocker. Filtering the walk would erase real work from the
// queue, which is the opposite of why any caller skips.
func TestSkipFiltersPopulationNotWalk(t *testing.T) {
	doc := report.Document{
		Module: "m",
		Functions: []report.Function{
			fn("m", "Hand", true, cause("call", CostPrefix+"m.Gen")),
			fn("m", "Gen", true, cause("loop", "loop with unrecognized trip count")),
		},
	}
	fr := Excluding(doc, func(f report.Function) bool { return f.Func == "Gen" })
	if fr.Top != 1 {
		t.Fatalf("Top = %d, want 1 — Gen leaves the population", fr.Top)
	}
	if got := fr.SoleBlocker["loop with unrecognized trip count"]; got != 1 {
		t.Errorf("SoleBlocker = %d, want 1 — the walk must still recurse THROUGH Gen to reach its leaf", got)
	}
}

// TestSoleBlockerCalleeCountsOnlyKeyedBlockers pins the map trust init reads.
// It is keyed by the cost-table key, not by the human cause sentence, and a
// blocker with no key contributes to SoleBlocker but never to this map —
// because no trust entry could address it.
func TestSoleBlockerCalleeCountsOnlyKeyedBlockers(t *testing.T) {
	keyed := func(callee string) report.CauseJSON {
		c := cause("call", CostPrefix+callee)
		c.Callee = callee
		return c
	}
	doc := report.Document{
		Module: "m",
		Functions: []report.Function{
			fn("m", "A", true, keyed("os.Getenv")),
			fn("m", "B", true, keyed("os.Getenv")),
			// No key: an interface method. Counts in SoleBlocker, not here.
			fn("m", "C", true, cause("call", CostPrefix+"(m.R).Read")),
			// Two distinct blockers: sole-blocker for neither.
			fn("m", "D", true, keyed("os.Getenv"), cause("loop", "loop with unrecognized trip count")),
		},
	}
	fr := Of(doc)

	if got := fr.SoleBlockerCallee["os.Getenv"]; got != 2 {
		t.Errorf("SoleBlockerCallee[os.Getenv] = %d, want 2 — D has two blockers and counts for neither", got)
	}
	if got := len(fr.SoleBlockerCallee); got != 1 {
		t.Errorf("SoleBlockerCallee has %d keys, want 1 — an unkeyed blocker must not appear", got)
	}
	// The existing detail-keyed map is unchanged by this addition.
	if got := fr.SoleBlocker[CostPrefix+"(m.R).Read"]; got != 1 {
		t.Errorf("SoleBlocker for the unkeyed blocker = %d, want 1", got)
	}
}

// TestSoleBlockerIndexAttributesByPosition pins the per-function form used to
// attribute a graduation to the blocker that caused it. Position identity
// matters: two same-named functions in one package are legal Go, and a
// name-keyed index would collapse them.
func TestSoleBlockerIndexAttributesByPosition(t *testing.T) {
	a := fn("m", "init", true, keyedCause("os.Getenv"))
	a.File, a.Line = "a.go", 1
	b := fn("m", "init", true, keyedCause("os.Setenv"))
	b.File, b.Line = "a.go", 9
	// Two blockers: attributed to neither.
	c := fn("m", "C", true, keyedCause("os.Getenv"), cause("loop", "loop with unrecognized trip count"))
	c.File, c.Line = "a.go", 20

	idx := SoleBlockerIndex(report.Document{Module: "m", Functions: []report.Function{a, b, c}})
	if got := idx[PositionKey(a)]; got != "os.Getenv" {
		t.Errorf("first init attributed to %q, want os.Getenv", got)
	}
	if got := idx[PositionKey(b)]; got != "os.Setenv" {
		t.Errorf("second init attributed to %q, want os.Setenv — same name, different position", got)
	}
	if _, ok := idx[PositionKey(c)]; ok {
		t.Error("a two-blocker function must not be attributed to either")
	}
}

func keyedCause(callee string) report.CauseJSON {
	c := cause("call", CostPrefix+callee)
	c.Callee = callee
	return c
}
