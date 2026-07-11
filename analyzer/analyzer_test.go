package analyzer

import (
	"io"
	"os"
	"strings"
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

func TestAnalyzerCostIgnore(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "costignore")
}

func TestStructuresArray(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "structures/array")
}

func TestStructuresList(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "structures/list")
}

func TestStructuresTree(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "structures/tree")
}

func TestStructuresGraph(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "structures/graph")
}

func TestStructuresHeap(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "structures/heap")
}

func TestStructuresHashmap(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "structures/hashmap")
}

func TestStructuresStringops(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "structures/stringops")
}

func TestReportModeUsesStdoutNotDiagnostics(t *testing.T) {
	if err := Analyzer.Flags.Set("report", "true"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Analyzer.Flags.Set("report", "false") }()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	// smoke has no budgets: report mode must print bounds to stdout and emit
	// ZERO diagnostics (analysistest fails on any unexpected diagnostic —
	// that is the exit-code guarantee).
	analysistest.Run(t, analysistest.TestData(), Analyzer, "smoke")
	_ = w.Close()
	os.Stdout = old
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "Noop: inferred complexity O(len(xs))") {
		t.Errorf("report output missing, got: %q", out)
	}
}
