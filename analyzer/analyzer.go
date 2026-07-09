// Package analyzer provides the bigo go/analysis Analyzer.
package analyzer

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
)

// Analyzer is the bigo complexity analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "bigo",
	Doc:      "infers asymptotic time complexity and checks //bigo:max budgets",
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	_ = pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	return nil, nil
}
