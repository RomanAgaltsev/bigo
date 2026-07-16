package report

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite the golden file")

// fixedNow is the injected clock shared by every fixture collection, so
// documents are byte-deterministic across tests.
func fixedNow() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

// Body lines of reportfix's IgnoredSmell, which carries //bigo:ignore. A smell
// is positioned at the offending instruction, not the func line, so the ignore
// check spans the body rather than pinning a single line.
const (
	ignoredSmellLine    = 133
	ignoredSmellEndLine = 139
)

// collectDoc builds the fixture document, for tests that assert on its
// structure rather than on the serialized bytes.
func collectDoc(t *testing.T) Document {
	t.Helper()
	doc, err := Collect("testdata/reportfix", []string{"./..."}, Options{
		Version: "test",
		Now:     fixedNow,
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	return doc
}

func collectFixture(t *testing.T) []byte {
	t.Helper()
	got, err := json.MarshalIndent(collectDoc(t), "", "  ")
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
	if len(doc.Functions) != 12 {
		t.Errorf("functions = %d, want 12 (Linear, WithinBudget, ExceedsBudget, Unverifiable, Sum, telemetry, extLookup, UsesTrust, Doubled, InvalidBudget, ConcatInLoop, IgnoredSmell)", len(doc.Functions))
	}
	if len(doc.Trusted) != 3 {
		t.Errorf("trusted = %d, want 3 (telemetry ignore, extLookup cost, IgnoredSmell ignore)", len(doc.Trusted))
	}
}

func TestCollectSmells(t *testing.T) {
	doc := collectDoc(t)
	if len(doc.Smells) == 0 {
		t.Fatal("no smells collected; reportfix declares a deliberate one")
	}
	var found bool
	for _, s := range doc.Smells {
		if s.Rule != "SM1" {
			continue
		}
		found = true
		if s.File == "" || s.Line == 0 {
			t.Errorf("smell %+v missing position", s)
		}
		if strings.Contains(s.Message, "smell(") {
			t.Errorf("message %q must not carry the diagnostic prefix", s.Message)
		}
	}
	if !found {
		t.Errorf("want an SM1 (concat in loop) finding, got %+v", doc.Smells)
	}
}

func TestCollectSmellsSorted(t *testing.T) {
	doc := collectDoc(t)
	for i := 1; i < len(doc.Smells); i++ {
		a, b := doc.Smells[i-1], doc.Smells[i]
		if a.File > b.File || (a.File == b.File && a.Line > b.Line) ||
			(a.File == b.File && a.Line == b.Line && a.Rule > b.Rule) {
			t.Errorf("smells not sorted at %d: %+v then %+v", i, a, b)
		}
	}
}

// TestCollectSmellsHonorsIgnore pins that an ignored decl contributes no
// smells, as it emits no diagnostics. ConcatInLoop and IgnoredSmell have
// identical bodies and differ only by //bigo:ignore, so a broken ignore yields
// two SM1 findings instead of one — the count is what makes this test able to
// fail. (Asserting on the message cannot: SM1's text is a fixed string that
// never names its function.)
func TestCollectSmellsHonorsIgnore(t *testing.T) {
	doc := collectDoc(t)
	var sm1 int
	for _, s := range doc.Smells {
		if s.Rule == "SM1" {
			sm1++
		}
		if s.File == "fix.go" && s.Line >= ignoredSmellLine && s.Line <= ignoredSmellEndLine {
			t.Errorf("smell inside the ignored decl: %+v", s)
		}
	}
	if sm1 != 1 {
		t.Errorf("SM1 findings = %d, want 1 (ConcatInLoop only; IgnoredSmell is //bigo:ignore'd)", sm1)
	}
}

func TestSchemaVersionIsMinorBump(t *testing.T) {
	// Adding smells is additive: the major must not move.
	if !strings.HasPrefix(SchemaVersion, "1.") {
		t.Errorf("SchemaVersion = %q; adding a field must not bump the major", SchemaVersion)
	}
	if SchemaVersion == "1.0.0" {
		t.Error("SchemaVersion still 1.0.0; an additive field requires a minor bump")
	}
}
