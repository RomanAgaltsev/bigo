// Package normalize rewrites annotation budgets into canonical size variables.
package normalize

import (
	"errors"
	"fmt"
	"go/types"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/size"
	"golang.org/x/tools/go/ssa"
)

// param is the name/type pair Budget needs — the common denominator of
// *ssa.Function params and *types.Signature params.
type param struct {
	name string
	typ  types.Type
}

// Budget rewrites d.Budget's terse variables into canonical size variables,
// using d.Bindings, defaulting an unbound "n" to fn's primary size parameter.
func Budget(d annotation.Directive, fn *ssa.Function) (bound.Bound, error) {
	ps := make([]param, len(fn.Params))
	for i, p := range fn.Params {
		ps[i] = param{p.Name(), p.Type()}
	}
	return budget(d, ps)
}

// BudgetSig is Budget for a bare signature — interface methods have no
// *ssa.Function to hand.
func BudgetSig(d annotation.Directive, sig *types.Signature) (bound.Bound, error) {
	ps := make([]param, sig.Params().Len())
	for i := range ps {
		v := sig.Params().At(i)
		ps[i] = param{v.Name(), v.Type()}
	}
	return budget(d, ps)
}

func budget(d annotation.Directive, ps []param) (bound.Bound, error) {
	rename := map[bound.Var]bound.Var{}
	for v, ref := range d.Bindings {
		rename[v] = size.FromRef(ref)
	}
	for _, v := range budgetVars(d.Budget) {
		if _, ok := rename[v]; ok {
			continue
		}
		if v == "n" {
			p := primarySize(ps)
			if p == "" {
				return bound.Bound{}, errors.New("cannot default 'n': function has no size parameter")
			}
			rename[v] = p
			continue
		}
		return bound.Bound{}, fmt.Errorf("unbound size variable %q (add a 'where' clause)", v)
	}
	return d.Budget.Subst(rename), nil
}

// primarySize returns the canonical size var of the first slice/map/string/
// array parameter, else the first integer parameter, else "".
func primarySize(ps []param) bound.Var {
	for _, p := range ps {
		switch p.typ.Underlying().(type) {
		case *types.Slice, *types.Map, *types.Array:
			return size.Len(p.name)
		}
		if b, ok := p.typ.Underlying().(*types.Basic); ok && b.Info()&types.IsString != 0 {
			return size.Len(p.name)
		}
	}
	for _, p := range ps {
		if b, ok := p.typ.Underlying().(*types.Basic); ok && b.Info()&types.IsInteger != 0 {
			return size.Num(p.name)
		}
	}
	return ""
}

// budgetVars returns the distinct variables appearing in a budget.
func budgetVars(b bound.Bound) []bound.Var {
	seen := map[bound.Var]bool{}
	var out []bound.Var
	for _, m := range b.Terms() {
		for _, v := range m.Vars() {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}
