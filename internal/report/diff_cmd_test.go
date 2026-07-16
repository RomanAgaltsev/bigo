package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeDoc marshals a document to a temp file and returns its path.
func writeDoc(t *testing.T, d Document) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "doc.json")
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDiffMainCleanExitsZero(t *testing.T) {
	base := writeDoc(t, doc(fn("F", bj(oN), nil)))
	head := writeDoc(t, doc(fn("F", bj(oN), nil)))
	if code := DiffMain([]string{base, head}); code != 0 {
		t.Errorf("DiffMain = %d, want 0", code)
	}
}

func TestDiffMainFindingsStillExitZero(t *testing.T) {
	// Verdicts never affect this subcommand's exit code — enforcement is the
	// Action's knob, matching `bigo json` and `bigo badge` (cmd.go:12-14).
	base := writeDoc(t, doc(fn("F", bj(oN), budget(oN, "within"))))
	head := writeDoc(t, doc(fn("F", bj(oN2), budget(oN, "exceeds"))))
	if code := DiffMain([]string{base, head}); code != 0 {
		t.Errorf("DiffMain with a budget break = %d, want 0", code)
	}
}

func TestDiffMainWritesMarkdownToFile(t *testing.T) {
	base := writeDoc(t, doc(fn("F", bj(oN), budget(oN, "within"))))
	head := writeDoc(t, doc(fn("F", bj(oN2), budget(oN, "exceeds"))))
	out := filepath.Join(t.TempDir(), "comment.md")
	if code := DiffMain([]string{"-format", "markdown", "-o", out, base, head}); code != 0 {
		t.Fatalf("DiffMain = %d, want 0", code)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 || string(raw[:len(CommentMarker)]) != CommentMarker {
		t.Errorf("output is not a marked markdown comment: %q", raw)
	}
}

func TestDiffMainWrongArgCountFails(t *testing.T) {
	for _, args := range [][]string{{}, {"only-one.json"}} {
		if code := DiffMain(args); code != 2 {
			t.Errorf("DiffMain(%v) = %d, want 2", args, code)
		}
	}
}

func TestDiffMainMissingFileFails(t *testing.T) {
	base := writeDoc(t, doc())
	if code := DiffMain([]string{base, filepath.Join(t.TempDir(), "nope.json")}); code != 1 {
		t.Error("DiffMain on a missing file should exit 1")
	}
}

func TestDiffMainBadDocFails(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := DiffMain([]string{bad, writeDoc(t, doc())}); code != 1 {
		t.Error("DiffMain on a malformed document should exit 1")
	}
}

func TestDiffMainIncompatibleFails(t *testing.T) {
	other := doc()
	other.Module = "example.com/other"
	if code := DiffMain([]string{writeDoc(t, doc()), writeDoc(t, other)}); code != 1 {
		t.Error("DiffMain across modules should exit 1")
	}
}

func TestDiffMainBadFormatFails(t *testing.T) {
	base, head := writeDoc(t, doc()), writeDoc(t, doc())
	if code := DiffMain([]string{"-format", "xml", base, head}); code != 2 {
		t.Error("DiffMain with an unknown -format should exit 2")
	}
}
