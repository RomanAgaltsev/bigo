package analyzer

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzerSmoke(t *testing.T) {
	// No //bigo annotations -> no diagnostics.
	analysistest.Run(t, analysistest.TestData(), Analyzer, "smoke")
}

func TestAnalyzerBudgets(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "budget")
}
