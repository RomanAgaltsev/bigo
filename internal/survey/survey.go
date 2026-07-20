// Package survey measures bigo's REACH: how much of a real, external Go
// codebase it can bound, and what stops it.
//
// It is the third instrument, and the three answer different questions:
//
//	corpus/  (oracle)  did we get the literature's answer?   56 self-authored algorithms
//	metrics/           did anything drift?                   231 self-authored fixtures
//	survey/  (this)    how much real Go can we bound?        arbitrary external repos
//
// Unlike the other two, this is NOT a golden test and must never become one:
// its inputs are repositories that exist on one machine at whatever commit they
// happen to sit, it takes minutes, and its numbers SHOULD move when a target is
// updated. A stale-golden failure would be noise. `task survey` is a manual
// measurement whose committed output is a record of one run, stamped with the
// date and each target's commit.
package survey

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RomanAgaltsev/bigo/internal/report"
)

// TargetConfig names one repository to measure. Paths are machine-specific by
// nature; a missing one is a skip, not an error, so the harness runs anywhere
// with whatever subset exists.
type TargetConfig struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Config is survey/targets.json.
type Config struct {
	Targets []TargetConfig `json:"targets"`
}

// Totals are the headline counts for one target or for the aggregate.
//
// Functions counts FIRST-PARTY functions only; Seen counts every function in
// the document. Both are reported so the size of what was excluded is visible
// rather than assumed — see firstParty.
type Totals struct {
	Functions   int    `json:"functions"`
	Bounded     int    `json:"bounded"`
	Seen        int    `json:"functions_total_seen"`
	CoveragePct string `json:"coverage_pct"`
}

// Target is one measured repository.
type Target struct {
	Name    string `json:"name"`
	Module  string `json:"module,omitempty"`
	Commit  string `json:"commit,omitempty"`
	Skipped string `json:"skipped,omitempty"` // non-empty: why; other fields zero

	Totals
	ByCause  map[string]int `json:"by_cause,omitempty"`  // cause KIND
	ByDetail map[string]int `json:"by_detail,omitempty"` // cause DETAIL — the actionable one
}

// Report is the committed record of one survey run.
type Report struct {
	Generated   string   `json:"generated"`
	BigoVersion string   `json:"bigo_version"`
	Targets     []Target `json:"targets"`
	Aggregate   Totals   `json:"aggregate"`

	AggByCause  map[string]int `json:"aggregate_by_cause"`
	AggByDetail map[string]int `json:"aggregate_by_detail"`
}

// firstParty reports whether pkg belongs to module — the correctness crux of
// this harness.
//
// An ad-hoc survey run over prometheus counted `pb33f/libopenapi` symbols,
// which are somebody else's code; any coverage number computed over those
// measures the wrong thing. Dependencies reach the document because Collect
// loads with NeedDeps, so the filter is not optional.
//
// The test is exact, not heuristic: the package path must equal the module path
// or sit beneath it. Note the boundary — "example.com/m" must not match
// "example.com/mtools".
func firstParty(pkg, module string) bool {
	if module == "" {
		return true // no module recorded: cannot filter, count everything
	}
	return pkg == module || strings.HasPrefix(pkg, module+"/")
}

// pct renders a coverage percentage with one decimal. Zero functions yields
// "0.0" rather than dividing by zero — an empty or fully-filtered target is a
// legitimate outcome, not a crash.
func pct(bounded, total int) string {
	if total == 0 {
		return "0.0"
	}
	return fmt.Sprintf("%.1f", float64(bounded)*100/float64(total))
}

// Summarize reduces one document to a target's counts and histograms. Pure:
// this is the part worth testing, and it is tested on fixtures rather than on
// real repositories.
func Summarize(doc report.Document) (Totals, map[string]int, map[string]int) {
	// Sized by KEY CARDINALITY, not by the loop bound: byCause is keyed by
	// engine.CauseKind, a closed set of six, and byDetail by distinct cause
	// prose, which grows far more slowly than the function count. Hinting
	// len(doc.Functions) here would over-allocate by three orders of magnitude
	// on a large target.
	byCause := make(map[string]int, 8)
	byDetail := make(map[string]int, 256)
	t := Totals{Seen: len(doc.Functions)}
	for _, f := range doc.Functions {
		if !firstParty(f.Package, doc.Module) {
			continue
		}
		t.Functions++
		if !f.Time.Top {
			t.Bounded++
			continue
		}
		for _, c := range f.Causes {
			byCause[c.Kind]++
			byDetail[c.Detail]++
		}
	}
	t.CoveragePct = pct(t.Bounded, t.Functions)
	return t, byCause, byDetail
}

// commitOf returns the target's short HEAD, or "" when it cannot be read. A
// number without a commit is not comparable across runs, but failing the whole
// target over a missing SHA would be worse than recording it as unknown.
func commitOf(path string) string {
	out, err := exec.Command("git", "-C", path, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Run measures every configured target. It never returns an error for a target:
// an unanalyzable repository is reported as skipped WITH ITS REASON, because a
// silently dropped target would quietly bias every aggregate.
func Run(cfg Config, version string, progress func(string, ...any)) Report {
	if progress == nil {
		progress = func(string, ...any) {}
	}
	r := Report{
		Generated:   time.Now().UTC().Format("2006-01-02"),
		BigoVersion: version,
		AggByCause:  map[string]int{},
		AggByDetail: map[string]int{},
	}
	for _, tc := range cfg.Targets {
		progress("survey: %s", tc.Name)
		t := Target{Name: tc.Name}
		if _, err := os.Stat(filepath.Clean(tc.Path)); err != nil {
			t.Skipped = "path not present on this machine"
			r.Targets = append(r.Targets, t)
			continue
		}
		doc, err := report.Collect(tc.Path, nil, report.Options{Version: version})
		if err != nil {
			t.Skipped = "analysis failed: " + firstLine(err.Error())
			r.Targets = append(r.Targets, t)
			continue
		}
		totals, byCause, byDetail := Summarize(doc)
		t.Module, t.Commit, t.Totals = doc.Module, commitOf(tc.Path), totals
		t.ByCause, t.ByDetail = byCause, byDetail
		for k, v := range byCause {
			r.AggByCause[k] += v
		}
		for k, v := range byDetail {
			r.AggByDetail[k] += v
		}
		r.Aggregate.Functions += totals.Functions
		r.Aggregate.Bounded += totals.Bounded
		r.Aggregate.Seen += totals.Seen
		r.Targets = append(r.Targets, t)
	}
	r.Aggregate.CoveragePct = pct(r.Aggregate.Bounded, r.Aggregate.Functions)
	return r
}

// firstLine keeps a skip reason to one line: go/packages errors can run to
// hundreds of lines and would swamp the rendered table.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// LoadConfig reads survey/targets.json.
func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path) //nolint:gosec // operator-supplied config path
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}
