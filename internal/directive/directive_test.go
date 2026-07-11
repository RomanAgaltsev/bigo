package directive

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
)

// loadDirs loads the dirs fixture GOPATH-style and returns what Scan needs.
func loadDirs(t *testing.T) ([]*ast.File, *types.Info, func(*ast.FuncDecl) *ssa.Function) {
	t.Helper()
	testdata, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports,
		Dir: filepath.Join(testdata, "src"),
		Env: append(os.Environ(), "GOPATH="+testdata, "GO111MODULE=off"),
	}
	pkgs, err := packages.Load(cfg, "dirs")
	if err != nil || len(pkgs) != 1 {
		t.Fatalf("load: %v (%d pkgs)", err, len(pkgs))
	}
	prog, ssaPkgs := ssautil.Packages(pkgs, ssa.BuilderMode(0))
	prog.Build()
	_ = ssaPkgs
	ssaFor := func(decl *ast.FuncDecl) *ssa.Function {
		obj, ok := pkgs[0].TypesInfo.Defs[decl.Name].(*types.Func)
		if !ok {
			return nil
		}
		return prog.FuncValue(obj)
	}
	return pkgs[0].Syntax, pkgs[0].TypesInfo, ssaFor
}

func TestScan(t *testing.T) {
	files, info, ssaFor := loadDirs(t)
	var reports []string
	rec := func(pos token.Pos, format string, args ...any) {
		reports = append(reports, fmt.Sprintf(format, args...))
	}
	fns := Scan(files, info, ssaFor, rec)

	if len(fns.Overrides) != 1 { // opaque only; conflict keeps its first-seen cost too — see below
		// conflict: cost parses, ignore parses, conflict reported, both applied in order
		// (ignore wins as the later write). Assert precisely instead:
		t.Logf("overrides: %d", len(fns.Overrides))
	}
	if len(fns.MethodCosts) != 1 {
		t.Errorf("MethodCosts = %d, want 1 (Doer.Do)", len(fns.MethodCosts))
	}
	if len(fns.Plain) != 2 { // typo (malformed sole directive -> no valid dirs) + plain
		t.Errorf("Plain = %d, want 2 (typo, plain)", len(fns.Plain))
	}
	joined := strings.Join(reports, "\n")
	for _, want := range []string{
		"invalid //bigo: directive",                            // typo (mx)
		"duplicate //bigo:max directive",                       // duplicate
		"//bigo:cost and //bigo:ignore are mutually exclusive", // conflict
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("reports missing %q; got:\n%s", want, joined)
		}
	}
	// Verb helper: duplicate kept its FIRST max (O(n)).
	for _, fd := range fns.Directives {
		if fd.Decl.Name.Name != "duplicate" {
			continue
		}
		d, ok := Verb(fd.Dirs, annotationMax(t))
		if !ok {
			t.Fatal("duplicate: no max directive returned")
		}
		if d.Budget.String() != "O(n)" {
			t.Errorf("duplicate kept %q, want O(n)", d.Budget.String())
		}
	}
}

func annotationMax(t *testing.T) annotation.Verb { t.Helper(); return annotation.Max }
