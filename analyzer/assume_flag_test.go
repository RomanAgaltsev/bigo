package analyzer

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAssumeFlagChangesVerdict sets the package-level flag var directly:
// analysistest offers no way to pass driver flags. The defer re-arms the
// once-loader so tests running after this one never see a stale set.
func TestAssumeFlagChangesVerdict(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "t.assume"), []byte("os.Getenv O(1)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Re-arm BEFORE running too: an earlier test in this package has already
	// consumed the once (with an empty flag), which would pin the set to nil.
	assumeOnce = sync.Once{}
	assumeSet, assumeErr = nil, nil
	assumeFile = filepath.Join(dir, "t.assume")
	defer func() {
		assumeFile = ""
		assumeOnce = sync.Once{}
		assumeSet, assumeErr = nil, nil
	}()
	analysistest.Run(t, analysistest.TestData(), Analyzer, "assumeok")
}
