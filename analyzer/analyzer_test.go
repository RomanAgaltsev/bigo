package analyzer

import (
	"io"
	"os"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzerSpace(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "space")
}

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

func TestAnalyzerFieldsize(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "fieldsize")
}

func TestAnalyzerRecursion(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "recursion")
}

func TestAnalyzerFuncValue(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "funcvalue")
}

func TestAnalyzerIterator(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "iterator")
}

func TestAnalyzerMutual(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "mutual")
}

func TestAnalyzerSmells(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "smells")
}

func TestReportModeSpaceLine(t *testing.T) {
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
	analysistest.Run(t, analysistest.TestData(), Analyzer, "space")
	_ = w.Close()
	os.Stdout = old
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "RecSum: space O(len(xs))") {
		t.Errorf("report output missing space line for RecSum, got: %q", out)
	}
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
	// Report mode must name unverifiable functions too — they are exactly the
	// ones a user would annotate — and say why.
	if !strings.Contains(string(out), "Closure: unverifiable") {
		t.Errorf("report output should name unverifiable functions, got: %q", out)
	}
}

func TestReportEmitsBothAxesInSourceOrder(t *testing.T) {
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
	analysistest.Run(t, analysistest.TestData(), Analyzer, "reportaxes")
	_ = w.Close()
	os.Stdout = old
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	out := string(raw)

	// Space is reported for a function with no space directive: it used to be
	// printed by the budget checker, so only annotated functions ever got it.
	for _, want := range []string{
		"Zed: inferred complexity O(len(xs))",
		"Zed: space ",
		"Alpha: inferred complexity O(len(xs))",
		"Alpha: space ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q\ngot:\n%s", want, out)
		}
	}

	// Source order, not map order: Zed is declared first though it sorts last.
	if i, j := strings.Index(out, "Zed:"), strings.Index(out, "Alpha:"); i < 0 || j < 0 || i > j {
		t.Errorf("report not in source order: Zed at %d, Alpha at %d\ngot:\n%s", i, j, out)
	}

	// A function's two axes must be adjacent so a reader can scan them in pairs.
	zt := strings.Index(out, "Zed: inferred complexity")
	zs := strings.Index(out, "Zed: space")
	if zt < 0 || zs < zt || strings.Count(out[zt:zs], "\n") != 1 {
		t.Errorf("time and space lines for Zed are not adjacent\ngot:\n%s", out)
	}
}

// runKataReport captures -report output for pkg, with -kata on or off.
func runKataReport(t *testing.T, pkg string, kata bool) string {
	t.Helper()
	if err := Analyzer.Flags.Set("report", "true"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Analyzer.Flags.Set("report", "false") }()
	if kata {
		if err := Analyzer.Flags.Set("kata", "true"); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = Analyzer.Flags.Set("kata", "false") }()
	}
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	analysistest.Run(t, analysistest.TestData(), Analyzer, pkg)
	_ = w.Close()
	os.Stdout = old
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// The kata cost model answers BOTH graded axes. Space is a separate profile
// and a separate claim per entry: an external callee has no body to analyze,
// so without an overlay every stdlib call is unverifiable heap.
func TestKataModeAnswersSpace(t *testing.T) {
	t.Run("off by default space stays unverifiable", func(t *testing.T) {
		out := runKataReport(t, "katamode", false)
		for _, want := range []string{"Convert: space unverifiable", "Compare: space unverifiable"} {
			if !strings.Contains(out, want) {
				t.Errorf("without -kata, expected %q\ngot:\n%s", want, out)
			}
		}
	})

	t.Run("with -kata space is answered", func(t *testing.T) {
		out := runKataReport(t, "katamode", true)
		for _, want := range []string{"Convert: space O(1)", "Compare: space O(1)"} {
			if !strings.Contains(out, want) {
				t.Errorf("with -kata, expected %q\ngot:\n%s", want, out)
			}
		}
	})
}

func TestKataModeAppliesTheOverlay(t *testing.T) {
	run := func(t *testing.T, kata bool) string {
		t.Helper()
		return runKataReport(t, "katamode", kata)
	}

	// The regression guard. A test that only checked the -kata path would pass
	// even if the overlay were always on, which would move every user's bounds.
	t.Run("off by default the curated bounds stand", func(t *testing.T) {
		out := run(t, false)
		for _, want := range []string{
			"Convert: inferred complexity O(len(s))",
			"Compare: inferred complexity O(len(a))",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("without -kata, expected %q\ngot:\n%s", want, out)
			}
		}
	})

	t.Run("with -kata the curated bounds are overridden", func(t *testing.T) {
		out := run(t, true)
		for _, want := range []string{
			"Convert: inferred complexity O(1)",
			"Compare: inferred complexity O(1)",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("with -kata, expected %q\ngot:\n%s", want, out)
			}
		}
	})

	// The overlay answers call COSTS and does not invent SIZES: a loop over a
	// call result has no nameable trip count either way. Pinned so a later
	// widening of the profile cannot be mistaken for fixing this class.
	t.Run("the overlay does not resolve sizes", func(t *testing.T) {
		for _, kata := range []bool{false, true} {
			if out := run(t, kata); !strings.Contains(out, "ParseLine: unverifiable") {
				t.Errorf("ParseLine should be unverifiable with kata=%v\ngot:\n%s", kata, out)
			}
		}
	})
}
