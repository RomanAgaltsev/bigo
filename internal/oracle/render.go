package oracle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// JSON renders the committed golden: indented, sorted map keys, trailing newline.
func (r Report) JSON() []byte {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err) // no unmarshalable types in Report
	}
	return append(b, '\n')
}

// Markdown renders CORPUS.md. GENERATED output — regenerate via task corpus.
func (r Report) Markdown() []byte {
	var b bytes.Buffer
	b.WriteString("# bigo canonical algorithm corpus — oracle golden\n\n")
	b.WriteString("GENERATED — do not edit; regenerate with `task corpus`.\n\n")
	b.WriteString("Literature-pinned worst-case bounds vs unaided inference. `exact` = inference\n")
	b.WriteString("matches the literature; `loose` = sound but imprecise (a graduation target);\n")
	b.WriteString("`top` = unverifiable (the annotate-or-trust evidence rows). A `wrong` never\n")
	b.WriteString("appears here: it fails the build. Algorithms considered and kept out are in\n")
	b.WriteString("[EXCLUSIONS.md](EXCLUSIONS.md). **This is not a coverage metric** — read\n")
	b.WriteString("composition, not a percentage.\n\n")

	writeCounts := func(title string, m map[string]int) {
		fmt.Fprintf(&b, "## %s\n\n| Status | Count |\n|---|---|\n", title)
		for _, s := range []string{"exact", "loose", "top"} {
			fmt.Fprintf(&b, "| %s | %d |\n", s, m[s])
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "**Entries: %d**\n\n", r.Total)
	writeCounts("Time statuses", r.TimeByStatus)
	writeCounts("Space statuses (pinned entries only)", r.SpaceByStatus)

	b.WriteString("## Per family\n\n| Family | Entries |\n|---|---|\n")
	fams := make([]string, 0, len(r.PerFamily))
	for f := range r.PerFamily {
		fams = append(fams, f)
	}
	sort.Strings(fams)
	for _, f := range fams {
		fmt.Fprintf(&b, "| %s | %d |\n", f, r.PerFamily[f])
	}

	b.WriteString("\n## Entries\n\n")
	b.WriteString("| Function | Time pin | Time got | Status | Space pin | Space got | Status | Cause | Source |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|\n")
	for _, e := range r.Entries {
		fmt.Fprintf(&b, "| %s.%s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			e.Pkg, e.Func, e.TimePin, e.TimeGot, e.TimeStatus,
			e.SpacePin, e.SpaceGot, e.SpaceStatus, e.Cause, e.Source)
	}
	return b.Bytes()
}
