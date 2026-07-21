package survey

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/RomanAgaltsev/bigo/internal/report"
)

// costPrefix is the engine's cause text for a call whose cost did not resolve.
// The frontier walk parses it to tell PROPAGATION (the callee is itself ⊤, so
// this is a hop) from a real LEAF blocker.
const costPrefix = "unresolved cost at call to "

// deepBucket collapses the tail of the distance histogram. A function ten
// blockers from a bound is not meaningfully different from one twenty away —
// both are out of reach — so the exact depth is not worth a column.
const deepBucket = "10+"

// nearDistance is the near-frontier cutoff: ⊤ functions within this many
// distinct leaf blockers of a bound.
const nearDistance = 2

// frontier is the distance measurement over one document's first-party
// functions.
type frontier struct {
	Top  int            // ⊤ functions
	Near int            // ⊤ functions at distance ≤ nearDistance
	Hist map[string]int // distance bucket -> functions

	// SoleBlocker counts, per leaf-blocker detail, the functions whose leaf set
	// is EXACTLY that one detail — the graduation count.
	//
	// Keyed by the cause detail VERBATIM, never by a collapsed package class.
	// A function blocked by both fmt.Errorf and fmt.Sprintf therefore counts
	// toward neither, so these are a LOWER bound per class. That is deliberate:
	// collapsing callee strings into classes was implemented once, during the
	// 2026-07-20 fmt probe, and got it wrong — it split at the dot inside
	// "github.com" and merged every third-party callee into one bucket of 527.
	// Under-counting a class is safe; inventing one is not.
	SoleBlocker map[string]int
}

// funcKey renders a function the way a cause detail names its callee, so the
// two can be joined: "(*pkg.T).Method", "(pkg.Iface).Method", or "pkg.Func".
func funcKey(f report.Function) string {
	if f.Receiver == "" {
		return f.Package + "." + f.Func
	}
	star, name := "", f.Receiver
	if strings.HasPrefix(name, "*") {
		star, name = "*", name[1:]
	}
	return "(" + star + f.Package + "." + name + ")." + f.Func
}

// leafSet collects the distinct leaf blockers standing between fn and a bound.
//
// A cause naming a callee that is itself ⊤ in this document is propagation, and
// the walk recurses into it; everything else is a leaf. Two rules matter:
//
//   - seen is a seen-SET, not a depth cap. Mutual recursion between ⊤ functions
//     is ordinary in real Go, and a cap would silently truncate exactly the
//     deepest chains this metric exists to measure.
//   - an AMBIGUOUS callee key resolves to nothing and stays a leaf. Several
//     same-named functions in one package is legal Go (the `bigo diff` F1
//     shape); picking one arbitrarily would fabricate a chain.
func leafSet(idx int, funcs []report.Function, byKey map[string][]int, seen map[int]bool, out map[string]bool) {
	if seen[idx] {
		return
	}
	seen[idx] = true
	for _, c := range funcs[idx].Causes {
		if callee, ok := strings.CutPrefix(c.Detail, costPrefix); ok {
			if hits := byKey[callee]; len(hits) == 1 && funcs[hits[0]].Time.Top {
				leafSet(hits[0], funcs, byKey, seen, out)
				continue
			}
		}
		out[c.Detail] = true
	}
}

// bucket names the histogram column for a distance.
func bucket(d int) string {
	if d >= 10 {
		return deepBucket
	}
	return strconv.Itoa(d)
}

// frontierOf measures distance to bound for every first-party ⊤ function.
//
// Pure: it reads the document the report already emits and computes nothing the
// engine does not already know. There is no inference here and no soundness
// surface — a wrong distance is a wrong PRIORITY, never a wrong bound.
func frontierOf(doc report.Document) frontier {
	return frontierExcluding(doc, nil)
}

// frontierExcluding is frontierOf over a population that omits every function
// for which skip returns true. A nil skip measures everything.
//
// THE SKIP FILTERS THE POPULATION, NEVER THE WALK. leafSet is untouched and
// still recurses through skipped ⊤ functions, because a counted function whose
// blocker sits BEHIND a skipped one has a genuine blocker — the skipped code
// stands between it and a bound. Filtering the walk instead would erase real
// work from the queue, which is the exact opposite of why any caller skips.
//
// Used by the generated-code split: generated functions leave the scoring
// population, while a hand-written caller of a generated ⊤ function keeps the
// leaf it actually waits on.
func frontierExcluding(doc report.Document, skip func(report.Function) bool) frontier {
	fr := frontier{Hist: map[string]int{}, SoleBlocker: map[string]int{}}

	byKey := make(map[string][]int, len(doc.Functions))
	for i, f := range doc.Functions {
		byKey[funcKey(f)] = append(byKey[funcKey(f)], i)
	}

	for i, f := range doc.Functions {
		if !firstParty(f.Package, doc.Module) || !f.Time.Top {
			continue
		}
		if skip != nil && skip(f) {
			continue // population only — the walk below still passes through it
		}
		fr.Top++

		leaves := map[string]bool{}
		leafSet(i, doc.Functions, byKey, map[int]bool{}, leaves)

		d := len(leaves)
		fr.Hist[bucket(d)]++
		if d <= nearDistance {
			fr.Near++
		}
		if d == 1 {
			for detail := range leaves {
				fr.SoleBlocker[detail]++
			}
		}
	}
	return fr
}

// ceilingPct renders coverage as it would stand if every near-frontier function
// were bounded.
//
// This is an UPPER BOUND and must never be quoted as a projection. Clearing a
// blocker for one function need not clear it for another, and the two 2026-07-20
// probes measured exactly that gap: `fmt` had 744 sole-blocker functions of
// which 298 were actually priceable, and unresolved function values had 573 of
// which ZERO were reachable. The metric measures distance, never removability.
func ceilingPct(bounded, near, functions int) string {
	if functions == 0 {
		return "0.0"
	}
	return fmt.Sprintf("%.1f", float64(bounded+near)*100/float64(functions))
}
