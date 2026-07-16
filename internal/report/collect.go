package report

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"time"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/callsummary"
	"github.com/RomanAgaltsev/bigo/internal/directive"
	"github.com/RomanAgaltsev/bigo/internal/engine"
	"github.com/RomanAgaltsev/bigo/internal/smell"
)

// Options configures a Collect run. Version fills the envelope's bigo_version;
// Now is injectable for deterministic goldens and defaults to time.Now.
type Options struct {
	Version string
	Now     func() time.Time
}

// Collect analyzes the module at dir (patterns as for `go build`, default
// ./...) and returns the report document. Analysis is the same InferTop /
// SpaceOf pipeline the analyzer runs; Collect adds no inference of its own.
func Collect(dir string, patterns []string, opts Options) (Document, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps |
			packages.NeedImports | packages.NeedModule,
		Dir: dir,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return Document{}, err
	}
	for _, p := range pkgs {
		if len(p.Errors) > 0 {
			return Document{}, fmt.Errorf("package %s: %v", p.PkgPath, p.Errors[0])
		}
	}
	prog, _ := ssautil.Packages(pkgs, ssa.BuilderMode(0))
	prog.Build()

	doc := Document{
		SchemaVersion: SchemaVersion,
		BigoVersion:   opts.Version,
		Generated:     now().UTC().Format(time.RFC3339),
		Functions:     []Function{},
	}
	root := ""
	for _, p := range pkgs {
		if p.Module != nil {
			doc.Module = p.Module.Path
			root = p.Module.Dir
			break
		}
	}

	// All rules always run: the document describes the module completely and
	// consumers filter. ParseRules("all") cannot fail for a literal.
	enabledSmells, err := smell.ParseRules("all")
	if err != nil {
		return Document{}, err
	}

	nop := func(token.Pos, string, ...any) {}
	for _, p := range pkgs {
		ssaFor := func(decl *ast.FuncDecl) *ssa.Function {
			obj, ok := p.TypesInfo.Defs[decl.Name].(*types.Func)
			if !ok {
				return nil
			}
			return prog.FuncValue(obj)
		}
		fns := directive.Scan(p.Syntax, p.TypesInfo, ssaFor, nop)
		resolver := callsummary.NewWithMethods(fns.Overrides, fns.MethodCosts)
		spaceResolver := callsummary.NewSpace(nil)

		// Ignored decls are skipped for smells exactly as for verdicts, so the
		// document never contradicts the analyzer (metrics.go:95-100 does the same).
		ignored := map[*ast.FuncDecl]bool{}
		for _, fd := range fns.Directives {
			if _, has := directive.Verb(fd.Dirs, annotation.Ignore); has {
				ignored[fd.Decl] = true
			}
		}

		measure := func(decl *ast.FuncDecl, dirs []annotation.Directive) {
			fn := ssaFor(decl)
			if fn == nil {
				return
			}
			pos := p.Fset.Position(decl.Pos())
			rec := Function{
				Package:  p.PkgPath,
				Func:     decl.Name.Name,
				Receiver: recvString(decl),
				File:     relPath(root, pos.Filename),
				Line:     pos.Line,
			}
			inferred, causes := resolver.InferTop(fn)
			rec.Time = boundJSON(inferred)
			if inferred.IsTop() {
				for _, c := range causes {
					cj := CauseJSON{Kind: c.Kind.String(), Detail: c.What}
					if c.Pos.IsValid() {
						cp := p.Fset.Position(c.Pos)
						cj.File = relPath(root, cp.Filename)
						cj.Line = cp.Line
					}
					rec.Causes = append(rec.Causes, cj)
				}
			}
			if maxDir, ok := directive.Verb(dirs, annotation.Max); ok {
				rec.Budget = budgetJSON(maxDir, fn, func(budget bound.Bound) bound.Verdict {
					return bound.Check(inferred, budget)
				})
			}
			if spDir, ok := directive.Verb(dirs, annotation.Space); ok {
				sp, _ := spaceResolver.SpaceOf(fn, resolver)
				sj := SpaceJSON{Heap: boundJSON(sp.Heap), Stack: boundJSON(sp.Stack)}
				sj.Budget = budgetJSON(spDir, fn, func(budget bound.Bound) bound.Verdict {
					return engine.SpaceVerdict(sp, budget)
				})
				rec.Space = &sj
			}
			for _, d := range dirs {
				if d.Verb == annotation.Cost || d.Verb == annotation.Ignore {
					rec.Trust = append(rec.Trust, d.Raw)
					doc.Trusted = append(doc.Trusted, TrustEntry{
						Package:   p.PkgPath,
						Func:      decl.Name.Name,
						Receiver:  rec.Receiver,
						Directive: d.Raw,
					})
				}
			}
			if !ignored[decl] {
				for _, sf := range smell.Detect(fn, enabledSmells) {
					sj := SmellJSON{Rule: sf.Rule, Message: sf.Message}
					if sf.Pos.IsValid() {
						sp := p.Fset.Position(sf.Pos)
						sj.File = relPath(root, sp.Filename)
						sj.Line = sp.Line
					}
					doc.Smells = append(doc.Smells, sj)
				}
			}
			doc.Functions = append(doc.Functions, rec)
		}

		for _, fd := range fns.Directives {
			measure(fd.Decl, fd.Dirs)
		}
		for _, decl := range fns.Plain {
			measure(decl, nil)
		}
	}

	sort.Slice(doc.Functions, func(i, j int) bool {
		a, b := doc.Functions[i], doc.Functions[j]
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	sort.Slice(doc.Trusted, func(i, j int) bool {
		a, b := doc.Trusted[i], doc.Trusted[j]
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		if a.Receiver != b.Receiver {
			return a.Receiver < b.Receiver
		}
		return a.Func < b.Func
	})
	// Sorted by file/line/rule rather than rule-first: the document's other
	// arrays sort positionally, and a human triaging a scan reads by location.
	sort.Slice(doc.Smells, func(i, j int) bool {
		a, b := doc.Smells[i], doc.Smells[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Rule < b.Rule
	})
	return doc, nil
}

// recvString renders a method's receiver type as written, e.g. "*Tree".
func recvString(decl *ast.FuncDecl) string {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return ""
	}
	return types.ExprString(decl.Recv.List[0].Type)
}

// relPath makes file positions module-relative with forward slashes; absolute
// paths never appear in a document.
func relPath(root, file string) string {
	if root != "" {
		if r, err := filepath.Rel(root, file); err == nil {
			return filepath.ToSlash(r)
		}
	}
	return filepath.ToSlash(file)
}
