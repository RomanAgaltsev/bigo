// Package analyzer provides the bigo go/analysis Analyzer.
package analyzer

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
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

	byDecl := map[*ast.FuncDecl]*ssa.Function{}
	for _, fn := range ssaInfo.SrcFuncs {
		if decl, ok := fn.Syntax().(*ast.FuncDecl); ok {
			byDecl[decl] = fn
		}
	}
	// ssaFor also resolves bodyless declarations (assembly/external), which
	// SrcFuncs omits — //bigo:cost on those is the headline use case.
	ssaFor := func(decl *ast.FuncDecl) *ssa.Function {
		if fn := byDecl[decl]; fn != nil {
			return fn
		}
		if obj, ok := pass.TypesInfo.Defs[decl.Name].(*types.Func); ok {
			return ssaInfo.Pkg.Prog.FuncValue(obj)
		}
		return nil
	}

	// Pass 1: collect every directive once (directiveOf reports parse errors,
	// so it must not run twice per decl).
	type funcDirective struct {
		decl *ast.FuncDecl
		fn   *ssa.Function
		dir  annotation.Directive
	}
	var directives []funcDirective
	var plain []*ast.FuncDecl // decls without directives, still shown by -report
	for _, file := range pass.Files {
		for _, d := range file.Decls {
			decl, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if dir, ok := directiveOf(pass, decl); ok {
				directives = append(directives, funcDirective{decl, ssaFor(decl), dir})
			} else {
				plain = append(plain, decl)
			}
		}
	}

	// Pass 2: cost/ignore assertions become resolver overrides.
	overrides := map[*ssa.Function]bound.Bound{}
	for _, fd := range directives {
		if fd.fn == nil {
			continue
		}
		switch fd.dir.Verb {
		case annotation.Ignore:
			overrides[fd.fn] = bound.Constant()
		case annotation.Cost:
			b, err := normalize.Budget(fd.dir, fd.fn)
			if err != nil {
				pass.Reportf(fd.decl.Pos(), "invalid //bigo:cost: %v", err)
				continue
			}
			overrides[fd.fn] = b
		}
	}
	resolver := callsummary.New(overrides)

	// Pass 3: infer and check.
	report := func(decl *ast.FuncDecl, fn *ssa.Function) (bound.Bound, []engine.Cause) {
		inferred, causes := engine.InferDetailed(fn, resolver)
		if reportMode && !inferred.IsTop() {
			pass.Reportf(decl.Pos(), "inferred complexity %s", inferred.String())
		}
		return inferred, causes
	}
	for _, decl := range plain {
		if fn := byDecl[decl]; fn != nil {
			report(decl, fn)
		}
	}
	for _, fd := range directives {
		if fd.fn == nil || len(fd.fn.Blocks) == 0 {
			continue // bodyless: nothing to check; its directive already feeds overrides
		}
		inferred, causes := report(fd.decl, fd.fn)
		if fd.dir.Verb == annotation.Max {
			checkBudget(pass, fd.decl, fd.fn, inferred, causes, fd.dir)
		}
	}
	return nil, nil
}

func checkBudget(pass *analysis.Pass, decl *ast.FuncDecl, fn *ssa.Function, inferred bound.Bound, causes []engine.Cause, dir annotation.Directive) {
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
			pass.Reportf(decl.Pos(), "cannot verify budget %s: %s", budget.String(), causeText(pass, causes, fn))
		} else {
			pass.Reportf(decl.Pos(), "cannot verify budget %s: inferred %s is not comparable", budget.String(), inferred.String())
		}
	case bound.Within:
		// ok
	}
}

// causeText names the first blocker with its position — spec §5's "name the
// exact unresolved node".
func causeText(pass *analysis.Pass, causes []engine.Cause, fn *ssa.Function) string {
	if len(causes) == 0 {
		return "unresolved cost in " + fn.Name()
	}
	c := causes[0]
	if !c.Pos.IsValid() {
		return c.What
	}
	p := pass.Fset.Position(c.Pos)
	return fmt.Sprintf("%s (%s:%d)", c.What, filepath.Base(p.Filename), p.Line)
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
