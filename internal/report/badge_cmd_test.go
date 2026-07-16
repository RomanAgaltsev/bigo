package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The reportfix fixture yields, across time and space budgets: within×3
// (WithinBudget, UsesTrust, Doubled-space), exceeds×1 (ExceedsBudget),
// unverifiable×1 (Unverifiable), invalid×1 (InvalidBudget) = 6 budgets.
func TestBadgeMainSelfContained(t *testing.T) {
	out := filepath.Join(t.TempDir(), "badge.json")
	if code := BadgeMain("test", []string{"-C", "testdata/reportfix", "-o", out, "./..."}); code != 0 {
		t.Fatalf("BadgeMain = %d, want 0 (verdicts must not affect the exit code)", code)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var ep Endpoint
	if err := json.Unmarshal(raw, &ep); err != nil {
		t.Fatalf("output is not an Endpoint: %v", err)
	}
	want := Endpoint{SchemaVersion: 1, Label: "bigo", Message: "6 budgets · 1 exceeds, 1 invalid, 1 unverifiable", Color: "red"}
	if ep != want {
		t.Errorf("badge = %+v, want %+v", ep, want)
	}
}

func TestBadgeMainReadsDocument(t *testing.T) {
	docPath := filepath.Join(t.TempDir(), "report.json")
	doc := Document{Functions: []Function{{Func: "F", Budget: &BudgetJSON{Verdict: "within"}}}}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(docPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "badge.json")
	if code := BadgeMain("test", []string{"-i", docPath, "-o", out}); code != 0 {
		t.Fatalf("BadgeMain -i = %d, want 0", code)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var ep Endpoint
	if err := json.Unmarshal(raw, &ep); err != nil {
		t.Fatal(err)
	}
	if ep.Message != "1 budget · all within" || ep.Color != "brightgreen" {
		t.Errorf("badge = %q/%q, want '1 budget · all within'/brightgreen", ep.Message, ep.Color)
	}
}

func TestBadgeMainBadDocFails(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := BadgeMain("test", []string{"-i", bad}); code != 1 {
		t.Errorf("BadgeMain on malformed document = %d, want 1", code)
	}
}

func TestBadgeMainBadFlagFails(t *testing.T) {
	if code := BadgeMain("test", []string{"-nosuchflag"}); code != 2 {
		t.Errorf("BadgeMain with bad flag = %d, want 2", code)
	}
}
