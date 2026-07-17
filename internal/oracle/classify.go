package oracle

import "github.com/RomanAgaltsev/bigo/internal/bound"

// Status is one entry's per-dimension oracle outcome.
type Status int

const (
	// Wrong means the emitted bound does not dominate the pin — strictly below
	// OR incomparable. A prime-directive break; fails the build unconditionally
	// and never appears in a golden (spec §4.2).
	Wrong Status = iota
	// Exact means emitted equals the pin.
	Exact
	// Loose means emitted strictly dominates the pin — sound; a graduation target.
	Loose
	// Top means emitted is ⊤ — safe; the annotate-or-trust evidence rows.
	Top
)

func (s Status) String() string {
	switch s {
	case Wrong:
		return "wrong"
	case Exact:
		return "exact"
	case Loose:
		return "loose"
	default:
		return "top"
	}
}

// Classify compares an emitted bound against a normalized pin. Soundness is
// domination: Check(pin, emitted) == Within proves pin ≤ emitted. Everything
// unproven — Exceeds and Unknown alike, so incomparable included — is Wrong:
// an emitted bound that grows slower than the true bound in any regime is a
// wrong bound. A legitimately-incomparable-but-sound case is a pin-expression
// bug and is fixed by restating the pin, never by weakening this rule.
func Classify(emitted, pin bound.Bound) Status {
	if emitted.IsTop() {
		return Top
	}
	if emitted.Equal(pin) {
		return Exact
	}
	if bound.Check(pin, emitted) == bound.Within {
		return Loose
	}
	return Wrong
}
