// Command bigo runs the bigo complexity analyzer.
package main

import (
	"fmt"
	"os"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/RomanAgaltsev/bigo/analyzer"
)

// version is injected by GoReleaser via -X main.version; "dev" locally.
var version = "dev"

func main() {
	// singlechecker owns the flag set, so handle -version before delegating.
	if len(os.Args) == 2 && (os.Args[1] == "-version" || os.Args[1] == "--version") {
		fmt.Println("bigo " + version)
		return
	}
	singlechecker.Main(analyzer.Analyzer)
}
