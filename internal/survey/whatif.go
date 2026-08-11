package survey

// What-if: the fourth instrument. corpus answers "is it correct", metrics
// "did it drift", survey "how far does it reach" — whatif answers "what would
// ACTUALLY graduate if this assumption held", exactly, by running the real
// engine per candidate assumption set over the survey targets. Never a graph
// simulation: the fmt probe measured the report-graph prediction at 348
// against the engine's 298 (spec §1).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/RomanAgaltsev/bigo/internal/assume"
	"github.com/RomanAgaltsev/bigo/internal/frontier"
	"github.com/RomanAgaltsev/bigo/internal/report"
)

// Candidate names one assumption set to measure.
type Candidate struct {
	Name string `json:"name"`
	File string `json:"file"` // assumption file, relative to the candidates file's directory
}

// WhatIfConfig is the candidates file.
type WhatIfConfig struct {
	Candidates []Candidate `json:"candidates"`
}

// TargetDelta is one candidate's exact yield on one target.
type TargetDelta struct {
	Target        string `json:"target"`
	Graduated     int    `json:"graduated"`
	GraduatedHand int    `json:"graduated_hand"`
}

// CandidateResult is one candidate's yield over every measured target. A
// non-empty Blocked means the integrity gate fired somewhere and the counts
// are WITHHELD (zeroed) — a blocked result that still showed numbers would be
// the exact failure mode the gate exists to prevent.
type CandidateResult struct {
	Name          string        `json:"name"`
	Graduated     int           `json:"graduated"`
	GraduatedHand int           `json:"graduated_hand"`
	DeltaPP       string        `json:"coverage_delta_pp"`
	PerTarget     []TargetDelta `json:"per_target"`
	Warnings      []string      `json:"warnings,omitempty"`
	Blocked       string        `json:"blocked,omitempty"`
}

// WhatIfReport is the committed record of one what-if run.
type WhatIfReport struct {
	Generated   string `json:"generated"`
	BigoVersion string `json:"bigo_version"`

	// Candidates names the candidates file this run measured. WHATIF.md is a
	// single-slot record like SURVEY.md — the next run overwrites it — so
	// without this a committed record cannot be told apart from the run that
	// replaced it. Learned by losing one.
	Candidates string `json:"candidates_file,omitempty"`

	BaselineFunctions int               `json:"baseline_functions"`
	BaselineBounded   int               `json:"baseline_bounded"`
	Results           []CandidateResult `json:"results"`
}

// LoadWhatIfConfig reads the candidates file and eagerly loads every
// referenced assumption set — a broken candidate file aborts the whole run
// before any analysis (hard errors, spec §3).
func LoadWhatIfConfig(path string) (WhatIfConfig, map[string]*assume.Set, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return WhatIfConfig{}, nil, err
	}
	var wc WhatIfConfig
	if err := json.Unmarshal(data, &wc); err != nil {
		return WhatIfConfig{}, nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(wc.Candidates) == 0 {
		return WhatIfConfig{}, nil, fmt.Errorf("%s: no candidates", path)
	}
	dir := filepath.Dir(path)
	sets := map[string]*assume.Set{}
	seen := map[string]bool{}
	for _, c := range wc.Candidates {
		if c.Name == "" || seen[c.Name] {
			return WhatIfConfig{}, nil, fmt.Errorf("%s: candidate names must be unique and non-empty", path)
		}
		seen[c.Name] = true
		s, err := assume.Load(filepath.Join(dir, c.File))
		if err != nil {
			return WhatIfConfig{}, nil, fmt.Errorf("candidate %s: %w", c.Name, err)
		}
		sets[c.Name] = s
	}
	return wc, sets, nil
}

// whatifKey identifies a function across two documents of the same load.
// Package+File+Line, never Package+Func — a package may declare several
// init()s (the bigo diff F1 lesson).
func whatifKey(f report.Function) string {
	return fmt.Sprintf("%s\x00%s\x00%d", f.Package, f.File, f.Line)
}

// compareDocs counts the candidate document's genuine graduations over the
// baseline: first-party ⊤→bounded transitions carrying assumption taint.
// Anything else that moved blocks the result — the reconciliation gate the
// probes taught (a non-zero unattributed count invalidated two probe runs
// AFTER their headline numbers looked plausible).
func compareDocs(base, cand report.Document, isGen func(string) bool) (grad, gradHand int, blocked string) {
	bm := make(map[string]report.Function, len(base.Functions))
	for _, f := range base.Functions {
		if frontier.FirstParty(f.Package, base.Module) {
			bm[whatifKey(f)] = f
		}
	}
	seen, unattributed := 0, 0
	for _, f := range cand.Functions {
		if !frontier.FirstParty(f.Package, cand.Module) {
			continue
		}
		bf, ok := bm[whatifKey(f)]
		if !ok {
			return 0, 0, "population mismatch: candidate run contains functions the baseline lacks"
		}
		seen++
		switch {
		case !bf.Time.Top && f.Time.Top:
			return 0, 0, fmt.Sprintf("bound lost under assumptions: %s.%s (%s:%d)", f.Package, f.Func, f.File, f.Line)
		case bf.Time.Top && !f.Time.Top:
			switch f.Provenance {
			case report.ProvenanceTainted:
				grad++
				if !isGen(f.File) {
					gradHand++
				}
			case report.ProvenanceAssumed:
				// The target itself: assuming a bound is not discovering one.
			default:
				unattributed++
			}
		}
	}
	if seen != len(bm) {
		return 0, 0, "population mismatch: baseline contains functions the candidate run lacks"
	}
	if unattributed > 0 {
		return 0, 0, fmt.Sprintf("%d unattributed graduations (bounded without assumption taint) — run is not comparable", unattributed)
	}
	return grad, gradHand, ""
}

