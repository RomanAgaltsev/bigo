package analyzer

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzerSmoke(t *testing.T) {
	// No //bigo annotations -> no diagnostics.
	analysistest.Run(t, analysistest.TestData(), Analyzer, "smoke")
}
