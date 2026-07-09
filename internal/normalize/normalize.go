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

// Budget rewrites d.Budget's terse variables into canonical size variables,
// using d.Bindings, defaulting an unbound "n" to fn's primary size parameter.
func Budget(d annotation.Directive, fn *ssa.Function) (bound.Bound, error) {
	rename := map[bound.Var]bound.Var{}
	for v, ref := range d.Bindings {
		rename[v] = size.FromRef(ref)
	}
	for _, v := range budgetVars(d.Budget) {
		if _, ok := rename[v]; ok {
			continue
		}
		if v == "n" {
			p := primarySize(fn)
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

// primarySize returns the canonical size var of fn's first slice/map/string/
// array parameter, else its first integer parameter, else "".
func primarySize(fn *ssa.Function) bound.Var {
	for _, p := range fn.Params {
		switch p.Type().Underlying().(type) {
		case *types.Slice, *types.Map, *types.Array:
			return size.Len(p.Name())
		}
		if b, ok := p.Type().Underlying().(*types.Basic); ok && b.Info()&types.IsString != 0 {
			return size.Len(p.Name())
		}
	}
	for _, p := range fn.Params {
		if b, ok := p.Type().Underlying().(*types.Basic); ok && b.Info()&types.IsInteger != 0 {
			return size.Num(p.Name())
		}
	}
	return ""
}
