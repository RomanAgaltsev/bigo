package report

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite the golden file")

func collectFixture(t *testing.T) []byte {
	t.Helper()
	doc, err := Collect("testdata/reportfix", []string{"./..."}, Options{
		Version: "test",
		Now:     func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	got, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return append(got, '\n')
}

func TestGolden(t *testing.T) {
	got := collectFixture(t)
	golden := filepath.Join("testdata", "golden", "report.json")
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run `task report-golden` to create): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("document differs from golden; if the change is intended run `task report-golden` and hand-verify the diff\n--- got ---\n%s", got)
	}
}

// TestGoldenEnvelope pins the deterministic envelope independent of inference
// results, so an engine-driven golden regeneration cannot silently break it.
func TestGoldenEnvelope(t *testing.T) {
	got := collectFixture(t)
	var doc Document
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %q, want %q", doc.SchemaVersion, SchemaVersion)
	}
	if doc.BigoVersion != "test" || doc.Module != "example.com/reportfix" {
		t.Errorf("envelope = %q/%q, want test/example.com/reportfix", doc.BigoVersion, doc.Module)
	}
	if doc.Generated != "2026-01-02T03:04:05Z" {
		t.Errorf("generated = %q, want fixed injected time", doc.Generated)
	}
	if len(doc.Functions) != 10 {
		t.Errorf("functions = %d, want 10 (Linear, WithinBudget, ExceedsBudget, Unverifiable, Sum, telemetry, extLookup, UsesTrust, Doubled, InvalidBudget)", len(doc.Functions))
	}
	if len(doc.Trusted) != 2 {
		t.Errorf("trusted = %d, want 2 (telemetry ignore, extLookup cost)", len(doc.Trusted))
	}
}
