// Package bound models asymptotic (big-O) cost as antichains of poly-log
// monomials, with the operations needed to combine and compare them.
package bound

import (
	"sort"
	"strings"
)

// Bound is an asymptotic bound: either the top element T (unverifiable) or an
// antichain of non-dominated monomials (a Pareto frontier). The empty antichain
// is not representable. O(1) is the antichain {One()}.
type Bound struct {
	top   bool
	terms []Monomial
}

// Top returns the T / unverifiable bound.
func Top() Bound {
	return Bound{top: true}
}

// Constant returns O(1).
func Constant() Bound {
	return Bound{
		terms: []Monomial{One()},
	}
}

// Of builds a bound from monomials, reducing to an antichain. Of() == Constant().
func Of(ms ...Monomial) Bound {
	if len(ms) == 0 {
		return Constant()
	}
	return Bound{terms: reduce(ms)}
}

// IsTop reports whether the bound is T (unverifiable).
func (b Bound) IsTop() bool {
	return b.top
}

// Terms returns the antichain of monomials (empty for T).
func (b Bound) Terms() []Monomial {
	return b.terms
}

// Vars returns the monomial's variables in canonical sorted order.
func (m Monomial) Vars() []Var {
	return m.vars()
}

// reduce keeps only the maximal monomials: any monomial dominated by other
// (equal counts as dominated) is dropped, so the result is an antichain.
func reduce(ms []Monomial) []Monomial {
	var out []Monomial
	for _, m := range ms {
		skip := false
		for _, k := range out {
			if Dominates(k, m) { // some kept term already covers m (incl. equal)
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		keep := make([]Monomial, 0, len(out)+1)
		for _, k := range out {
			if Dominates(m, k) { // m strictly covers k, drop k
				continue
			}
			keep = append(keep, k)
		}
		keep = append(keep, m)
		out = keep
	}
	return out
}

// Join returns the dominant terms of both bounds (sequantial-sum and branch-max)
// are the same asymptotic operation). T is absorbing.
func (b Bound) Join(o Bound) Bound {
	if b.top || o.top {
		return Top()
	}
	all := make([]Monomial, 0, len(b.terms)+len(o.terms))
	all = append(all, b.terms...)
	all = append(all, o.terms...)
	return Bound{terms: reduce(all)}
}

// Equal reports whether two bounds are asymptotically identical (order-insensitive).
func (b Bound) Equal(o Bound) bool {
	if b.top != o.top {
		return false
	}
	if b.top {
		return true
	}
	if len(b.terms) != len(o.terms) {
		return false
	}
	for _, m := range b.terms {
		found := false
		for _, n := range o.terms {
			if m.Equal(n) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// String renders the bound as "O(...)" with terms in canonical sorted order, or
// "unverifiable" for T.
func (b Bound) String() string {
	if b.top {
		return "unverifiable"
	}
	ss := make([]string, len(b.terms))
	for i, m := range b.terms {
		ss[i] = m.String()
	}
	sort.Strings(ss)
	return "O(" + strings.Join(ss, " + ") + ")"
}

// Mul multiplies two bounds: the loop operation (trip-count * body). It forms
// the pairwise products of the two antichains, then reduce. T is absorbing.
func (b Bound) Mul(o Bound) Bound {
	if b.top || o.top {
		return Top()
	}
	prod := make([]Monomial, 0, len(b.terms)+len(o.terms))
	for _, x := range b.terms {
		for _, y := range o.terms {
			prod = append(prod, x.Mul(y))
		}
	}
	return Bound{terms: reduce(prod)}
}
