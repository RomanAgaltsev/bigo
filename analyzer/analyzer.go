// Package analyzer provides the bigo go/analysis Analyzer.
package analyzer

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
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

	// Pass 1: collect every directive once (directivesOf reports parse errors,
	// duplicates and conflicts, so it must not run twice per decl).
	type funcDirectives struct {
		decl *ast.FuncDecl
		fn   *ssa.Function
		dirs []annotation.Directive
	}
	var directives []funcDirectives
	var plain []*ast.FuncDecl // decls without directives, still shown by -report
	for _, file := range pass.Files {
		for _, d := range file.Decls {
			decl, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if dirs := directivesOf(pass, decl); len(dirs) > 0 {
				directives = append(directives, funcDirectives{decl, ssaFor(decl), dirs})
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
		for _, dir := range fd.dirs {
			switch dir.Verb {
			case annotation.Ignore:
				overrides[fd.fn] = bound.Constant()
			case annotation.Cost:
				b, err := normalize.Budget(dir, fd.fn)
				if err != nil {
					pass.Reportf(fd.decl.Pos(), "invalid //bigo:cost: %v", err)
					continue
				}
				overrides[fd.fn] = b
			}
		}
	}
	methodCosts := map[*types.Func]bound.Bound{}
	for _, file := range pass.Files {
		for _, d := range file.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				it, ok := ts.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				for _, field := range it.Methods.List {
					if field.Doc == nil || len(field.Names) == 0 {
						continue
					}
					for _, cmt := range field.Doc.List {
						if !strings.HasPrefix(cmt.Text, "//bigo:") {
							continue
						}
						dir, err := annotation.Parse(cmt.Text)
						if err != nil {
							pass.Reportf(field.Pos(), "invalid //bigo: directive: %v", err)
							continue
						}
						if dir.Verb != annotation.Cost && dir.Verb != annotation.Ignore {
							pass.Reportf(field.Pos(), "only //bigo:cost and //bigo:ignore apply to interface methods")
							continue
						}
						obj, ok := pass.TypesInfo.Defs[field.Names[0]].(*types.Func)
						if !ok {
							continue
						}
						if dir.Verb == annotation.Ignore {
							methodCosts[obj] = bound.Constant()
							continue
						}
						b, err := normalize.BudgetSig(dir, obj.Type().(*types.Signature))
						if err != nil {
							pass.Reportf(field.Pos(), "invalid //bigo:cost: %v", err)
							continue
						}
						methodCosts[obj] = b
					}
				}
			}
		}
	}
	resolver := callsummary.NewWithMethods(overrides, methodCosts)

	// Pass 3: infer and check.
	report := func(decl *ast.FuncDecl, fn *ssa.Function) (bound.Bound, []engine.Cause) {
		inferred, causes := engine.InferDetailed(fn, resolver)
		if reportMode && !inferred.IsTop() {
			p := pass.Fset.Position(decl.Pos())
			_, _ = fmt.Fprintf(os.Stdout, "%s:%d: %s: inferred complexity %s\n", p.Filename, p.Line, decl.Name.Name, inferred.String())
		}
		return inferred, causes
	}
	for _, decl := range plain {
		if fn := byDecl[decl]; fn != nil {
			report(decl, fn)
		}
	}
	for _, fd := range directives {
		if fd.fn == nil {
			continue
		}
		maxDir, hasMax := verb(fd.dirs, annotation.Max)
		inferred, causes := report(fd.decl, fd.fn)
		if hasMax {
			checkBudget(pass, fd.decl, fd.fn, inferred, causes, maxDir)
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
			pass.Reportf(decl.Pos(), "cannot verify budget %s: %s (annotate the callee with //bigo:cost or //bigo:ignore)", budget.String(), causeText(pass, causes, fn))
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

// directivesOf returns every //bigo: directive on the function's doc comment,
// in source order. A comment that looks like a directive but fails to parse is
// reported and skipped: a silently dropped budget is indistinguishable from a
// passing one, which is the worst failure mode a CI gate can have. Duplicate
// and conflicting verbs are reported for the same reason — the loser would
// otherwise vanish without a word.
func directivesOf(pass *analysis.Pass, decl *ast.FuncDecl) []annotation.Directive {
	if decl.Doc == nil {
		return nil
	}
	var dirs []annotation.Directive
	seen := map[annotation.Verb]bool{}
	for _, c := range decl.Doc.List {
		if !strings.HasPrefix(c.Text, "//bigo:") {
			continue
		}
		dir, err := annotation.Parse(c.Text)
		if err != nil {
			pass.Reportf(decl.Pos(), "invalid //bigo: directive: %v", err)
			continue
		}
		if seen[dir.Verb] {
			pass.Reportf(decl.Pos(), "duplicate //bigo:%s directive", dir.Verb)
			continue
		}
		seen[dir.Verb] = true
		dirs = append(dirs, dir)
	}
	if seen[annotation.Cost] && seen[annotation.Ignore] {
		pass.Reportf(decl.Pos(), "//bigo:cost and //bigo:ignore are mutually exclusive")
	}
	return dirs
}

// verb returns the directive with verb v, if the declaration carries one.
// annotation.Max is the zero value of annotation.Verb, so callers must consult
// the boolean rather than comparing against a zero Directive.
func verb(dirs []annotation.Directive, v annotation.Verb) (annotation.Directive, bool) {
	for _, d := range dirs {
		if d.Verb == v {
			return d, true
		}
	}
	return annotation.Directive{}, false
}
