package bound

// factorGE reports whether factor a is >= factor b under the lexicographic
// order on (Pow, Log): higher polynomial power wins. Ties broken by log power.
func factorGE(a, b Factor) bool {
	if a.Pow != b.Pow {
		return a.Pow > b.Pow
	}
	return a.Log >= b.Log
}

// Dominates reports whether a grows at least as fast as b (b = O(a)).
// It is the per-variable product order: a dominates b if for every variable
// appearing in either monomial, a's factor is >= b's factor. Variables absent
// from a monomial have the zero factor {0, 0}. The order is partial: n and m
// are mutually non-dominating (incomparable).
func Dominates(a, b Monomial) bool {
	seen := make(map[Var]bool, len(a.factors))
	for v := range a.factors {
		if !factorGE(a.at(v), b.at(v)) {
			return false
		}
		seen[v] = true
	}
	for v := range b.factors {
		if seen[v] {
			continue
		}
		if !factorGE(a.at(v), b.at(v)) {
			return false
		}
	}
	return true
}
