// Command katadiff prints the kata-model bound and cause for every function in
// a directory, so a reduced corpus fixture can be checked against the original
// it was reduced from.
//
// A development tool, not part of the shipped analyzer. It exists because "the
// reduction did not change the verdict" is a measurable claim, and checking it
// by eye across fourteen files is how a silent fixture drift enters a
// regression net at the moment it is created.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"sort"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/RomanAgaltsev/bigo/internal/callsummary"
	"github.com/RomanAgaltsev/bigo/internal/directive"
	"github.com/RomanAgaltsev/bigo/internal/kata"
)

func main() {
	dir := flag.String("dir", "", "directory to analyze")
	only := flag.String("func", "", "only this function name (default: all)")
	flag.Parse()
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "katadiff: -dir is required")
		os.Exit(2)
	}
	if err := run(*dir, *only); err != nil {
		fmt.Fprintln(os.Stderr, "katadiff:", err)
		os.Exit(1)
	}
}

func run(dir, only string) error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports,
		Dir: dir,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("loaded 0 packages under %s", dir)
	}

	prog, _ := ssautil.Packages(pkgs, ssa.BuilderMode(0))
	prog.Build()

	profile, err := kata.Profile()
	if err != nil {
		return err
	}
	spaceProfile, err := kata.SpaceProfile()
	if err != nil {
		return err
	}

	nop := func(token.Pos, string, ...any) {}
	var lines []string
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
		resolver.UseOverlay(profile)
		spaceResolver := callsummary.NewSpace(nil)
		spaceResolver.UseOverlay(spaceProfile)

		for _, file := range p.Syntax {
			for _, decl := range file.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if only != "" && fd.Name.Name != only {
					continue
				}
				fn := ssaFor(fd)
				if fn == nil {
					continue
				}
				b, causes := resolver.InferTop(fn)
				sp, _ := spaceResolver.SpaceOf(fn, resolver)
				cause := ""
				if len(causes) > 0 {
					cause = causes[0].Kind.String()
				}
				lines = append(lines, fmt.Sprintf("%s time=%s space=%s cause=%s",
					fd.Name.Name, b.String(), sp.Heap.Join(sp.Stack).String(), cause))
			}
		}
	}
	sort.Strings(lines)
	for _, l := range lines {
		fmt.Println(l)
	}
	return nil
}
