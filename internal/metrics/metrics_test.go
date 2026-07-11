package metrics

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestCollectFixture(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "src"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.Total != 5 || r.Bounded != 2 {
		t.Errorf("Total/Bounded = %d/%d, want 5/2", r.Total, r.Bounded)
	}
	want := map[string]int{"call": 1, "loop": 1, "nobody": 1}
	for k, n := range want {
		if r.ByCause[k] != n {
			t.Errorf("ByCause[%q] = %d, want %d", k, r.ByCause[k], n)
		}
	}
	if r.CoveragePct != "40.0" {
		t.Errorf("CoveragePct = %q, want 40.0", r.CoveragePct)
	}
	if pc := r.PerPackage["fixture"]; pc.Total != 5 || pc.Bounded != 2 {
		t.Errorf("PerPackage[fixture] = %+v, want {5 2}", pc)
	}
}

var update = flag.Bool("update", false, "rewrite metrics/metrics.json and metrics/METRICS.md")

func TestGolden(t *testing.T) {
	srcRoot, err := filepath.Abs(filepath.Join("..", "..", "analyzer", "testdata", "src"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := Collect(srcRoot)
	if err != nil {
		t.Fatal(err)
	}
	jsonPath, err := filepath.Abs(filepath.Join("..", "..", "metrics", "metrics.json"))
	if err != nil {
		t.Fatal(err)
	}
	mdPath, err := filepath.Abs(filepath.Join("..", "..", "metrics", "METRICS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(jsonPath, r.JSON(), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(mdPath, r.Markdown(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	for _, tc := range []struct {
		path string
		got  []byte
	}{{jsonPath, r.JSON()}, {mdPath, r.Markdown()}} {
		want, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatalf("read golden (run `task metrics` to create it): %v", err)
		}
		if !bytes.Equal(normalizeEOL(tc.got), normalizeEOL(want)) {
			t.Errorf("%s is stale — the engine's coverage changed; run `task metrics` and commit the diff", filepath.Base(tc.path))
		}
	}
}

// normalizeEOL strips CR so the comparison survives core.autocrlf checkouts.
func normalizeEOL(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
