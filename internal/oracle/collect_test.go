package oracle

import (
	"path/filepath"
	"testing"
)

func TestCollectGoodFixture(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "good", "src"))
	if err != nil {
		t.Fatal(err)
	}
	r, wrongs, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrongs) != 0 {
		t.Fatalf("wrongs = %+v, want none", wrongs)
	}
	if r.Total != 2 || len(r.Entries) != 2 {
		t.Fatalf("Total = %d, entries = %d, want 2/2", r.Total, len(r.Entries))
	}
	// Entries are sorted by (Pkg, Func): LinearSum before Opaque.
	ls := r.Entries[0]
	if ls.Func != "LinearSum" || ls.TimeStatus != "exact" || ls.SpaceStatus != "exact" {
		t.Errorf("LinearSum entry = %+v", ls)
	}
	if ls.TimeGot != "O(len(s))" {
		t.Errorf("LinearSum TimeGot = %q", ls.TimeGot)
	}
	op := r.Entries[1]
	if op.Func != "Opaque" || op.TimeStatus != "top" || op.Cause != "call" {
		t.Errorf("Opaque entry = %+v", op)
	}
	if op.SpacePin != "" || op.SpaceStatus != "" {
		t.Errorf("Opaque space should be unpinned, got %+v", op)
	}
	if r.TimeByStatus["exact"] != 1 || r.TimeByStatus["top"] != 1 {
		t.Errorf("TimeByStatus = %v", r.TimeByStatus)
	}
	if r.PerFamily["goodpkg"] != 2 {
		t.Errorf("PerFamily = %v", r.PerFamily)
	}
}

func TestCollectWrongFires(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "wrong", "src"))
	if err != nil {
		t.Fatal(err)
	}
	_, wrongs, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrongs) != 1 {
		t.Fatalf("the alarm did not ring: wrongs = %+v, want exactly 1", wrongs)
	}
	w := wrongs[0]
	if w.Func != "TooGood" || w.Dim != "time" {
		t.Errorf("wrong = %+v", w)
	}
}

func TestRenderDeterministic(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("testdata", "good", "src"))
	r1, _, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	r2, _, _ := Collect(root)
	if string(r1.JSON()) != string(r2.JSON()) || string(r1.Markdown()) != string(r2.Markdown()) {
		t.Error("render is not deterministic across runs")
	}
}
