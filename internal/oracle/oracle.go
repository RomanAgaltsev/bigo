package oracle

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
	"github.com/RomanAgaltsev/bigo/internal/normalize"
)

// Entry is one corpus function's golden row. All bounds are rendered strings;
// the golden is a document, not an API.
type Entry struct {
	Pkg         string `json:"pkg"`
	Func        string `json:"func"`
	TimePin     string `json:"time_pin"`
	TimeGot     string `json:"time_got"`
	TimeStatus  string `json:"time_status"`
	SpacePin    string `json:"space_pin,omitempty"`
	SpaceGot    string `json:"space_got,omitempty"`
	SpaceStatus string `json:"space_status,omitempty"`
	Cause       string `json:"cause,omitempty"` // CauseKind of causes[0] when time is top
	Source      string `json:"source"`
}

// WrongBound is a prime-directive break: an emitted bound that does not
// dominate its pin. Wrongs are returned separately and never rendered into a
// golden.
//
// Named WrongBound, not Wrong: the plan's `type Wrong struct` collided with the
// Wrong Status constant in classify.go. Status is the one the tests and Classify
// name, so the struct took the new name.
type WrongBound struct {
	Pkg, Func, Dim, Pin, Got string
}

// Report is the golden document. Deterministic by construction: sorted
// entries, sorted map keys (encoding/json), no timestamps, no absolute paths.
type Report struct {
	Total         int            `json:"total"`
	TimeByStatus  map[string]int `json:"time_by_status"`
	SpaceByStatus map[string]int `json:"space_by_status"`
	PerFamily     map[string]int `json:"per_family"`
	Entries       []Entry        `json:"entries"`
}

// Collect analyzes every pinned function under srcRoot (GOPATH-shaped, like
// the metrics corpus) with the shipped, unaided InferTop/SpaceOf pipeline and
// classifies each dimension against its pin. Reconciliation is asserted, not
// logged: a pinned function that fails to load, resolve to SSA, or normalize
// its pin is an error, never a skip.
func Collect(srcRoot string) (Report, []WrongBound, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports,
		Dir: srcRoot,
		Env: append(os.Environ(), "GOPATH="+filepath.Dir(srcRoot), "GO111MODULE=off"),
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return Report{}, nil, err
	}
	if len(pkgs) == 0 {
		return Report{}, nil, fmt.Errorf("loaded 0 packages under %s", srcRoot)
	}
	for _, p := range pkgs {
		if len(p.Errors) > 0 {
			return Report{}, nil, fmt.Errorf("package %s: %v", p.PkgPath, p.Errors[0])
		}
	}
	prog, _ := ssautil.Packages(pkgs, ssa.BuilderMode(0))
	prog.Build()

	r := Report{
		TimeByStatus:  map[string]int{},
		SpaceByStatus: map[string]int{},
		PerFamily:     map[string]int{},
	}
	var wrongs []WrongBound
	nop := func(token.Pos, string, ...any) {}
	for _, p := range pkgs {
		ssaFor := func(decl *ast.FuncDecl) *ssa.Function {
			obj, ok := p.TypesInfo.Defs[decl.Name].(*types.Func)
			if !ok {
				return nil
			}
			return prog.FuncValue(obj)
		}
		// The identical resolver wiring the analyzer and metrics use. Corpus
		// code carries no //bigo: directives (enforced below), so Overrides
		// and MethodCosts are empty — the engine runs unaided.
		fns := directive.Scan(p.Syntax, p.TypesInfo, ssaFor, nop)
		if len(fns.Overrides) > 0 || len(fns.MethodCosts) > 0 {
			return Report{}, nil, fmt.Errorf("package %s: corpus code must not carry //bigo: directives", p.PkgPath)
		}
		resolver := callsummary.NewWithMethods(fns.Overrides, fns.MethodCosts)
		spaceResolver := callsummary.NewSpace(nil)

		for _, file := range p.Syntax {
			pins, err := ExtractPins(file)
			if err != nil {
				return Report{}, nil, fmt.Errorf("package %s: %w", p.PkgPath, err)
			}
			for decl, pin := range pins {
				fn := ssaFor(decl)
				if fn == nil {
					return Report{}, nil, fmt.Errorf("%s.%s: pinned function has no SSA", p.PkgPath, decl.Name.Name)
				}
				e := Entry{Pkg: p.PkgPath, Func: decl.Name.Name, Source: pin.Source}

				timePin, err := normalize.Budget(pin.Time, fn)
				if err != nil {
					return Report{}, nil, fmt.Errorf("%s.%s: time pin: %w", p.PkgPath, decl.Name.Name, err)
				}
				emitted, causes := resolver.InferTop(fn)
				st := Classify(emitted, timePin)
				e.TimePin, e.TimeGot, e.TimeStatus = timePin.String(), emitted.String(), st.String()
				if st == Wrong {
					wrongs = append(wrongs, WrongBound{p.PkgPath, decl.Name.Name, "time", timePin.String(), emitted.String()})
				}
				if st == Top && len(causes) > 0 {
					e.Cause = causes[0].Kind.String()
				}

				if pin.Space != nil {
					spacePin, err := normalize.Budget(*pin.Space, fn)
					if err != nil {
						return Report{}, nil, fmt.Errorf("%s.%s: space pin: %w", p.PkgPath, decl.Name.Name, err)
					}
					sp, _ := spaceResolver.SpaceOf(fn, resolver)
					// One emitted space bound: heap ∨ stack upper-bounds the
					// literature's "auxiliary space" whichever channel carries it.
					got := sp.Heap.Join(sp.Stack)
					sst := Classify(got, spacePin)
					e.SpacePin, e.SpaceGot, e.SpaceStatus = spacePin.String(), got.String(), sst.String()
					if sst == Wrong {
						wrongs = append(wrongs, WrongBound{p.PkgPath, decl.Name.Name, "space", spacePin.String(), got.String()})
					}
					r.SpaceByStatus[sst.String()]++
				}

				r.Entries = append(r.Entries, e)
				r.TimeByStatus[st.String()]++
				r.PerFamily[p.PkgPath]++
				r.Total++
			}
		}
	}
	sort.Slice(r.Entries, func(i, j int) bool {
		if r.Entries[i].Pkg != r.Entries[j].Pkg {
			return r.Entries[i].Pkg < r.Entries[j].Pkg
		}
		return r.Entries[i].Func < r.Entries[j].Func
	})
	sort.Slice(wrongs, func(i, j int) bool {
		if wrongs[i].Pkg != wrongs[j].Pkg {
			return wrongs[i].Pkg < wrongs[j].Pkg
		}
		return wrongs[i].Func < wrongs[j].Func
	})
	return r, wrongs, nil
}
