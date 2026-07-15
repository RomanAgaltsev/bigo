package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMainWritesValidDocument(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.json")
	code := Main("test", []string{"-C", "testdata/reportfix", "-o", out, "./..."})
	if code != 0 {
		t.Fatalf("Main = %d, want 0 (exceeds/unverifiable verdicts must not affect the exit code)", code)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("output is not a Document: %v", err)
	}
	if doc.SchemaVersion != SchemaVersion || doc.Module != "example.com/reportfix" {
		t.Errorf("envelope = %q/%q, want %q/example.com/reportfix", doc.SchemaVersion, doc.Module, SchemaVersion)
	}
}

func TestMainBadDirFails(t *testing.T) {
	if code := Main("test", []string{"-C", "testdata/nosuchdir", "./..."}); code != 1 {
		t.Errorf("Main on missing dir = %d, want 1", code)
	}
}

func TestMainBadFlagFails(t *testing.T) {
	if code := Main("test", []string{"-nosuchflag"}); code != 2 {
		t.Errorf("Main with bad flag = %d, want 2", code)
	}
}
