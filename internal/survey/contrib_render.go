package survey

import (
	"bytes"
	"fmt"
)

// SampleSize and SamplePerRule are the probe's pre-registered sampling
// parameters, from investigations/2026-08-18-contribution-lane-thresholds.md.
// They are constants here so the executed draw and the registered rule are the
// same number in one place.
const (
	SampleSize    = 40
	SamplePerRule = 8
)

// RenderContribQueue renders the triage queue: the per-rule counts, the skipped
// targets, and the SAMPLE that will actually be triaged.
//
// The sample rather than the full finding list, for two reasons: it is what gets
// judged, and committing it makes the draw auditable — a reader can check the
// executed sample against the rule registered before the scan. The full list is
// in contrib-queue.json.
func RenderContribQueue(r Report, sample []SmellFinding) string {
	var b bytes.Buffer
	b.WriteString("# bigo contribution lane — triage queue\n\n")
	b.WriteString("GENERATED — do not edit; regenerate with `task contrib-scan`.\n\n")
	b.WriteString("**A MANUAL measurement, not a golden test.** No test asserts its contents and\n")
	b.WriteString("CI never runs it. Targets are repositories on one machine at whatever commit\n")
	b.WriteString("they happen to sit; compare runs only via the per-target commit below.\n\n")
	fmt.Fprintf(&b, "Run %s with bigo %s.\n\n", r.Generated, r.BigoVersion)

	fmt.Fprintf(&b, "**Sample: %d findings, at most %d per rule**, drawn in target order then "+
		"file, line, rule. The rule was registered in "+
		"`docs/bigo/investigations/2026-08-18-contribution-lane-thresholds.md` before this "+
		"scan ran, and is implemented in `survey.Sample` so the two cannot drift.\n\n",
		SampleSize, SamplePerRule)

	b.WriteString("Population: first-party, hand-written. Generated code is excluded — nobody\n")
	b.WriteString("hand-tunes it, so a finding there is not a contribution.\n\n")

	b.WriteString("## Targets scanned\n\n| Target | Module | Commit | Findings |\n|---|---|---|---|\n")
	for _, t := range r.Targets {
		if t.Skipped != "" {
			fmt.Fprintf(&b, "| %s | — | — | skipped: %s |\n", t.Name, escapePipes(t.Skipped))
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %d |\n",
			t.Name, t.Module, t.Commit, len(t.Smells.Findings))
	}

	if len(r.AggSmellsByRule) > 0 {
		b.WriteString("\n## Findings by rule (whole population, not the sample)\n\n")
		b.WriteString("| Rule | Findings |\n|---|---|\n")
		for _, p := range ranked(r.AggSmellsByRule) {
			fmt.Fprintf(&b, "| %s | %d |\n", p.Key, p.Count)
		}
	}

	b.WriteString("\n## The sample\n\n")
	if len(sample) == 0 {
		b.WriteString("**No findings drawn.** Either no target was present on this machine or the\n")
		b.WriteString("population is empty — both are Stage 2 outcomes and must be reported as\n")
		b.WriteString("such, not as a clean run.\n")
		return b.String()
	}
	b.WriteString("Verdict column is filled by hand during triage: `sendable`, `not-sendable`,\n")
	b.WriteString("or `FP`. Read the target's own `.golangci.yml` before judging anything\n")
	b.WriteString("sendable — a finding its CI already declines to enforce is not-sendable\n")
	b.WriteString("however correct it is.\n\n")
	b.WriteString("| # | Target | Rule | Position | Message | Verdict |\n|---|---|---|---|---|---|\n")
	for i, f := range sample {
		fmt.Fprintf(&b, "| %d | %s | %s | `%s:%d` | %s | |\n",
			i+1, f.Target, f.Rule, f.File, f.Line, escapePipes(f.Message))
	}
	return b.String()
}
