// Package analyzer provides the bigo go/analysis Analyzer.
package analyzer

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/callsummary"
	"github.com/RomanAgaltsev/bigo/internal/directive"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/normalize"
	"github.com/RomanAgaltsev/bigo/internal/smell"
)

var reportMode bool

var smellsFlag string

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
	a.Flags.StringVar(&smellsFlag, "smells", "all", "smell rules to run: all, none, or comma-separated (SM1..SM8)")
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

	fns := directive.Scan(pass.Files, pass.TypesInfo, ssaFor, pass.Reportf)
	resolver := callsummary.NewWithMethods(fns.Overrides, fns.MethodCosts)
	spaceResolver := callsummary.NewSpace(nil)

	// Pass 3: infer and check.
	report := func(decl *ast.FuncDecl, fn *ssa.Function) (bound.Bound, []engine.Cause) {
		inferred, causes := resolver.InferTop(fn)
		if reportMode {
			p := pass.Fset.Position(decl.Pos())
			if inferred.IsTop() {
				// Name the unverifiable functions too, with their blocker: they
				// are exactly the ones a user explores -report to find and annotate.
				_, _ = fmt.Fprintf(os.Stdout, "%s:%d: %s: unverifiable — %s\n", p.Filename, p.Line, decl.Name.Name, causeText(pass, causes, fn))
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "%s:%d: %s: inferred complexity %s\n", p.Filename, p.Line, decl.Name.Name, inferred.String())
			}
		}
		return inferred, causes
	}
	for _, decl := range fns.Plain {
		if fn := byDecl[decl]; fn != nil {
			report(decl, fn)
		}
	}
	for _, fd := range fns.Directives {
		if fd.Fn == nil {
			continue
		}
		maxDir, hasMax := directive.Verb(fd.Dirs, annotation.Max)
		inferred, causes := report(fd.Decl, fd.Fn)
		if hasMax {
			checkBudget(pass, fd.Decl, fd.Fn, inferred, causes, maxDir)
		}
		if spaceDir, hasSpace := directive.Verb(fd.Dirs, annotation.Space); hasSpace {
			checkSpace(pass, fd.Decl, fd.Fn, spaceResolver, resolver, spaceDir)
		}
	}

	// Smell pass: advisory complexity smells, firewalled from verdicts. Runs after
	// and independent of the budget pass; //bigo:ignore suppresses smells too.
	if enabled, err := smell.ParseRules(smellsFlag); err != nil {
		return nil, err
	} else if len(enabled) > 0 {
		ignored := map[*ast.FuncDecl]bool{}
		for _, fd := range fns.Directives {
			if _, has := directive.Verb(fd.Dirs, annotation.Ignore); has {
				ignored[fd.Decl] = true
			}
		}
		// Iterate decls in source-position order: byDecl is a map, and emitting
		// diagnostics in its randomized iteration order would make the smell
		// output nondeterministic run to run.
		decls := make([]*ast.FuncDecl, 0, len(byDecl))
		for decl := range byDecl {
			decls = append(decls, decl)
		}
		sort.Slice(decls, func(i, j int) bool { return decls[i].Pos() < decls[j].Pos() })
		for _, decl := range decls {
			if ignored[decl] {
				continue
			}
			for _, f := range smell.Detect(byDecl[decl], enabled) {
				pass.Reportf(f.Pos, "smell(%s): %s", f.Rule, f.Message)
			}
		}
	}
	return nil, nil
}

// checkSpace verifies a //bigo:space budget. Heap is an upper bound on peak
// (total allocated) so it proves Within only; stack (recursion depth) is a true
// peak and proves both verdicts. spaceVerdict enforces that asymmetry, so a
// space budget can never produce a false Exceeds.
func checkSpace(pass *analysis.Pass, decl *ast.FuncDecl, fn *ssa.Function, spaceResolver *callsummary.SpaceResolver, timeModel engine.CostModel, dir annotation.Directive) {
	sp, causes := spaceResolver.SpaceOf(fn, timeModel)
	inferred := sp.Heap.Join(sp.Stack)
	if reportMode {
		p := pass.Fset.Position(decl.Pos())
		_, _ = fmt.Fprintf(os.Stdout, "%s:%d: %s: space %s\n", p.Filename, p.Line, decl.Name.Name, inferred.String())
	}
	budget, err := normalize.Budget(dir, fn)
	if err != nil {
		pass.Reportf(decl.Pos(), "invalid //bigo:space: %v", err)
		return
	}
	switch engine.SpaceVerdict(sp, budget) {
	case bound.Exceeds:
		pass.Reportf(decl.Pos(), "space %s exceeds budget %s", inferred.String(), budget.String())
	case bound.Unknown:
		pass.Reportf(decl.Pos(), "cannot verify space budget %s: %s", budget.String(), spaceCause(pass, sp, causes, fn))
	case bound.Within:
		// ok
	}
}

// spaceCause explains an Unknown space verdict. When both heap and stack are
// fully known (not ⊤), the budget failed only because heap is a total-allocation
// upper bound that cannot prove a smaller peak — so report the known space, not
// causeText's misleading "unresolved cost" (there is no unresolved call). A
// genuinely ⊤ sub-bound (unknown make length, unresolved callee) keeps the
// causeText annotate hint.
func spaceCause(pass *analysis.Pass, sp engine.Space, causes []engine.Cause, fn *ssa.Function) string {
	if !sp.Heap.IsTop() && !sp.Stack.IsTop() {
		return fmt.Sprintf("inferred space %s is a total-allocation upper bound and cannot prove a smaller peak", sp.Heap.Join(sp.Stack).String())
	}
	return causeText(pass, causes, fn)
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
