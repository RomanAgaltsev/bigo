// Package ssasupport builds SSA from single-file source, for tests and tooling.
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

// Build builds SSA with sanity checks.
func Build(src string) (*ssa.Package, *token.FileSet, error) {
	return buildMode(src, ssa.SanityCheckFunctions)
}

// BuildGeneric builds SSA with generic instantiation enabled.
func BuildGeneric(src string) (*ssa.Package, *token.FileSet, error) {
	return buildMode(src, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
}

// buildMode builds SSA with the given builder mode.
func buildMode(src string, mode ssa.BuilderMode) (*ssa.Package, *token.FileSet, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "input.go", src, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse: %w", err)
	}
	// go/ssa requires Instances to resolve calls to generic functions, and
	// FileVersions to apply per-file language version rules.
	info := &types.Info{
		Types:        map[ast.Expr]types.TypeAndValue{},
		Defs:         map[*ast.Ident]types.Object{},
		Uses:         map[*ast.Ident]types.Object{},
		Implicits:    map[ast.Node]types.Object{},
		Instances:    map[*ast.Ident]types.Instance{},
		Selections:   map[*ast.SelectorExpr]*types.Selection{},
		Scopes:       map[ast.Node]*types.Scope{},
		FileVersions: map[*ast.File]string{},
	}
	conf := types.Config{Importer: importer.Default()}
	tpkg, err := conf.Check("input", fset, []*ast.File{f}, info)
	if err != nil {
		return nil, nil, fmt.Errorf("typecheck: %w", err)
	}
	prog := ssa.NewProgram(fset, mode)
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
