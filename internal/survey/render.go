package survey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
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

	b.WriteString("## Per target\n\n")
	b.WriteString("| Target | Module | Commit | Functions | Bounded | Coverage |\n|---|---|---|---|---|---|\n")
	for _, t := range r.Targets {
		if t.Skipped != "" {
			fmt.Fprintf(&b, "| %s | — | — | — | — | skipped: %s |\n", t.Name, t.Skipped)
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %d | %s%% |\n",
			t.Name, t.Module, t.Commit, t.Functions, t.Bounded, t.CoveragePct)
	}

	b.WriteString("\n## Unverifiable by cause kind\n\n| Cause | Count |\n|---|---|\n")
	for _, p := range ranked(r.AggByCause) {
		fmt.Fprintf(&b, "| %s | %d |\n", p.Key, p.Count)
	}
	b.WriteString("\nCompare with `corpus/CORPUS.md`: the canonical corpus and real code do not\nagree on this ranking, and real code is the one that reflects adoption.\n")

	fmt.Fprintf(&b, "\n## Top %d blockers by detail\n\n", detailTop)
	b.WriteString("**This table is the deliverable.** It ranks work by what actually stops bigo\n")
	b.WriteString("on real code, rather than by what the self-authored corpus happens to contain.\n\n")
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
