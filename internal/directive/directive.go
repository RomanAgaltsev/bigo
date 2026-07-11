// Package directive collects //bigo: directives from a package's syntax and
// turns cost/ignore assertions into resolver inputs. The analyzer and the
// metrics harness both consume it, so diagnostics and coverage numbers can
// never drift apart.
package directive

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/normalize"
	"github.com/RomanAgaltsev/bigo/internal/size"
)

// Reporter receives directive-level problems. The analyzer passes
// pass.Reportf; the metrics harness passes a no-op.
type Reporter func(pos token.Pos, format string, args ...any)

// FuncDirectives is one declaration's parsed directives, in source order.
type FuncDirectives struct {
	Decl *ast.FuncDecl
	Fn   *ssa.Function // nil when SSA has no function for the decl
	Dirs []annotation.Directive
}

// Funcs is everything a caller needs to construct a resolver and run checks.
type Funcs struct {
	Directives  []FuncDirectives
	Plain       []*ast.FuncDecl // decls without directives, still shown by -report
	Overrides   map[*ssa.Function]bound.Bound
	MethodCosts map[*types.Func]bound.Bound
}

// Scan walks the files once, parses every //bigo: directive, reports
// malformed/duplicate/conflicting ones, and materializes cost/ignore
// assertions as resolver inputs.
func Scan(files []*ast.File, info *types.Info, ssaFor func(*ast.FuncDecl) *ssa.Function, report Reporter) Funcs {
	fns := Funcs{
		Overrides:   map[*ssa.Function]bound.Bound{},
		MethodCosts: map[*types.Func]bound.Bound{},
	}
	for _, file := range files {
		for _, d := range file.Decls {
			switch decl := d.(type) {
			case *ast.FuncDecl:
				if dirs := directivesOf(decl, report); len(dirs) > 0 {
					fns.Directives = append(fns.Directives, FuncDirectives{decl, ssaFor(decl), dirs})
				} else {
					fns.Plain = append(fns.Plain, decl)
				}
			case *ast.GenDecl:
				scanInterfaces(decl, info, fns.MethodCosts, report)
			}
		}
	}
	for _, fd := range fns.Directives {
		if fd.Fn == nil {
			continue
		}
		for _, dir := range fd.Dirs {
			switch dir.Verb {
			case annotation.Ignore:
				fns.Overrides[fd.Fn] = bound.Constant()
			case annotation.Cost:
				b, err := normalize.Budget(dir, fd.Fn)
				if err != nil {
					report(fd.Decl.Pos(), "invalid //bigo:cost: %v", err)
					continue
				}
				if hasFieldPathVar(b) {
					report(fd.Decl.Pos(), "//bigo:cost with field-path sizes does not propagate through calls yet; callers remain unverifiable")
				}
				fns.Overrides[fd.Fn] = b
			}
		}
	}
	return fns
}

// Verb returns the directive with verb v, if present. annotation.Max is the
// zero Verb, so callers must consult the boolean, never a zero Directive.
//
//bigo:max O(n) where n=len(dirs)
func Verb(dirs []annotation.Directive, v annotation.Verb) (annotation.Directive, bool) {
	for _, d := range dirs {
		if d.Verb == v {
			return d, true
		}
	}
	return annotation.Directive{}, false
}

// directivesOf returns every //bigo: directive on the function's doc comment,
// in source order. A comment that looks like a directive but fails to parse is
// reported and skipped: a silently dropped budget is indistinguishable from a
// passing one, which is the worst failure mode a CI gate can have. Duplicate
// and conflicting verbs are reported for the same reason — the loser would
// otherwise vanish without a word.
func directivesOf(decl *ast.FuncDecl, report Reporter) []annotation.Directive {
	if decl.Doc == nil {
		return nil
	}
	var dirs []annotation.Directive
	seen := map[annotation.Verb]bool{}
	for _, c := range decl.Doc.List {
		if !strings.HasPrefix(c.Text, "//bigo:") {
			if isNearMiss(c.Text) {
				report(decl.Pos(), nearMissMsg)
			}
			continue
		}
		dir, err := annotation.Parse(c.Text)
		if err != nil {
			report(decl.Pos(), "invalid //bigo: directive: %v", err)
			continue
		}
		if seen[dir.Verb] {
			report(decl.Pos(), "duplicate //bigo:%s directive", dir.Verb)
			continue
		}
		seen[dir.Verb] = true
		dirs = append(dirs, dir)
	}
	if seen[annotation.Cost] && seen[annotation.Ignore] {
		report(decl.Pos(), "//bigo:cost and //bigo:ignore are mutually exclusive")
	}
	return dirs
}

// scanInterfaces materializes //bigo:cost and //bigo:ignore directives on
// interface methods into out. Only cost and ignore apply to methods; any other
// verb is reported. This is the interface-scan loop from analyzer.run, writing
// into a caller-owned map instead of a local.
func scanInterfaces(gd *ast.GenDecl, info *types.Info, out map[*types.Func]bound.Bound, report Reporter) {
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
					if isNearMiss(cmt.Text) {
						report(field.Pos(), nearMissMsg)
					}
					continue
				}
				dir, err := annotation.Parse(cmt.Text)
				if err != nil {
					report(field.Pos(), "invalid //bigo: directive: %v", err)
					continue
				}
				if dir.Verb != annotation.Cost && dir.Verb != annotation.Ignore {
					report(field.Pos(), "only //bigo:cost and //bigo:ignore apply to interface methods")
					continue
				}
				obj, ok := info.Defs[field.Names[0]].(*types.Func)
				if !ok {
					continue
				}
				if dir.Verb == annotation.Ignore {
					out[obj] = bound.Constant()
					continue
				}
				b, err := normalize.BudgetSig(dir, obj.Type().(*types.Signature))
				if err != nil {
					report(field.Pos(), "invalid //bigo:cost: %v", err)
					continue
				}
				if hasFieldPathVar(b) {
					report(field.Pos(), "//bigo:cost with field-path sizes does not propagate through calls yet; callers remain unverifiable")
				}
				out[obj] = b
			}
		}
	}
}

// nearMissMsg is reported for a comment that looks like a //bigo: directive but
// has whitespace after the slashes. A silently dropped budget is
// indistinguishable from a passing one — the worst failure mode for a CI gate —
// so a suspected directive is named rather than ignored.
const nearMissMsg = "//bigo: directive must not have a space after '//'; this looks like a misplaced directive and is being ignored"

// isNearMiss reports whether text is a comment whose body, after the slashes and
// any spaces, begins with "bigo:" — i.e. the //go:-shape prefix with an errant
// space (`// bigo:max`). The exact `//bigo:` prefix is handled by the caller
// before this is consulted, so an exact directive never reaches here.
func isNearMiss(text string) bool {
	body := strings.TrimLeft(text, "/")
	if body == text { // not a // line comment (e.g. a /* block */)
		return false
	}
	return strings.HasPrefix(strings.TrimLeft(body, " \t"), "bigo:")
}

// hasFieldPathVar reports whether the bound references a frame-local
// field-path size (len(s.items) etc.), which cannot propagate to callers yet.
func hasFieldPathVar(b bound.Bound) bool {
	for _, m := range b.Terms() {
		for _, v := range m.Vars() {
			if size.IsFieldPath(v) {
				return true
			}
		}
	}
	return false
}
