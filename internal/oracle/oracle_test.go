package oracle

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite corpus/corpus.json and corpus/CORPUS.md")

// TestGolden is the oracle gate. WRONG entries fail regardless of -update:
// a wrong bound on a literature-pinned algorithm is a prime-directive break
// and no golden may bless it (spec §4.2).
func TestGolden(t *testing.T) {
	srcRoot, err := filepath.Abs(filepath.Join("..", "..", "corpus", "testdata", "src"))
	if err != nil {
		t.Fatal(err)
	}
	r, wrongs, err := Collect(srcRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range wrongs {
		t.Errorf("WRONG BOUND (release-blocking): %s.%s %s — pinned %s, emitted %s",
			w.Pkg, w.Func, w.Dim, w.Pin, w.Got)
	}
	if len(wrongs) > 0 {
		t.Fatal("the oracle found wrong bounds; fix the engine or the pin (with written reasoning) — never the classifier")
	}

	jsonPath, err := filepath.Abs(filepath.Join("..", "..", "corpus", "corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	mdPath, err := filepath.Abs(filepath.Join("..", "..", "corpus", "CORPUS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if *update {
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
			t.Fatalf("read golden (run `task corpus` to create it): %v", err)
		}
		if !bytes.Equal(normalizeEOL(tc.got), normalizeEOL(want)) {
			t.Errorf("%s is stale — verdicts changed; run `task corpus` and commit the diff", filepath.Base(tc.path))
		}
	}
}

// normalizeEOL strips CR so the comparison survives core.autocrlf checkouts.
func normalizeEOL(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
