// Package ssasupport builds SSA from single-file source, for test and tooling.
package ssasupport

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// Build type-checks and builds SSA for a single-file source program.
func Build(src string) (*ssa.Package, *token.FileSet, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "input.go", src, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse: %w", err)
	}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Implicits:  map[ast.Node]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     map[ast.Node]*types.Scope{},
	}
	conf := types.Config{Importer: importer.Default()}
	tpkg, err := conf.Check("input", fset, []*ast.File{f}, info)
	if err != nil {
		return nil, nil, fmt.Errorf("typecheck: %w", err)
	}
	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	for _, imp := range tpkg.Imports() {
		createAll(prog, imp)
	}
	ssaPkg := prog.CreatePackage(tpkg, []*ast.File{f}, info, false)
	ssaPkg.Build()
	return ssaPkg, fset, nil
}

func createAll(prog *ssa.Program, pkg *types.Package) {
	if prog.Package(pkg) != nil {
		return
	}
	prog.CreatePackage(pkg, nil, nil, true)
	for _, imp := range pkg.Imports() {
		createAll(prog, imp)
	}
}

// Func returns the named top-level function or nil.
func Func(pkg *ssa.Package, name string) *ssa.Function {
	if fn, ok := pkg.Members[name].(*ssa.Function); ok {
		return fn
	}
	return nil
}
