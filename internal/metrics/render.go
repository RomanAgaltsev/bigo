package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// JSON renders the report as the committed golden: indented, sorted (Go's
// encoding/json sorts map keys), trailing newline.
func (r Report) JSON() []byte {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err) // no unmarshalable types in Report
	}
	return append(b, '\n')
}

// Markdown renders METRICS.md. GENERATED output — regenerate via task metrics.
func (r Report) Markdown() []byte {
	var b bytes.Buffer
	b.WriteString("# bigo corpus coverage\n\n")
	b.WriteString("GENERATED — do not edit; regenerate with `task metrics`.\n\n")
	fmt.Fprintf(&b, "**Coverage: %s%%** — %d of %d corpus functions bounded.\n\n", r.CoveragePct, r.Bounded, r.Total)

	b.WriteString("## Per package\n\n| Package | Functions | Bounded | Unverifiable |\n|---|---|---|---|\n")
	pkgs := make([]string, 0, len(r.PerPackage))
	for p := range r.PerPackage {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	for _, p := range pkgs {
		c := r.PerPackage[p]
		fmt.Fprintf(&b, "| %s | %d | %d | %d |\n", p, c.Total, c.Bounded, c.Total-c.Bounded)
	}

	b.WriteString("\n## Unverifiable by cause\n\n| Cause | Count |\n|---|---|\n")
	causes := make([]string, 0, len(r.ByCause))
	for c := range r.ByCause {
		causes = append(causes, c)
	}
	sort.Strings(causes)
	for _, c := range causes {
		fmt.Fprintf(&b, "| %s | %d |\n", c, r.ByCause[c])
	}
	b.WriteString("\nThe cause histogram is the Phase-2 prioritization signal: the biggest\nbucket is the next feature.\n")

	if len(r.Smells) > 0 {
		b.WriteString("\n## Smell fires (drift alarm)\n\n")
		b.WriteString("Not coverage. A change in a rule's corpus fire count is a behavior change and must be deliberate.\n\n")
		b.WriteString("| Rule | Corpus fires |\n|---|---|\n")
		var rules []string
		for rule := range r.Smells {
			rules = append(rules, rule)
		}
		sort.Strings(rules)
		for _, rule := range rules {
			fmt.Fprintf(&b, "| %s | %d |\n", rule, r.Smells[rule])
		}
	}
	return b.Bytes()
}
