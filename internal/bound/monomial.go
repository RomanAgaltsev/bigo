package bound

import (
	"fmt"
	"sort"
	"strings"
)

// Var is a symbolic size variable, e.g. "n", "m", or "len(a)".
type Var string

// Factor is the exponent of one variable within a monomial: v^Pow · (log v)^Log.
type Factor struct {
	Pow int
	Log int
}

// Monomial is a product of variable factors. The empty Monomial is O(1).
// Invariant: factors never stores a zero Factor{0, 0}.
type Monomial struct {
	factors map[Var]Factor
}

func newMono(f map[Var]Factor) Monomial {
	m := Monomial{
		factors: make(map[Var]Factor),
	}
	for v, fac := range f {
		if fac.Pow != 0 || fac.Log != 0 {
			m.factors[v] = fac
		}
	}
	return m
}

// One returns the O(1) monomial.
func One() Monomial {
	return Monomial{
		factors: make(map[Var]Factor),
	}
}

// Mono returns the single-variable monomial v^pow · (log v)^log.
func Mono(v Var, pow, log int) Monomial {
	return newMono(map[Var]Factor{
		v: {
			Pow: pow,
			Log: log,
		},
	})
}

// Term returns v^1.
func Term(v Var) Monomial {
	return Mono(v, 1, 0)
}

// LogOf returns (log v)^1.
func LogOf(v Var) Monomial {
	return Mono(v, 0, 1)
}

func (m Monomial) at(v Var) Factor {
	return m.factors[v]
}

// FactorOf returns the polynomial and logarithmic exponents of variable v in
// the monomial — (0, 0) when v is absent.
func (m Monomial) FactorOf(v Var) (pow, log int) {
	f := m.factors[v]
	return f.Pow, f.Log
}

func (m Monomial) vars() []Var {
	vs := make([]Var, 0, len(m.factors))
	for v := range m.factors {
		vs = append(vs, v)
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i] < vs[j] })
	return vs
}

// Vars returns the monomial's variables in canonical sorted order.
func (m Monomial) Vars() []Var {
	return m.vars()
}

// Mul multiplies two monomials by adding their per-variable exponents.
func (m Monomial) Mul(o Monomial) Monomial {
	res := make(map[Var]Factor, len(m.factors)+len(o.factors))
	for v, f := range m.factors {
		res[v] = f
	}
	for v, f := range o.factors {
		e := res[v]
		res[v] = Factor{
			Pow: e.Pow + f.Pow,
			Log: e.Log + f.Log,
		}
	}
	return newMono(res)
}

// Equal reports whether two monomials have identical factors.
func (m Monomial) Equal(o Monomial) bool {
	if len(m.factors) != len(o.factors) {
		return false
	}
	for v, f := range m.factors {
		if o.factors[v] != f {
			return false
		}
	}
	return true
}

// String renders the monomial in canonical from.
func (m Monomial) String() string {
	if len(m.factors) == 0 {
		return "1"
	}
	var parts []string
	for _, v := range m.vars() {
		f := m.factors[v]
		switch {
		case f.Pow == 1:
			parts = append(parts, string(v))
		case f.Pow > 1:
			parts = append(parts, fmt.Sprintf("%s^%d", v, f.Pow))
		}
		switch {
		case f.Log == 1:
			parts = append(parts, fmt.Sprintf("log(%s)", v))
		case f.Log > 1:
			parts = append(parts, fmt.Sprintf("log(%s)^%d", v, f.Log))
		}
	}
	return strings.Join(parts, " ")
}
