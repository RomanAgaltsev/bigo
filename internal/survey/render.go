package survey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// detailTop bounds the rendered detail histogram. The full histogram is in the
// JSON; the Markdown is for reading, and a thousand-row table is not.
const detailTop = 30

// JSON renders the committed record: indented, map keys sorted by
// encoding/json, trailing newline.
func (r Report) JSON() []byte {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err) // no unmarshalable types in Report
	}
	return append(b, '\n')
}

// pair is one histogram row, for deterministic count-then-name ordering.
type pair struct {
	Key   string
	Count int
}

// ranked sorts a histogram by count descending, then key ascending. The name
// tiebreak is what makes the output byte-stable across runs — map iteration
// order must never reach the rendered file.
func ranked(h map[string]int) []pair {
	out := make([]pair, 0, len(h))
	for k, v := range h {
		out = append(out, pair{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// distanceOrder returns the distance buckets in NUMERIC order with the deep
// bucket last. `ranked` sorts by count, which would scramble a histogram whose
// x-axis is ordinal, and plain string sort puts "10+" before "2".
func distanceOrder(h map[string]int) []string {
	keys := make([]string, 0, len(h))
	deep := false
	for k := range h {
		if k == deepBucket {
			deep = true
			continue
		}
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, _ := strconv.Atoi(keys[i])
		b, _ := strconv.Atoi(keys[j])
		return a < b
	})
	if deep {
		keys = append(keys, deepBucket)
	}
	return keys
}

// Markdown renders SURVEY.md.
func (r Report) Markdown() []byte {
	var b bytes.Buffer
	b.WriteString("# bigo real-world survey\n\n")
	b.WriteString("GENERATED — do not edit; regenerate with `task survey`.\n\n")
	b.WriteString("**This is a MANUAL measurement, not a golden test.** No test asserts its\n")
	b.WriteString("contents and CI never runs it. Its targets are repositories that exist on one\n")
	b.WriteString("machine at whatever commit they happen to sit, so these numbers are a record\n")
	b.WriteString("of one run — compare across runs only via the per-target commit below.\n\n")
	fmt.Fprintf(&b, "Run %s with bigo %s.\n\n", r.Generated, r.BigoVersion)

	fmt.Fprintf(&b, "**Aggregate: %s%%** — %d of %d first-party functions bounded",
		r.Aggregate.CoveragePct, r.Aggregate.Bounded, r.Aggregate.Functions)
	if r.Aggregate.Seen > r.Aggregate.Functions {
		fmt.Fprintf(&b, " (%d functions seen in total; dependencies excluded)",
			r.Aggregate.Seen)
	}
	b.WriteString(".\n\n")

	fmt.Fprintf(&b, "**Hand-written: %s%%** — %d of %d functions bounded, with %d generated "+
		"functions excluded.\n\n", r.Aggregate.Hand.CoveragePct, r.Aggregate.Hand.Bounded,
		r.Aggregate.Hand.Functions, r.Aggregate.Generated)
	b.WriteString("Generated code is first-party by module path and is real code, but nobody\n")
	b.WriteString("hand-tunes it and its unverifiability is usually the CORRECT answer — the\n")
	b.WriteString("2026-07-21 `(*sync.Once).Do` probe measured 239 of that class's 326\n")
	b.WriteString("sole-blocker functions as generated protobuf whose verdict is right.\n")
	b.WriteString("**The aggregate above is kept unrebased** so it stays comparable with the\n")
	b.WriteString("2026-07-20/21 probes, which pin their population to it.\n\n")
	fmt.Fprintf(&b, "**Hand-written near frontier: %d of %d (%s%%), ceiling %s%%.**\n\n",
		r.Aggregate.Hand.NearFrontier, r.Aggregate.Hand.Top,
		pct(r.Aggregate.Hand.NearFrontier, r.Aggregate.Hand.Top), r.Aggregate.Hand.CeilingPct)

	fmt.Fprintf(&b, "**Near frontier: %d of %d unverifiable functions (%s%%) sit within %d "+
		"distinct blockers of a bound.** Clearing all of them would put coverage at "+
		"**%s%%** — an UPPER BOUND, not a forecast: clearing a blocker for one function "+
		"need not clear it for another. Two 2026-07-20 probes measured that gap directly "+
		"(`fmt`: 744 sole-blocker functions, 298 actually priceable; function values: 573, "+
		"zero reachable).\n\n",
		r.Aggregate.NearFrontier, r.Aggregate.Top,
		pct(r.Aggregate.NearFrontier, r.Aggregate.Top), nearDistance, r.Aggregate.CeilingPct)

	b.WriteString("## Per target\n\n")
	b.WriteString("| Target | Module | Commit | Functions | Bounded | Coverage | Generated | Hand | Hand cov | Near | Ceiling |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, t := range r.Targets {
		if t.Skipped != "" {
			fmt.Fprintf(&b, "| %s | — | — | — | — | skipped: %s | — | — | — | — | — |\n", t.Name, t.Skipped)
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %d | %s%% | %d | %d | %s%% | %d | %s%% |\n",
			t.Name, t.Module, t.Commit, t.Functions, t.Bounded, t.CoveragePct,
			t.Generated, t.Hand.Functions, t.Hand.CoveragePct,
			t.NearFrontier, t.CeilingPct)
	}

	b.WriteString("\n## Distance to bound\n\n")
	b.WriteString("How many DISTINCT leaf blockers stand between an unverifiable function and a\n")
	b.WriteString("bound, walking through propagation. This is why a single headline coverage\n")
	b.WriteString("number is misleading: it averages a near frontier that incremental work can\n")
	b.WriteString("reach against a deep tail that no achievable engine work will.\n\n")
	b.WriteString("| Blockers | Functions | Share |\n|---|---|---|\n")
	for _, k := range distanceOrder(r.Aggregate.DistanceHist) {
		fmt.Fprintf(&b, "| %s | %d | %s%% |\n", k, r.Aggregate.DistanceHist[k],
			pct(r.Aggregate.DistanceHist[k], r.Aggregate.Top))
	}

	b.WriteString("\n## Unverifiable by cause kind\n\n| Cause | Count |\n|---|---|\n")
	for _, p := range ranked(r.AggByCause) {
		fmt.Fprintf(&b, "| %s | %d |\n", p.Key, p.Count)
	}
	b.WriteString("\nCompare with `corpus/CORPUS.md`: the canonical corpus and real code do not\nagree on this ranking, and real code is the one that reflects adoption.\n")
	b.WriteString("\nPopulation: hand-written code only.\n")

	fmt.Fprintf(&b, "\n## Top %d blockers by GRADUATION count\n\n", detailTop)
	b.WriteString("**This table is the deliverable.** It counts functions whose ONLY blocker is\n")
	b.WriteString("each entry — the number that would actually graduate if it were cleared.\n\n")
	b.WriteString("A class here is one cause detail verbatim, so a function blocked by two\n")
	b.WriteString("different `fmt` calls counts toward neither: **these are a LOWER bound per\n")
	b.WriteString("class**, deliberately, because collapsing callee strings into classes is\n")
	b.WriteString("fragile and got it wrong once already.\n\n")
	b.WriteString("**Population: hand-written code only.** Generated functions are excluded\n")
	b.WriteString("here, because this table ranks work and generated code is not work anyone\n")
	b.WriteString("does. Before that exclusion the 2026-07-21 measurement had this table's\n")
	b.WriteString("`(*sync.Once).Do` row at 326 functions, 239 of them generated.\n\n")
	b.WriteString("| Blocker | Functions |\n|---|---|\n")
	for i, p := range ranked(r.AggSoleBlocker) {
		if i >= detailTop {
			break
		}
		fmt.Fprintf(&b, "| %s | %d |\n", escapePipes(p.Key), p.Count)
	}

	fmt.Fprintf(&b, "\n## Top %d blockers by SITES\n\n", detailTop)
	b.WriteString("**A concentration measure, not a work queue.** It shows where unverifiability\n")
	b.WriteString("clusters, never whether that blocker can be removed — the two 2026-07-20\n")
	b.WriteString("probes worked this ranking from the top down and produced no engine slice\n")
	b.WriteString("(`fmt` 8,367 sites → 298 priceable functions; function values 2,878 → zero).\n")
	b.WriteString("Rank work by the table above; use this one to understand shape.\n\n")
	b.WriteString("Population: hand-written code only, as above.\n\n")
	b.WriteString("| Blocker | Sites |\n|---|---|\n")
	for i, p := range ranked(r.AggByDetail) {
		if i >= detailTop {
			break
		}
		fmt.Fprintf(&b, "| %s | %d |\n", escapePipes(p.Key), p.Count)
	}
	return b.Bytes()
}

// escapePipes keeps a cause detail from breaking the Markdown table it sits in.
// Details are engine-generated prose and can contain generic type arguments
// with pipes in them.
func escapePipes(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		if r == '|' {
			b.WriteString("\\|")
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
