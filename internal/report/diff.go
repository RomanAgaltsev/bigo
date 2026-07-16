package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// Compat decides whether two documents may be compared, and reports a warning
// when they may be compared but the comparison deserves a caveat.
//
// Hard errors (apples-to-oranges): a different module, or a different schema
// major. Within a schema major the format is additive-only (document.go:5-7),
// so a minor difference is safe by construction.
//
// Warning: a bigo version difference. Bounds may then differ because the
// engine changed rather than because the code changed, and reporting that as a
// regression would be a false accusation. The envelope carries no analysis
// configuration today, so the version is the only signal available.
func Compat(base, head Document) (string, error) {
	if base.Module != head.Module {
		return "", fmt.Errorf("module mismatch: base %q, head %q", base.Module, head.Module)
	}
	bm, hm := schemaMajor(base.SchemaVersion), schemaMajor(head.SchemaVersion)
	if bm != hm {
		return "", fmt.Errorf("schema major mismatch: base %s, head %s", base.SchemaVersion, head.SchemaVersion)
	}
	if base.BigoVersion != head.BigoVersion {
		return fmt.Sprintf(
			"bigo version differs (base %s, head %s): bound changes may reflect the engine, not the code",
			base.BigoVersion, head.BigoVersion), nil
	}
	return "", nil
}

// schemaMajor extracts the leading major component of a semver string.
// An unparseable version yields "" and thus compares equal only to itself.
func schemaMajor(v string) string {
	if i := strings.IndexByte(v, '.'); i >= 0 {
		return v[:i]
	}
	return v
}

// boundOf reconstructs a bound from its structured serialization. ok is false
// when no bound was recorded — a zero BoundJSON must never read as O(1).
//
// This is the inverse of boundJSON (document.go:97) and exists so the diff can
// hand real bounds to bound.Check instead of comparing prose.
func boundOf(bj BoundJSON) (bound.Bound, bool) {
	if bj.Top {
		return bound.Top(), true
	}
	if bj.Terms == nil {
		return bound.Bound{}, false
	}
	ms := make([]bound.Monomial, 0, len(bj.Terms))
	for _, t := range bj.Terms {
		m := bound.One()
		for v, f := range t {
			m = m.Mul(bound.Mono(bound.Var(v), f.Pow, f.Log))
		}
		ms = append(ms, m)
	}
	return bound.Of(ms...), true
}

// Class is a finding's severity class, in the ecosystem spec's §5 order.
// Lower is more severe; Improvement is last because it is good news.
type Class int

const (
	// BudgetBreak - a declared budget went from within to exceeds
	BudgetBreak Class = iota
	// ProvenRegression - both sides proven, head is asymptotically worse
	ProvenRegression
	// NewTop - a proven bound became unverifiable
	NewTop
	// NewFuncBreak - a function was added already exceeding its budget
	NewFuncBreak
	// Improvement - exceeds→within, a tightened bound, or ⊤→proven
	Improvement
)

func (c Class) String() string {
	switch c {
	case BudgetBreak:
		return "budget break"
	case ProvenRegression:
		return "regression"
	case NewTop:
		return "new unverifiable"
	case NewFuncBreak:
		return "new function over budget"
	case Improvement:
		return "improvement"
	default:
		return "unknown"
	}
}

// Finding is one reportable difference.
type Finding struct {
	Class   Class
	Key     string // package.(receiver).func — the join identity
	File    string
	Line    int
	Message string
}

// key is a function's join identity: package + receiver + name. A rename is a
// remove plus an add, never tracked as a rename (spec §4).
func key(f Function) string {
	if f.Receiver != "" {
		return f.Package + ".(" + f.Receiver + ")." + f.Func
	}
	return f.Package + "." + f.Func
}

// Diff compares two report documents and returns findings ordered by severity
// then key, plus any compatibility warning.
//
// Silence is the default and is load-bearing: ⊤→⊤, unchanged bounds, removed
// functions, and pre-existing breaks all produce nothing. Only differences this
// diff can attribute to the change between base and head are reported.
func Diff(base, head Document) ([]Finding, string, error) {
	warn, err := Compat(base, head)
	if err != nil {
		return nil, "", err
	}
	prior := make(map[string]Function, len(base.Functions))
	for _, f := range base.Functions {
		prior[key(f)] = f
	}
	var out []Finding
	for _, h := range head.Functions {
		b, existed := prior[key(h)]
		if f, ok := classify(b, h, existed); ok {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Class != out[j].Class {
			return out[i].Class < out[j].Class
		}
		return out[i].Key < out[j].Key
	})
	return out, warn, nil
}

