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

func TestAnalyzerPositive(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "positive")
}

func TestAnalyzerNegative(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "negative")
}

func TestAnalyzerInterproc(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "interproc")
}

func TestAnalyzerMultivar(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "multivar")
}

func TestAnalyzerEdge(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "edge")
}

func TestAnalyzerConcurrent(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "concurrent")
}
