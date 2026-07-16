package report

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Main runs the `bigo json` subcommand. Exit codes: 0 success (verdicts never
// affect the exit code — the report describes, a future diff tool enforces),
// 1 analysis or IO error, 2 flag error.
func Main(version string, args []string) int {
	fs := flag.NewFlagSet("bigo json", flag.ContinueOnError)
	dir := fs.String("C", ".", "analyze the module in this directory")
	out := fs.String("o", "", "write the report to this file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	doc, err := Collect(*dir, fs.Args(), Options{Version: version})
	if err != nil {
		fmt.Fprintln(os.Stderr, "bigo json:", err)
		return 1
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "bigo json:", err)
		return 1
	}
	data = append(data, '\n')
	if *out == "" {
		_, err = os.Stdout.Write(data)
	} else {
		err = os.WriteFile(*out, data, 0o600)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bigo json:", err)
		return 1
	}
	return 0
}

// readInput reads a report document from a file path, or from stdin when path
// is "-", so `bigo json ./... | bigo badge -i -` composes. Reads are scoped
// under the file's directory via an os.Root (Go 1.24+), which rejects any
// traversal — the gosec G304 (CWE-22) clean idiom for a user-supplied path.
func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	return root.ReadFile(filepath.Base(path))
}

// BadgeMain runs the `bigo badge` subcommand: it emits the shields.io endpoint
// badge, either by analyzing the module (default) or by projecting a report
// document supplied with -i ("-" for stdin). Exit codes match `bigo json`:
// 0 success (verdicts never affect the exit code), 1 analysis/IO/parse error,
// 2 flag error.
func BadgeMain(version string, args []string) int {
	fs := flag.NewFlagSet("bigo badge", flag.ContinueOnError)
	dir := fs.String("C", ".", "analyze the module in this directory")
	out := fs.String("o", "", "write the badge JSON to this file instead of stdout")
	in := fs.String("i", "", `read a bigo json document instead of analyzing ("-" for stdin)`)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	var doc Document
	if *in != "" {
		data, err := readInput(*in)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bigo badge:", err)
			return 1
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			fmt.Fprintln(os.Stderr, "bigo badge:", err)
			return 1
		}
	} else {
		d, err := Collect(*dir, fs.Args(), Options{Version: version})
		if err != nil {
			fmt.Fprintln(os.Stderr, "bigo badge:", err)
			return 1
		}
		doc = d
	}
	data, err := json.MarshalIndent(Badge(doc), "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "bigo badge:", err)
		return 1
	}
	data = append(data, '\n')
	if *out == "" {
		_, err = os.Stdout.Write(data)
	} else {
		err = os.WriteFile(*out, data, 0o600)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bigo badge:", err)
		return 1
	}
	return 0
}

// DiffMain runs the `bigo diff base.json head.json` subcommand: it compares two
// report documents and renders the findings. Pure and offline — it analyzes
// nothing and reads no source.
//
// Exit codes match the other subcommands: 0 success, 1 IO/parse/compatibility
// error, 2 usage error. Findings affect the exit code only when the caller opts
// in with -fail-on, which then adds 3 for a policy violation: by default the
// report describes and the consumer enforces.
func DiffMain(args []string) int {
	fs := flag.NewFlagSet("bigo diff", flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text | markdown")
	out := fs.String("o", "", "write the output to this file instead of stdout")
	failOn := fs.String("fail-on", "none", "exit 3 when findings reach this severity: none | break | regression")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *format != "text" && *format != "markdown" {
		fmt.Fprintf(os.Stderr, "bigo diff: unknown -format %q (want text or markdown)\n", *format)
		return 2
	}
	if *failOn != "none" && *failOn != "break" && *failOn != "regression" {
		fmt.Fprintf(os.Stderr, "bigo diff: unknown -fail-on %q (want none, break, or regression)\n", *failOn)
		return 2
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: bigo diff [-format text|markdown] [-fail-on none|break|regression] [-o file] base.json head.json")
		return 2
	}
	base, err := loadDoc(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "bigo diff:", err)
		return 1
	}
	head, err := loadDoc(fs.Arg(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "bigo diff:", err)
		return 1
	}
	findings, warn, err := Diff(base, head)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bigo diff:", err)
		return 1
	}
	var text string
	if *format == "markdown" {
		text = FormatMarkdown(findings, warn)
	} else {
		text = FormatText(findings, warn)
	}
	if *out == "" {
		_, err = os.Stdout.WriteString(text)
	} else {
		err = os.WriteFile(*out, []byte(text), 0o600)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bigo diff:", err)
		return 1
	}
	return policyCode(findings, *failOn)
}

// policyCode applies the exit-code policy after the output has been written.
// Exit 3 is a policy violation — distinct from 1 (bigo failed) and 2 (usage),
// so a CI step can tell "your code broke" from "the tool broke".
//
// "break" fails only on a broken declared budget (class 1) and on a new
// function that arrives already over budget (class 4): both are violations of
// a contract the repo itself wrote. "regression" additionally fails on a proven
// asymptotic regression in unbudgeted code (class 2). No policy fails on a new
// ⊤ (class 3): losing visibility is worth reporting but is not a defect, and
// failing on it would pressure authors to avoid constructs bigo cannot yet see.
func policyCode(fs []Finding, failOn string) int {
	worst, found := Severity(fs)
	if !found || failOn == "none" {
		return 0
	}
	switch failOn {
	case "break":
		if worst == BudgetBreak || worst == NewFuncBreak {
			return 3
		}
	case "regression":
		if worst == BudgetBreak || worst == NewFuncBreak || worst == ProvenRegression {
			return 3
		}
	}
	return 0
}

// loadDoc reads and parses a report document. Path "-" reads stdin, so
// `bigo json ./... | bigo diff base.json -` composes.
func loadDoc(path string) (Document, error) {
	data, err := readInput(path)
	if err != nil {
		return Document{}, err
	}
	var d Document
	if err := json.Unmarshal(data, &d); err != nil {
		return Document{}, fmt.Errorf("%s: %w", path, err)
	}
	return d, nil
}
