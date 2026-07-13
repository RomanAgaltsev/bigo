// Package metrics computes verdict coverage over the analysistest corpora and
// renders it as a committed golden (metrics/metrics.json) plus a
// human-readable table (metrics/METRICS.md). Silence is a verdict: every
// source function is counted, budgeted or not.
package metrics

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/RomanAgaltsev/bigo/internal/callsummary"
	"github.com/RomanAgaltsev/bigo/internal/directive"
)

// FuncMetric is one function's verdict.
type FuncMetric struct {
	Pkg     string `json:"pkg"`
	Func    string `json:"func"`
	Verdict string `json:"verdict"` // "bounded" | "unverifiable"
	Bound   string `json:"bound,omitempty"`
	Cause   string `json:"cause,omitempty"` // CauseKind of causes[0]
}

// PkgCount aggregates one package.
type PkgCount struct {
	Total   int `json:"total"`
	Bounded int `json:"bounded"`
}

// Report is the full coverage picture. Deterministic by construction: no
// timestamps, no toolchain versions, no absolute paths, sorted everything.
type Report struct {
	Total       int                 `json:"total"`
	Bounded     int                 `json:"bounded"`
	CoveragePct string              `json:"coverage_pct"`
	ByCause     map[string]int      `json:"by_cause"`
	PerPackage  map[string]PkgCount `json:"per_package"`
	Functions   []FuncMetric        `json:"functions"`
}

// Collect analyzes every package under srcRoot (a GOPATH-shaped src dir).
func Collect(srcRoot string) (Report, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports,
		Dir: srcRoot,
		Env: append(os.Environ(), "GOPATH="+filepath.Dir(srcRoot), "GO111MODULE=off"),
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return Report{}, err
	}
	for _, p := range pkgs {
		if len(p.Errors) > 0 {
			return Report{}, fmt.Errorf("package %s: %v", p.PkgPath, p.Errors[0])
		}
	}
	prog, _ := ssautil.Packages(pkgs, ssa.BuilderMode(0))
	prog.Build()

	r := Report{
		ByCause:    map[string]int{},
		PerPackage: map[string]PkgCount{},
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

		measure := func(decl *ast.FuncDecl) {
			fn := ssaFor(decl)
			if fn == nil {
				return
			}
			m := FuncMetric{Pkg: p.PkgPath, Func: decl.Name.Name}
			b, causes := resolver.InferTop(fn)
			if b.IsTop() {
				m.Verdict = "unverifiable"
				if len(causes) > 0 {
					m.Cause = causes[0].Kind.String()
				}
			} else {
				m.Verdict = "bounded"
				m.Bound = b.String()
			}
			r.Functions = append(r.Functions, m)
			pc := r.PerPackage[p.PkgPath]
			pc.Total++
			r.Total++
			if m.Verdict == "bounded" {
				pc.Bounded++
				r.Bounded++
			} else {
				r.ByCause[m.Cause]++
			}
			r.PerPackage[p.PkgPath] = pc
		}
		for _, fd := range fns.Directives {
			measure(fd.Decl)
		}
		for _, decl := range fns.Plain {
			measure(decl)
		}
	}
	sort.Slice(r.Functions, func(i, j int) bool {
		if r.Functions[i].Pkg != r.Functions[j].Pkg {
			return r.Functions[i].Pkg < r.Functions[j].Pkg
		}
		return r.Functions[i].Func < r.Functions[j].Func
	})
	if r.Total > 0 {
		r.CoveragePct = fmt.Sprintf("%.1f", 100*float64(r.Bounded)/float64(r.Total))
	} else {
		r.CoveragePct = "0.0"
	}
	return r, nil
}
