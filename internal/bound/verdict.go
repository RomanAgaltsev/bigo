package bound

// Verdict is the three-valued result of comparing an inferred vound to a budget.
type Verdict int

const (
	// Within means the inferred bound is provable no larger than the budget.
	Within Verdict = iota
	// Exceeds means the inferred bound provable violates the budget.
	Exceeds
	// Unknown means the comparison cannot be decided soundly (annotate to resolve).
	Unknown
)

func (v Verdict) String() string {
	switch v {
	case Within:
		return "within"
	case Exceeds:
		return "exceeds"
	default:
		return "unknown"
	}
}

// Check compares an inferred bound against a declared budget, soundly. It never
// returns Exceeds unless the violation is provable, and never returns Within
// unless it is provable. All remaining cases are Unknown.
func Check(inferred, budget Bound) Verdict {
	if inferred.top || budget.top {
		return Unknown
	}
	allWithin := true
	for _, t := range inferred.terms {
		covered := false
		for _, m := range budget.terms {
			if Dominates(m, t) { // t = O(m)
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		// t is not covered by any single term. It is a provable violation
		// only if it strictly dominates every budget term (grows at least as fast
		// as all of them and is not equal to any).
		exceedsAll := true
		for _, m := range budget.terms {
			if !Dominates(t, m) || t.Equal(m) {
				exceedsAll = false
				break
			}
		}
		if exceedsAll {
			return Exceeds
		}
		allWithin = false
	}
	if allWithin {
		return Within
	}
	return Unknown
}
