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

func TestMainAssumeFlag(t *testing.T) {
	out := filepath.Join(t.TempDir(), "doc.json")
	code := Main("test", []string{"-C", "testdata/assumefix", "-assume", "testdata/assumefix/fix.assume", "-o", out})
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Assumptions) != 1 {
		t.Fatalf("assumptions = %+v, want the os.Getenv entry", doc.Assumptions)
	}
}

func TestMainAssumeFlagBadFileFails(t *testing.T) {
	if code := Main("test", []string{"-C", "testdata/assumefix", "-assume", "testdata/assumefix/nosuch.assume"}); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
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

// TestTrustFlagLoadsLikeAssume: -trust is the product entry point for the same
// loader. Same mechanism, different epistemic status — an assumption is
// hypothetical, a trust entry is a claim the user stands behind — so they get
// different names and one code path.
//
// The path is process-relative, not -C-relative, like -assume.
func TestTrustFlagLoadsLikeAssume(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.json")
	if code := Main("test", []string{"-C", "testdata/assumefix", "-trust", "testdata/assumefix/fix.assume", "-o", out}); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Assumptions) != 1 {
		t.Fatalf("trust surface = %+v, want the one entry", doc.Assumptions)
	}
	var tainted int
	for _, f := range doc.Functions {
		if f.Provenance == ProvenanceTainted {
			tainted++
		}
	}
	if tainted == 0 {
		t.Error("no tainted verdict — the trust file was not applied")
	}
}

// TestTrustAndAssumeTogetherIsAUsageError: silently unioning them would erase
// the distinction the flag exists to make.
func TestTrustAndAssumeTogetherIsAUsageError(t *testing.T) {
	code := Main("test", []string{
		"-C", "testdata/assumefix",
		"-trust", "testdata/assumefix/fix.assume",
		"-assume", "testdata/assumefix/fix.assume",
	})
	if code != 2 {
		t.Errorf("exit = %d, want 2 (usage error)", code)
	}
}

// bigo json had no way to ask for the kata cost model, so a kata user could
// get -report under their model but not the document. That split is what
// produced the trust init -kata defect: the report path could not apply an
// overlay at all.
//
// The fixture's only blocker is a bufio write, which the kata profile prices
// because K-1 says input and output are not graded work.
func TestMainKataFlag(t *testing.T) {
	load := func(args ...string) Document {
		t.Helper()
		out := filepath.Join(t.TempDir(), "doc.json")
		code := Main("test", append(args, "-o", out))
		if code != 0 {
			t.Fatalf("exit = %d, want 0", code)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		var doc Document
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		return doc
	}
	topFor := func(doc Document, name string) bool {
		t.Helper()
		for _, f := range doc.Functions {
			if f.Func == name {
				return f.Time.Top
			}
		}
		t.Fatalf("%s not in the document", name)
		return false
	}

	base := load("-C", "testdata/trustinitkata")
	if !topFor(base, "Write") {
		t.Fatal("Write must be unverifiable under the default cost model, or the fixture proves nothing")
	}

	kataDoc := load("-C", "testdata/trustinitkata", "-kata")
	if topFor(kataDoc, "Write") {
		t.Error("Write must be bounded under -kata: the profile prices the write")
	}
}