// classify decides the single finding a function pair yields, if any. Order is
// severity order: the first predicate that holds wins, so a function that both
// broke its budget and regressed reports once, as the budget break — the budget
// is the declared contract and the more actionable message.
func classify(b, h Function, existed bool) (Finding, bool) {
	at := func(c Class, msg string) (Finding, bool) {
		return Finding{Class: c, Key: key(h), File: h.File, Line: h.Line, Message: msg}, true
	}
	name := h.Func
	if h.Receiver != "" {
		name = "(" + h.Receiver + ")." + h.Func
	}

	// 4: added already exceeding. Nothing to compare against.
	if !existed {
		if h.Budget != nil && h.Budget.Verdict == "exceeds" {
			return at(NewFuncBreak, fmt.Sprintf("new function %s exceeds its %s budget: inferred %s",
				name, budgetStr(h.Budget), boundStr(h.Time)))
		}
		return Finding{}, false
	}

	// 1: within → exceeds.
	if b.Budget != nil && h.Budget != nil && b.Budget.Verdict == "within" && h.Budget.Verdict == "exceeds" {
		return at(BudgetBreak, fmt.Sprintf("%s exceeds its %s budget: inferred %s",
			name, budgetStr(h.Budget), boundStr(h.Time)))
	}

	bb, bok := boundOf(b.Time)
	hb, hok := boundOf(h.Time)
	if !bok || !hok {
		return Finding{}, false // no bound recorded on one side: nothing provable
	}

	// 3: proven → ⊤. Visibility loss is reportable, never silently absorbed.
	if !bb.IsTop() && hb.IsTop() {
		return at(NewTop, fmt.Sprintf("%s became unverifiable (was %s)%s", name, bb.String(), causeSuffix(h)))
	}
	// 5a: ⊤ → proven.
	if bb.IsTop() && !hb.IsTop() {
		return at(Improvement, fmt.Sprintf("%s is now provable: %s", name, hb.String()))
	}
	// ⊤ → ⊤: silent. This is the property that makes diffing immune to ⊤ noise.
	if bb.IsTop() && hb.IsTop() {
		return Finding{}, false
	}

	// Both proven. Reuse the shipped dominance algebra: Check(x, y) reports
	// whether x fits within y, so Exceeds in one direction is a regression and
	// in the other an improvement. No comparison logic is reimplemented here.
	switch {
	case bound.Check(hb, bb) == bound.Exceeds:
		// 2: head does not fit within base.
		return at(ProvenRegression, fmt.Sprintf("%s regressed: %s → %s", name, bb.String(), hb.String()))
	case bound.Check(bb, hb) == bound.Exceeds:
		// 5b: base does not fit within head — the bound tightened.
		return at(Improvement, fmt.Sprintf("%s improved: %s → %s", name, bb.String(), hb.String()))
	}

	// 5c: exceeds → within with an unchanged bound (e.g. the budget was raised
	// deliberately, or trust was added). Still good news, still reported.
	if b.Budget != nil && h.Budget != nil && b.Budget.Verdict == "exceeds" && h.Budget.Verdict == "within" {
		return at(Improvement, fmt.Sprintf("%s is now within its %s budget", name, budgetStr(h.Budget)))
	}
	return Finding{}, false
}

// boundStr renders a serialized bound for a message.
func boundStr(bj BoundJSON) string {
	if bj.Top {
		return "unverifiable"
	}
	if bj.Str == "" {
		return "no bound"
	}
	return bj.Str
}

// budgetStr renders a budget's normalized bound, falling back to the raw
// directive when the budget did not normalize.
func budgetStr(b *BudgetJSON) string {
	if b.Bound != nil && b.Bound.Str != "" {
		return b.Bound.Str
	}
	return b.Raw
}

// Severity reports the worst non-improvement class present, and whether any
// exists. Improvements are never severe: a change that only makes things better
// must never trip an exit-code policy.
func Severity(fs []Finding) (Class, bool) {
	worst, found := Improvement, false
	for _, f := range fs {
		if f.Class == Improvement {
			continue
		}
		if !found || f.Class < worst {
			worst, found = f.Class, true
		}
	}
	return worst, found
}

// causeSuffix names where a new ⊤ is blocked, using causes[0] — the same
// convention the metrics harness uses for its histogram.
func causeSuffix(h Function) string {
	if len(h.Causes) == 0 {
		return ""
	}
	c := h.Causes[0]
	if c.File == "" {
		return ": " + c.Detail
	}
	return fmt.Sprintf(": %s at %s:%d", c.Detail, c.File, c.Line)
}
