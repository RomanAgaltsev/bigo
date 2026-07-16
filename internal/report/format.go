package report

import (
	"fmt"
	"strings"
)

// CommentMarker identifies a bigo PR comment so the Action updates one comment
// in place instead of appending a new one per push. It is an HTML comment and
// renders invisibly on GitHub.
const CommentMarker = "<!-- bigo-diff -->"

// FormatText renders findings for a terminal.
func FormatText(fs []Finding, warning string) string {
	var b strings.Builder
	if warning != "" {
		fmt.Fprintf(&b, "warning: %s\n\n", warning)
	}
	if len(fs) == 0 {
		b.WriteString("bigo: no complexity changes\n")
		return b.String()
	}
	for _, f := range fs {
		fmt.Fprintf(&b, "%s:%d: %s: %s\n", f.File, f.Line, f.Class, f.Message)
	}
	fmt.Fprintf(&b, "\n%s\n", summary(fs))
	return b.String()
}

// FormatMarkdown renders findings as a PR comment body, grouped by class in
// severity order. Always carries CommentMarker, including the all-clear case.
func FormatMarkdown(fs []Finding, warning string) string {
	var b strings.Builder
	b.WriteString(CommentMarker + "\n## bigo — complexity diff\n\n")
	if warning != "" {
		fmt.Fprintf(&b, "> ⚠️ %s\n\n", warning)
	}
	if len(fs) == 0 {
		b.WriteString("No complexity changes.\n")
		return b.String()
	}
	for _, c := range []Class{BudgetBreak, ProvenRegression, NewTop, NewFuncBreak, Improvement} {
		group := filter(fs, c)
		if len(group) == 0 {
			continue
		}
		fmt.Fprintf(&b, "**%s** (%d)\n\n", heading(c), len(group))
		for _, f := range group {
			fmt.Fprintf(&b, "- `%s:%d` — %s\n", f.File, f.Line, f.Message)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "%s\n", summary(fs))
	return b.String()
}

// heading is Class.String() capitalized for a markdown heading.
func heading(c Class) string {
	s := c.String()
	return strings.ToUpper(s[:1]) + s[1:]
}

func filter(fs []Finding, c Class) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Class == c {
			out = append(out, f)
		}
	}
	return out
}

// summary counts findings, separating the good news so a PR that only improves
// things never reads as an alarm.
func summary(fs []Finding) string {
	good := len(filter(fs, Improvement))
	bad := len(fs) - good
	switch {
	case bad == 0:
		return fmt.Sprintf("%d improvement(s), no regressions.", good)
	case good == 0:
		return fmt.Sprintf("%d finding(s).", bad)
	default:
		return fmt.Sprintf("%d finding(s), %d improvement(s).", bad, good)
	}
}
