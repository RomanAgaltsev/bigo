package survey

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "run the survey and rewrite survey/survey.json and survey/SURVEY.md")

var whatifFile = flag.String("whatif", "", "candidates file; runs the what-if harness and writes survey/WHATIF.md and survey/whatif.json")

// TestSurvey is the harness's entry point, and it is deliberately NOT a golden
// test — it asserts nothing and SKIPS unless -update is passed.
//
// The other two instruments (corpus, metrics) are golden tests gated in CI.
// This one cannot be: its inputs are repositories that exist on one machine at
// whatever commit they happen to sit, it takes minutes on large targets, and
// its numbers SHOULD move when a target is updated — so a stale-golden failure
// would be noise, not signal. Skipping by default is what keeps `go test ./...`
// honest on any machine, CI included.
//
// Run it with: task survey
func TestSurvey(t *testing.T) {
	if !*update {
		t.Skip("survey is a manual measurement; run `task survey`")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(root, "survey", "targets.json")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("read %s: %v", cfgPath, err)
	}
	if len(cfg.Targets) == 0 {
		t.Fatalf("%s lists no targets", cfgPath)
	}

	r := Run(cfg, version(t, root), func(f string, a ...any) { t.Logf(f, a...) })

	outDir := filepath.Join(root, "survey")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "survey.json"), r.JSON(), 0o644); err != nil { //nolint:gosec // generated record, not a secret
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "SURVEY.md"), r.Markdown(), 0o644); err != nil { //nolint:gosec // generated record, not a secret
		t.Fatal(err)
	}
	t.Logf("survey: %d of %d first-party functions bounded (%s%%) across %d targets",
		r.Aggregate.Bounded, r.Aggregate.Functions, r.Aggregate.CoveragePct, len(r.Targets))
}

// TestWhatIf is the what-if harness's entry point, TestSurvey's sibling: it
// asserts nothing and SKIPS unless -whatif names a candidates file. Its
// committed output is the record of one run — deliberately NOT a golden, for
// exactly TestSurvey's reasons.
//
// Run it with: task whatif -- candidates.json
func TestWhatIf(t *testing.T) {
	if *whatifFile == "" {
		t.Skip("what-if harness runs only with -whatif <candidates.json> (task whatif)")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(root, "survey", "targets.json")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("read %s: %v", cfgPath, err)
	}
	// `go test ./internal/survey/` runs with the PACKAGE as the working
	// directory, so a path typed relative to the repo root (the only place
	// `task whatif` is ever run from) does not resolve as given. Try it
	// verbatim first, then relative to the root.
	candPath := *whatifFile
	if !filepath.IsAbs(candPath) {
		if _, statErr := os.Stat(candPath); statErr != nil {
			candPath = filepath.Join(root, candPath)
		}
	}
	wc, sets, err := LoadWhatIfConfig(candPath)
	if err != nil {
		t.Fatal(err)
	}
	rep := RunWhatIf(cfg, wc, sets, version(t, root), func(f string, a ...any) { t.Logf(f, a...) })
	rep.Candidates = filepath.Base(candPath)
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(root, "survey")
	if err := os.WriteFile(filepath.Join(outDir, "whatif.json"), append(data, '\n'), 0o644); err != nil { //nolint:gosec // generated record, not a secret
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "WHATIF.md"), []byte(RenderWhatIf(rep)), 0o644); err != nil { //nolint:gosec // generated record, not a secret
		t.Fatal(err)
	}
	for _, res := range rep.Results {
		if res.Blocked != "" {
			t.Logf("whatif: %s BLOCKED: %s", res.Name, res.Blocked)
			continue
		}
		t.Logf("whatif: %s graduated %d (%d hand-written, %s)", res.Name, res.Graduated, res.GraduatedHand, res.DeltaPP)
	}
}

// version reads the LAST RELEASED version from .release-please-manifest.json,
// which is exactly what that file holds: the bump lands in the release PR, so
// a survey run made on a feature branch — the normal case, since a lane
// regenerates the survey before it merges — stamps the PREVIOUS tag. v1.40.0's
// run is the worked example: its numbers are v1.40.0's and its stamp reads
// 1.39.1.
//
// Documented rather than "fixed", because the alternative is stamping a
// version that does not exist yet. Compare runs by the per-target commit.
func version(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, ".release-please-manifest.json"))
	if err != nil {
		return "dev"
	}
	// The manifest is {".":"1.33.1"}; a tiny scan beats pulling in a decoder
	// for one value in a test helper.
	s := string(b)
	i, j := -1, -1
	for k := 0; k < len(s); k++ {
		if s[k] == '"' {
			if i == -1 && k > 0 && s[k-1] == ':' {
				i = k + 1
			} else if i != -1 {
				j = k
				break
			}
		}
	}
	if i == -1 || j == -1 || j <= i {
		return "dev"
	}
	return s[i:j]
}
