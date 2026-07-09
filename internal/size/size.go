// Package size defines the canonical identity of input-size variables shared by
// the complexity engine and the annotation layer.
package size

import (
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// Len is the size variable for len(param).
func Len(param string) bound.Var {
	return bound.Var("len(" + param + ")")
}

// Cap is the size variable for cap(param).
func Cap(param string) bound.Var {
	return bound.Var("cap(" + param + ")")
}

// Num is the size variable for a numeric parameter's value.
func Num(param string) bound.Var {
	return bound.Var(param)
}

// FromRef maps a parsed annotation size reference to a canonical size variable.
func FromRef(r annotation.SizeRef) bound.Var {
	switch r.Kind {
	case annotation.Len:
		return Len(r.Param)
	case annotation.Cap:
		return Cap(r.Param)
	default:
		return Num(r.Param)
	}
}

// Value returns the canonical size variable of an SSA value when it is a
// parameter whose size is meaningful (slice/map/array/string -> len,
// integer -> the value itself) and false otherwise.
func Value(v ssa.Value) (bound.Var, bool) {
	p, ok := v.(*ssa.Parameter)
	if !ok {
		return "", false
	}
	switch p.Type().Underlying().(type) {
	case *types.Slice, *types.Map, *types.Array:
		return Len(p.Name()), true
	}
	if b, ok := p.Type().Underlying().(*types.Basic); ok {
		switch {
		case b.Info()&types.IsString != 0:
			return Len(p.Name()), true
		case b.Info()&types.IsInteger != 0:
			return Num(p.Name()), true
		}
	}
	return "", false
}
