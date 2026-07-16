package report

import (
	"strings"
	"testing"
)

var sample = []Finding{
	{Class: BudgetBreak, Key: "m/p.SortUsers", File: "p/sort.go", Line: 12, Message: "SortUsers exceeds its O(n) budget: inferred O(n^2)"},
	{Class: Improvement, Key: "m/p.Scan", File: "p/scan.go", Line: 30, Message: "Scan improved: O(n^2) → O(n)"},
}

func TestFormatTextEmptyIsReassuring(t *testing.T) {
	got := FormatText(nil, "")
	if !strings.Contains(got, "no complexity changes") {
		t.Errorf("FormatText(nil) = %q, want an explicit all-clear", got)
	}
}

func TestFormatTextListsFindingsWithPositions(t *testing.T) {
	got := FormatText(sample, "")
	for _, want := range []string{"p/sort.go:12", "SortUsers exceeds", "p/scan.go:30", "improvement"} {
		if !strings.Contains(got, want) {
			t.Errorf("FormatText missing %q in:\n%s", want, got)
		}
	}
}

func TestFormatTextShowsWarning(t *testing.T) {
	got := FormatText(nil, "bigo version differs")
	if !strings.Contains(got, "bigo version differs") {
		t.Errorf("FormatText dropped the warning: %q", got)
	}
}

func TestFormatMarkdownHasMarkerAndHeading(t *testing.T) {
	got := FormatMarkdown(sample, "")
	if !strings.HasPrefix(got, CommentMarker) {
		t.Errorf("markdown must start with %q for comment updating, got:\n%s", CommentMarker, got)
	}
	if !strings.Contains(got, "## bigo") {
		t.Errorf("markdown missing heading:\n%s", got)
	}
}

func TestFormatMarkdownGroupsBySeverity(t *testing.T) {
	got := FormatMarkdown(sample, "")
	bi := strings.Index(got, "Budget break")
	ii := strings.Index(got, "Improvement")
	if bi < 0 || ii < 0 {
		t.Fatalf("markdown missing class headings:\n%s", got)
	}
	if bi > ii {
		t.Error("budget breaks must be listed before improvements")
	}
}

func TestFormatMarkdownEmptyStillHasMarker(t *testing.T) {
	// The all-clear comment must still carry the marker so it can be updated
	// in place on the next push.
	got := FormatMarkdown(nil, "")
	if !strings.HasPrefix(got, CommentMarker) {
		t.Errorf("empty markdown lost the marker:\n%s", got)
	}
}

func TestFormatMarkdownWarningIsProminent(t *testing.T) {
	got := FormatMarkdown(nil, "bigo version differs (base 1.19.0, head 1.20.0)")
	if !strings.Contains(got, "⚠️") || !strings.Contains(got, "1.20.0") {
		t.Errorf("markdown warning not prominent:\n%s", got)
	}
}
