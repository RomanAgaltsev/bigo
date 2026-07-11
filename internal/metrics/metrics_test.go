package metrics

import (
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
