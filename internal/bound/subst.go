package bound

// Subst renames variables according to m (unmapped variables are unchanged).
// T maps to T.
func (b Bound) Subst(m map[Var]Var) Bound {
	if b.top {
		return Top()
	}
	out := make([]Monomial, 0, len(b.terms))
	for _, mono := range b.terms {
		nf := make(map[Var]Factor, len(mono.factors))
		for v, f := range mono.factors {
			nv := v
			if r, ok := m[v]; ok {
				nv = r
			}
			e := nf[nv]
			nf[nv] = Factor{Pow: e.Pow + f.Pow, Log: e.Log + f.Log}
		}
		out = append(out, newMono(nf))
	}
	return Bound{terms: reduce(out)}
}
