package survey

import (
	"fmt"
	"strings"
)

// RenderWhatIf renders WHATIF.md — one committed record per run, SURVEY.md's
// sibling. Blocked candidates render their reason and NO numbers: a blocked
// result that still showed counts would be the exact failure mode the
// integrity gate exists to prevent.
func RenderWhatIf(r WhatIfReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# bigo what-if — %s (bigo %s)\n\n", r.Generated, r.BigoVersion)
	if r.Candidates != "" {
		fmt.Fprintf(&b, "Candidates: `%s`. This file is a SINGLE-SLOT record — the next run\noverwrites it, so a committed record is only the last run measured.\n\n", r.Candidates)
	}
	fmt.Fprintf(&b, "Baseline: %d first-party functions, %d bounded.\n\n", r.BaselineFunctions, r.BaselineBounded)
	b.WriteString("Graduations are EXACT engine results under each candidate's assumption\n")
	b.WriteString("set — tainted ⊤→bounded transitions only; assumed targets excluded.\n")
	b.WriteString("A graduation says the propagation clears, NOT that the assumed bound is\n")
	b.WriteString("truthful — that half needs a probe or an implementation argument (spec §6).\n\n")
	b.WriteString("| candidate | graduated | hand-written | Δ coverage |\n|---|---|---|---|\n")
	for _, res := range r.Results {
		if res.Blocked != "" {
			fmt.Fprintf(&b, "| %s | BLOCKED | — | — |\n", res.Name)
			continue
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %s |\n", res.Name, res.Graduated, res.GraduatedHand, res.DeltaPP)
	}
	for _, res := range r.Results {
		if res.Blocked == "" && len(res.PerTarget) == 0 && len(res.Warnings) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n\n", res.Name)
		if res.Blocked != "" {
			fmt.Fprintf(&b, "**BLOCKED — result withheld:** %s\n", res.Blocked)
			continue
		}
		for _, td := range res.PerTarget {
			fmt.Fprintf(&b, "- %s: %d graduated (%d hand-written)\n", td.Target, td.Graduated, td.GraduatedHand)
		}
		for _, w := range res.Warnings {
			fmt.Fprintf(&b, "- warning: %s\n", w)
		}
	}
	return b.String()
}
