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
	"github.com/RomanAgaltsev/bigo/internal/trust"
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

// withKataTestDefault injects -test=false when -kata is given and -test is not,
// returning args unchanged otherwise.
//
// singlechecker loads with Tests: true, so go/packages returns both `p` and
// `p [p.test]` as initial packages and the analyzer's -report pass, which
// prints to stdout rather than emitting diagnostics, runs once per package —
// reporting every function of the solution TWICE. It also reports the test
// functions and the generated test main, whose file name is a build-cache path.
// Diagnostics are unaffected: x/tools deduplicates identical ones, and
// `bigo json`/`diff`/`badge` load without Tests at all.
//
// A kata's test file is not the solution and is never budgeted, so -kata
// choosing a different default is the honest behaviour rather than a
// convenience. An explicit -test always wins, in either direction.
//
// The flag is inserted at the FRONT: flag parsing stops at the first package
// pattern, so appending it would leave it parsed as a pattern instead.
func withKataTestDefault(args []string) []string {
	name := func(s string) (flag string, ok bool) {
		if s == "--" || !strings.HasPrefix(s, "-") {
			return "", false
		}
		flag = strings.TrimPrefix(strings.TrimPrefix(s, "-"), "-")
		flag, _, _ = strings.Cut(flag, "=")
		return flag, true
	}
	kata := false
	for _, a := range args {
		flag, ok := name(a)
		if !ok {
			break // a package pattern: flag parsing stops here, and so do we
		}
		switch flag {
		case "test":
			return args // an explicit -test wins, whichever way it points
		case "kata":
			// -kata=false turns the mode off, so it must not change the default.
			kata = !strings.HasSuffix(a, "=false") && !strings.HasSuffix(a, "=0")
		}
	}
	if !kata {
		return args
	}
	out := make([]string, 0, len(args)+1)
	out = append(out, "-test=false")
	return append(out, args...)
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
	// trust init scaffolds a trust file for the module in -C: the keys blocking
	// the most of your functions, commented out, for you to fill in.
	if len(os.Args) >= 3 && os.Args[1] == "trust" && os.Args[2] == "init" {
		os.Exit(trust.InitMain(os.Args[3:]))
	}
	if len(os.Args) >= 2 && os.Args[1] == "trust" {
		fmt.Fprintln(os.Stderr, "usage: bigo trust init [-C dir] [-o file] [-force]")
		os.Exit(2)
	}
	// A subcommand name appearing AFTER the driver's own flags is a flag-order
	// mistake, not a package pattern. `bigo -C dir trust init` is the natural
	// way to write it and used to fall through to the analyzer, which took
	// "trust" and "init" as packages and failed with "package trust is not in
	// std" — an error naming neither the subcommand nor the flag. Found by
	// running this lane's own discovery pass across six repositories before
	// noticing none of them had produced output (2026-08-12 blind-repo lane 2).
	for i, a := range os.Args[1:] {
		if a != "trust" && a != "json" && a != "diff" && a != "badge" {
			continue
		}
		if i == 0 {
			break // already dispatched above
		}
		fmt.Fprintf(os.Stderr, "bigo: %q is a subcommand and must come first: bigo %s [flags]\n"+
			"       the driver's own flags apply to analysis, not to subcommands, which own their -C\n", a, a)
		os.Exit(2)
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
	}
	os.Args = append(os.Args[:1], withKataTestDefault(rest)...)
	singlechecker.Main(analyzer.Analyzer)
}
