package report

import (
	"encoding/json"
	"flag"
	"fmt"
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