// RunWhatIf measures every candidate against every available target. SSA is
// built once per TARGET and reused for the baseline plus every candidate —
// the honest price of exactness is one analysis pass per (target, candidate),
// never a graph shortcut (spec §5).
//
// Key absence is judged over the POPULATION, not per module: a key missing
// from one module contributes zero to it by arithmetic, so it is warned about;
// a key missing from every measured module never contributed at all, so it
// blocks its candidate. Any other per-target failure blocks too, and the
// render always makes a block visible.
func RunWhatIf(cfg Config, wc WhatIfConfig, sets map[string]*assume.Set, version string, progress func(string, ...any)) WhatIfReport {
	if progress == nil {
		progress = func(string, ...any) {}
	}
	r := WhatIfReport{Generated: time.Now().UTC().Format("2006-01-02"), BigoVersion: version}
	results := make(map[string]*CandidateResult, len(wc.Candidates))
	unmatchedIn := make(map[string]map[string]int, len(wc.Candidates))
	for _, c := range wc.Candidates {
		results[c.Name] = &CandidateResult{Name: c.Name}
		unmatchedIn[c.Name] = make(map[string]int, len(sets[c.Name].Entries()))
	}
	warnSeen := make(map[string]map[string]bool, len(wc.Candidates))
	// unmatchedIn[candidate][key] counts targets where the key matched nothing
	// (allocated per candidate above). A key unmatched in EVERY measured target
	// is a typo or an inexpressible key and blocks its candidate; a key absent
	// from some targets only is reported and tolerated (assume.ValidateMatched
	// documents why).
	measuredTargets := 0
	for _, tc := range cfg.Targets {
		if _, err := os.Stat(filepath.Clean(tc.Path)); err != nil {
			progress("whatif: %s skipped (path not present)", tc.Name)
			continue
		}
		progress("whatif: loading %s", tc.Name)
		l, err := report.LoadModule(tc.Path, nil)
		if err != nil {
			progress("whatif: %s skipped (load failed: %v)", tc.Name, err)
			continue
		}
		base, err := l.Document(report.Options{Version: version})
		if err != nil {
			progress("whatif: %s skipped (baseline failed: %v)", tc.Name, err)
			continue
		}
		measuredTargets++
		isGen := newGeneratedDetector(tc.Path).isGenerated
		for _, f := range base.Functions {
			if frontier.FirstParty(f.Package, base.Module) {
				r.BaselineFunctions++
				if !f.Time.Top {
					r.BaselineBounded++
				}
			}
		}
		for _, c := range wc.Candidates {
			res := results[c.Name]
			if res.Blocked != "" {
				continue // one blocked target invalidates the candidate's totals
			}
			progress("whatif: %s × %s", tc.Name, c.Name)
			var warns []string
			cand, err := l.Document(report.Options{Version: version, Assume: sets[c.Name],
				Warn: func(w string) { warns = append(warns, w) },
				AssumeUnmatchedKeys: func(keys []string) {
					for _, k := range keys {
						unmatchedIn[c.Name][k]++
					}
				}})
			if err != nil {
				res.Blocked = fmt.Sprintf("%s: %v", tc.Name, err)
				res.Graduated, res.GraduatedHand, res.PerTarget = 0, 0, nil
				continue
			}
			grad, hand, blocked := compareDocs(base, cand, isGen)
			if blocked != "" {
				res.Blocked = tc.Name + ": " + blocked
				res.Graduated, res.GraduatedHand, res.PerTarget = 0, 0, nil
				continue
			}
			res.Graduated += grad
			res.GraduatedHand += hand
			res.PerTarget = append(res.PerTarget, TargetDelta{Target: tc.Name, Graduated: grad, GraduatedHand: hand})
			if warnSeen[c.Name] == nil {
				warnSeen[c.Name] = make(map[string]bool, len(warns))
			}
			for _, w := range warns {
				if !warnSeen[c.Name][w] {
					warnSeen[c.Name][w] = true
					res.Warnings = append(res.Warnings, w)
				}
			}
		}
	}
	for _, c := range wc.Candidates {
		res := results[c.Name]
		// A key that matched nothing in ANY target never contributed to this
		// candidate's number, so that number is not what it appears to be.
		// This is the sweep's replacement for the single-module hard error,
		// and it is the case the hard error actually exists to catch.
		// Both walks go over the candidate's entries in FILE order rather than
		// over the map: deterministic without a sort, and a key list reads back
		// in the order its author wrote it.
		um := unmatchedIn[c.Name]
		if res.Blocked == "" && measuredTargets > 0 {
			var nowhere []string
			for _, e := range sets[c.Name].Entries() {
				if um[e.Key] == measuredTargets {
					nowhere = append(nowhere, e.Key)
				}
			}
			if len(nowhere) > 0 {
				res.Blocked = "keys matched no function in any of the " +
					strconv.Itoa(measuredTargets) + " measured targets: " + strings.Join(nowhere, ", ")
				res.Graduated, res.GraduatedHand, res.PerTarget = 0, 0, nil
			}
		}
		// Keys absent from some targets but present in others are reported,
		// never silent: the count is real but the population it came from is
		// smaller than the candidate file implies.
		if res.Blocked == "" {
			for _, e := range sets[c.Name].Entries() {
				if n := um[e.Key]; n > 0 {
					res.Warnings = append(res.Warnings,
						fmt.Sprintf("%s (absent from %d of %d targets)", e.Key, n, measuredTargets))
				}
			}
		}
		if res.Blocked == "" && r.BaselineFunctions > 0 {
			res.DeltaPP = fmt.Sprintf("+%.2fpp", 100*float64(res.Graduated)/float64(r.BaselineFunctions))
		}
		r.Results = append(r.Results, *res)
	}
	return r
}
