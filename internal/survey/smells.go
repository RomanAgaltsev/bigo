package survey

import (
	"github.com/RomanAgaltsev/bigo/internal/frontier"
	"github.com/RomanAgaltsev/bigo/internal/report"
)

// SmellFinding is one advisory finding that survived the contribution filters:
// first-party by module path, and hand-written.
//
// Target is stamped by Run, not by SummarizeSmells — a pure summarizer of one
// document has no idea which config entry produced it.
type SmellFinding struct {
	Target  string `json:"target,omitempty"`
	Rule    string `json:"rule"`
	Package string `json:"package"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// SmellTotals is the OUTWARD-facing half of a target's measurement, and the
// only part of this harness addressed to somebody else's maintainers.
//
// Everything else the survey computes — coverage, cause histograms, sole
// blockers, near-frontier distance — answers "what should bigo build next".
// These two fields answer "what could we tell a maintainer", which is a
// different question with a different audience.
type SmellTotals struct {
	// ByRule is the concentration measure, mirroring ByCause and ByDetail.
	ByRule map[string]int `json:"by_rule,omitempty"`
	// Findings is the triage queue, in report.Collect's file/line/rule order.
	Findings []SmellFinding `json:"findings,omitempty"`
}

// SummarizeSmells reduces one document's advisory findings to a target's smell
// totals. Pure, and a sibling of Summarize rather than a fifth return value
// from it: Summarize already returns four, and every existing call site would
// have churned for a value most of them ignore.
//
// isGen classifies a module-relative file as machine-generated; nil means
// nothing is. Generated findings are dropped entirely rather than counted
// separately, because unlike a coverage number a triage queue has no use for
// them — nobody hand-tunes generated code, so a finding there is not a
// contribution.
//
// The input order is preserved. report.Collect already sorts doc.Smells by
// file, line, then rule, and the probe's sampling rule is pinned to that order;
// re-sorting here would create a second source of truth for it.
func SummarizeSmells(doc report.Document, isGen func(string) bool) SmellTotals {
	if isGen == nil {
		isGen = func(string) bool { return false }
	}
	// Sized by rule cardinality: AllRules is a closed set of eight.
	out := SmellTotals{ByRule: make(map[string]int, 8)}
	for _, s := range doc.Smells {
		if !frontier.FirstParty(s.Package, doc.Module) {
			continue
		}
		if isGen(s.File) {
			continue
		}
		out.ByRule[s.Rule]++
		out.Findings = append(out.Findings, SmellFinding{
			Rule:    s.Rule,
			Package: s.Package,
			File:    s.File,
			Line:    s.Line,
			Message: s.Message,
		})
	}
	return out
}

// Sample draws the probe's triage sample: the first n findings in the input's
// order, admitting at most perRule from any one rule and at most perTarget from
// any one target. A cap of zero or less means that axis is uncapped; n of zero
// or less draws nothing.
//
// This is the anti-cherry-picking control, and it is code rather than judgment
// on purpose. The order and all three parameters are pre-registered in
// investigations/2026-08-18-contribution-lane-thresholds.md; implementing them
// as a pure function is what stops the registered rule and the executed rule
// from drifting apart.
//
// Both caps exist because finding volume is uneven along two INDEPENDENT axes,
// and an uncapped draw on either one measures something narrower than it claims:
//
//   - By rule: SM3 was predicted to dominate and did, 122 of 304.
//   - By target: the first scan's draw put 28 of 40 rows in caddy alone, because
//     it sits first in the config and had findings under most rules. That is
//     Amendment 1 to the thresholds — the per-rule cap guards one axis and did
//     nothing for the other.
//
// Returns a new slice; the input is not modified, because Run keeps the full
// list for the JSON artifact.
func Sample(findings []SmellFinding, n, perRule, perTarget int) []SmellFinding {
	if n <= 0 {
		return nil
	}
	out := make([]SmellFinding, 0, min(n, len(findings)))
	byRule := make(map[string]int, 8)
	byTarget := make(map[string]int, 16)
	for _, f := range findings {
		if len(out) == n {
			break
		}
		if perRule > 0 && byRule[f.Rule] >= perRule {
			continue
		}
		if perTarget > 0 && byTarget[f.Target] >= perTarget {
			continue
		}
		byRule[f.Rule]++
		byTarget[f.Target]++
		out = append(out, f)
	}
	return out
}
