// Command bigo runs the bigo complexity analyzer.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/RomanAgaltsev/bigo/analyzer"
	"github.com/RomanAgaltsev/bigo/internal/report"
)

// version is injected by GoReleaser via -X main.version; "dev" locally.
var version = "dev"

// splitChdir extracts a leading `-C dir` (or `-C=dir`) from args, returning the
// directory and the remaining arguments. dir is "" when the flag is absent.
//
// singlechecker owns its flag set and offers no way to set go/packages'
// Config.Dir, so the driver resolves ./... against its own working directory —
// analyzing a module elsewhere otherwise needs an `env -C` / `cd` shim in the
// consuming repo (issue #48). Handling -C here, before delegating, mirrors
// `go -C` and `git -C`.
//
// Like `go`, -C must be the first flag: a late -C is an error rather than a
// silent no-op, since singlechecker would otherwise report the unhelpful
// "flag provided but not defined: -C".
func splitChdir(args []string) (dir string, rest []string, err error) {
	isC := func(s string) (value string, hasValue, ok bool) {
		flag := strings.TrimPrefix(strings.TrimPrefix(s, "-"), "-")
		if flag == "C" {
			return "", false, true
		}
		if v, found := strings.CutPrefix(flag, "C="); found {
			return v, true, true
		}
		return "", false, false
	}
	if len(args) == 0 {
		return "", args, nil
	}
	value, hasValue, ok := isC(args[0])
	if !ok {
		for _, a := range args[1:] {
			if _, _, isFlag := isC(a); isFlag {
				return "", nil, errors.New("-C must be the first flag on the command line")
			}
		}
		return "", args, nil
	}
	if hasValue {
		if value == "" {
			return "", nil, errors.New("-C requires a directory")
		}
		return value, args[1:], nil
	}
	if len(args) < 2 {
		return "", nil, errors.New("-C requires a directory")
	}
	return args[1], args[2:], nil
}

func main() {
	// singlechecker owns the flag set, so handle -version before delegating.
	if len(os.Args) == 2 && (os.Args[1] == "-version" || os.Args[1] == "--version") {
		fmt.Println("bigo " + version)
		return
	}
	// The json subcommand needs one document per run; go/analysis runs per
	// package with no end hook, so it gets its own go/packages driver.
	if len(os.Args) >= 2 && os.Args[1] == "json" {
		os.Exit(report.Main(version, os.Args[2:]))
	}
	// badge projects that same document into a shields.io endpoint badge.
	if len(os.Args) >= 2 && os.Args[1] == "badge" {
		os.Exit(report.BadgeMain(version, os.Args[2:]))
	}
	// diff compares two documents; it analyzes nothing, so it takes no version.
	if len(os.Args) >= 2 && os.Args[1] == "diff" {
		os.Exit(report.DiffMain(os.Args[2:]))
	}
	// -C dir makes ./... resolve against a target module. The subcommands above
	// take their own -C (they own their flag sets); this one is the driver's.
	dir, rest, err := splitChdir(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "bigo:", err)
		os.Exit(2)
	}
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			fmt.Fprintln(os.Stderr, "bigo:", err)
			os.Exit(1)
		}
		os.Args = append(os.Args[:1], rest...)
	}
	singlechecker.Main(analyzer.Analyzer)
}
