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

// Markdown renders CORPUS.md or KATA.md. GENERATED output — regenerate via
// task corpus / task kata-corpus.
//
// The preamble is chosen by r.Corpus rather than shared, because the two
// corpora pin different KINDS of claim and a reader who mistakes one for the
// other draws the wrong conclusion from a `loose` row: against literature it is
// a graduation target, against a human's average-case claim it is bigo and the
// author answering different questions.
func (r Report) Markdown() []byte {
	var b bytes.Buffer
	if r.Corpus == KataCorpus {
		b.WriteString("# bigo kata corpus — oracle golden\n\n")
		b.WriteString("GENERATED — do not edit; regenerate with `task kata-corpus`.\n\n")
		b.WriteString("Human-claimed bounds on real submitted solutions, scored under the KATA cost\n")
		b.WriteString("model. `exact` = inference matches the author's claim; `loose` = sound but\n")
		b.WriteString("imprecise, which here often means bigo answered the worst case where the\n")
		b.WriteString("author claimed the average one; `top` = unverifiable. A `wrong` never appears:\n")
		b.WriteString("it fails the build. What was reduced away, and why, is in\n")
		b.WriteString("[kata/README.md](kata/README.md). **This is not a coverage metric** — read\n")
		b.WriteString("composition, not a percentage.\n\n")
		b.WriteString("Sibling: [CORPUS.md](CORPUS.md) pins literature bounds under the default cost\n")
		b.WriteString("model. The two are never summed.\n\n")
	} else {
		b.WriteString("# bigo canonical algorithm corpus — oracle golden\n\n")
		b.WriteString("GENERATED — do not edit; regenerate with `task corpus`.\n\n")
		b.WriteString("Literature-pinned worst-case bounds vs unaided inference. `exact` = inference\n")
		b.WriteString("matches the literature; `loose` = sound but imprecise (a graduation target);\n")
		b.WriteString("`top` = unverifiable (the annotate-or-trust evidence rows). A `wrong` never\n")
		b.WriteString("appears here: it fails the build. Algorithms considered and kept out are in\n")
		b.WriteString("[EXCLUSIONS.md](EXCLUSIONS.md). **This is not a coverage metric** — read\n")
		b.WriteString("composition, not a percentage.\n\n")
		b.WriteString("Sibling: [KATA.md](KATA.md) pins human-claimed bounds on real kata solutions\n")
		b.WriteString("under the kata cost model. The two are never summed.\n\n")
	}

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
			e.Pkg, e.Func, unpinned(e.TimePin), unpinned(e.TimeGot), unpinned(e.TimeStatus),
			unpinned(e.SpacePin), unpinned(e.SpaceGot), unpinned(e.SpaceStatus), e.Cause, e.Source)
	}
	return b.Bytes()
}

// unpinned renders an axis that carries no pin. Either axis may be absent —
// space always could be, time since 2026-08-29 — and an empty table cell reads
// as missing data rather than as a deliberate refusal to state a claim the pin
// grammar cannot express. The dash says the row is partial on purpose.
func unpinned(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
