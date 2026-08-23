package oracle

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The kata fixtures are REDUCED. This pins the reduction rather than trusting
// it: a fixture that regrows a contest URL or a block of course prose is a
// privacy defect in a public repository, and reviewers do not reliably catch a
// paste.
//
// The banned list is deliberately about the ORIGINAL's header furniture rather
// than about Russian text in general — an //oracle:source quoting the author's
// claim is exactly what a pin should carry.
func TestKataFixturesAreReduced(t *testing.T) {
	root := filepath.Join("..", "..", "corpus", "kata", "testdata", "src")
	banned := []string{
		"contest.yandex.ru",         // submission report links
		"Практикум",                 // course name
		"когорта",                   // cohort line
		"ПРИНЦИП РАБОТЫ",            // the review template's sections
		"-- ВРЕМЕННАЯ СЛОЖНОСТЬ --", // the claim block belongs in //oracle: pins
		"-- ПРОСТРАНСТВЕННАЯ СЛОЖНОСТЬ --",
	}
	var files int
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		files++
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(b)
		for _, bad := range banned {
			if strings.Contains(body, bad) {
				t.Errorf("%s contains %q — fixtures are reduced; the original's header does not belong in a public repo", path, bad)
			}
		}
		if !strings.Contains(body, "//oracle:") {
			t.Errorf("%s has no //oracle: pin — an unpinned fixture is a row that cannot fail", path)
		}
		if strings.Contains(body, "func main(") {
			t.Errorf("%s has a main function — the I/O scaffolding is what the reduction removes", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files == 0 {
		t.Fatal("no kata fixtures found; the corpus root moved or the walk is wrong")
	}
}
