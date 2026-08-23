package oracle

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/kata"
)

// TestKataGolden is the kata gate. It mirrors TestGolden and shares its rule:
// WRONG entries fail regardless of -update. A wrong bound on a human-graded
// solution is a prime-directive break exactly as one on a literature-pinned
// algorithm is.
//
// It scores under the kata cost model because that is the model the pins were
// claimed in. Running it unaided would report a corpus of top rows and measure
// a question nobody asked.
func TestKataGolden(t *testing.T) {
	srcRoot, err := filepath.Abs(filepath.Join("..", "..", "corpus", "kata", "testdata", "src"))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := kata.Profile()
	if err != nil {
		t.Fatal(err)
	}
	spaceProfile, err := kata.SpaceProfile()
	if err != nil {
		t.Fatal(err)
	}
	r, wrongs, err := CollectWith(srcRoot, Options{Corpus: KataCorpus, Overlay: profile, SpaceOverlay: spaceProfile})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range wrongs {
		t.Errorf("WRONG BOUND (release-blocking): %s.%s %s — pinned %s, emitted %s",
			w.Pkg, w.Func, w.Dim, w.Pin, w.Got)
	}
	if len(wrongs) > 0 {
		t.Fatal("the kata oracle found wrong bounds; fix the engine or the pin (with written reasoning) — never the classifier")
	}

	jsonPath, err := filepath.Abs(filepath.Join("..", "..", "corpus", "kata.json"))
	if err != nil {
		t.Fatal(err)
	}
	mdPath, err := filepath.Abs(filepath.Join("..", "..", "corpus", "KATA.md"))
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
			t.Fatal(err)
		}
		if !bytes.Equal(want, tc.got) {
			t.Errorf("%s is stale — run `task kata-corpus` and READ the diff before committing it", filepath.Base(tc.path))
		}
	}
}
