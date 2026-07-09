// Command bigo runs the bigo complexity analyzer.
package main

import (
	"github.com/RomanAgaltsev/bigo/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(analyzer.Analyzer) }
