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
	rec := func(_ token.Pos, format string, args ...any) {
		reports = append(reports, fmt.Sprintf(format, args...))
	}
	fns := Scan(files, info, ssaFor, rec)

	// overrides: opaque, conflict (cost then ignore both write the same fn key),
	// and fieldCost — 3 entries. Soft-logged: conflict's dual-write ordering is
	// the subtle part, so the count is informational rather than pinned.
	if len(fns.Overrides) != 3 {
		t.Logf("overrides: %d", len(fns.Overrides))
	}
	if len(fns.MethodCosts) != 2 {
		t.Errorf("MethodCosts = %d, want 2 (Doer.Do, Scanner.Scan)", len(fns.MethodCosts))
	}
	if len(fns.Plain) != 2 { // typo (malformed sole directive -> no valid dirs) + plain
		t.Errorf("Plain = %d, want 2 (typo, plain)", len(fns.Plain))
	}
	joined := strings.Join(reports, "\n")
	for _, want := range []string{
		"invalid //bigo: directive",                            // typo (mx)
		"duplicate //bigo:max directive",                       // duplicate
		"//bigo:cost and //bigo:ignore are mutually exclusive", // conflict
		"//bigo:cost with field-path sizes does not propagate through calls yet",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("reports missing %q; got:\n%s", want, joined)
		}
	}
	gap := 0
	for _, r := range reports {
		if strings.Contains(r, "does not propagate through calls yet") {
			gap++
		}
	}
	if gap != 2 {
		t.Errorf("propagation-gap diagnostic reported %d times, want 2 (func + interface method)", gap)
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
