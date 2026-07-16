// Command bigo runs the bigo complexity analyzer.
package main

import (
	"fmt"
	"os"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/RomanAgaltsev/bigo/analyzer"
	"github.com/RomanAgaltsev/bigo/internal/report"
)

// version is injected by GoReleaser via -X main.version; "dev" locally.
var version = "dev"

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
	singlechecker.Main(analyzer.Analyzer)
}
