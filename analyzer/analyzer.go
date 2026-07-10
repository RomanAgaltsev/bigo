// Package analyzer provides the bigo go/analysis Analyzer.
package analyzer

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/callsummary"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/normalize"
)

var reportMode bool

// Analyzer is the bigo complexity analyzer.
var Analyzer = newAnalyzer()

func newAnalyzer() *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name:     "bigo",
		Doc:      "infers asymptotic time complexity and checks //bigo:max budgets",
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
		Run:      run,
	}
	a.Flags.BoolVar(&reportMode, "report", false, "report inferred complexity for every function")
	return a
}

func run(pass *analysis.Pass) (any, error) {
	ssaInfo := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	resolver := callsummary.New()

	byDecl := map[*ast.FuncDecl]*ssa.Function{}
	for _, fn := range ssaInfo.SrcFuncs {
		if decl, ok := fn.Syntax().(*ast.FuncDecl); ok {
			byDecl[decl] = fn
		}
	}

	for _, file := range pass.Files {
		for _, d := range file.Decls {
			decl, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			fn := byDecl[decl]
			if fn == nil {
				continue
			}
			inferred := engine.Infer(fn, resolver)

			if reportMode && !inferred.IsTop() {
				pass.Reportf(decl.Pos(), "inferred complexity %s", inferred.String())
			}
			dir, ok := directiveOf(pass, decl)
			if !ok || dir.Verb != annotation.Max {
				continue
			}
			checkBudget(pass, decl, fn, inferred, dir)
		}
	}
	return nil, nil
}

func checkBudget(pass *analysis.Pass, decl *ast.FuncDecl, fn *ssa.Function, inferred bound.Bound, dir annotation.Directive) {
	budget, err := normalize.Budget(dir, fn)
	if err != nil {
		pass.Reportf(decl.Pos(), "invalid //bigo:max: %v", err)
		return
	}
	switch bound.Check(inferred, budget) {
	case bound.Exceeds:
		pass.Reportf(decl.Pos(), "complexity %s exceeds budget %s", inferred.String(), budget.String())
	case bound.Unknown:
		if inferred.IsTop() {
			pass.Reportf(decl.Pos(), "cannot verify budget %s: unresolved cost in %s", budget.String(), fn.Name())
		} else {
			pass.Reportf(decl.Pos(), "cannot verify budget %s: inferred %s is not comparable", budget.String(), inferred.String())
		}
	case bound.Within:
		// ok
	}
}

// directiveOf returns the first //bigo: directive in the function's doc
// comment. A comment that looks like a directive but fails to parse is
// reported: a silently dropped budget is indistinguishable from a passing
// one, which is the worst failure mode a CI gate can have.
func directiveOf(pass *analysis.Pass, decl *ast.FuncDecl) (annotation.Directive, bool) {
	if decl.Doc == nil {
		return annotation.Directive{}, false
	}
	for _, c := range decl.Doc.List {
		if !strings.HasPrefix(c.Text, "//bigo:") {
			continue
		}
		dir, err := annotation.Parse(c.Text)
		if err != nil {
			pass.Reportf(decl.Pos(), "invalid //bigo: directive: %v", err)
			return annotation.Directive{}, false
		}
		return dir, true
	}
	return annotation.Directive{}, false
}
