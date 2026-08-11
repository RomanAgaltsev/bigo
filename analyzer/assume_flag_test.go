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

// TestTrustFlagChangesVerdict is TestAssumeFlagChangesVerdict through the
// PRODUCT flag: same loader, same fixture, same verdict. It exists because the
// two names are the user-visible surface and a regression that broke only one
// of them would otherwise ship.
func TestTrustFlagChangesVerdict(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "t.trust"), []byte("os.Getenv O(1)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	assumeOnce = sync.Once{}
	assumeSet, assumeErr = nil, nil
	trustFile = filepath.Join(dir, "t.trust")
	defer func() {
		trustFile = ""
		assumeOnce = sync.Once{}
		assumeSet, assumeErr = nil, nil
	}()
	analysistest.Run(t, analysistest.TestData(), Analyzer, "assumeok")
}

// TestTrustAndAssumeTogetherIsAnError: unioning two flags that mean different
// things would erase the distinction they exist to draw, so the loader refuses.
func TestTrustAndAssumeTogetherIsAnError(t *testing.T) {
	assumeOnce = sync.Once{}
	assumeSet, assumeErr = nil, nil
	assumeFile, trustFile = "a.assume", "b.trust"
	defer func() {
		assumeFile, trustFile = "", ""
		assumeOnce = sync.Once{}
		assumeSet, assumeErr = nil, nil
	}()
	if _, err := loadAssumptions(); err == nil {
		t.Error("loadAssumptions accepted both flags; want an error")
	}
}
