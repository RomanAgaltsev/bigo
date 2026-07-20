package survey

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "run the survey and rewrite survey/survey.json and survey/SURVEY.md")

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

// version reads the released version from .release-please-manifest.json so the
// record is stamped with something meaningful rather than "dev".
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
