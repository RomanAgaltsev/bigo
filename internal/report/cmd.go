package report

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
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
// is "-", so `bigo json ./... | bigo badge -i -` composes.
func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
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
