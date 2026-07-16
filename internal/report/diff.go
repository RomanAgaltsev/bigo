package report

import (
	"fmt"
	"strings"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// Compat decides whether two documents may be compared, and reports a warning
// when they may be compared but the comparison deserves a caveat.
//
// Hard errors (apples-to-oranges): a different module, or a different schema
// major. Within a schema major the format is additive-only (document.go:5-7),
// so a minor difference is safe by construction.
//
// Warning: a bigo version difference. Bounds may then differ because the
// engine changed rather than because the code changed, and reporting that as a
// regression would be a false accusation. The envelope carries no analysis
// configuration today, so the version is the only signal available.
func Compat(base, head Document) (string, error) {
	if base.Module != head.Module {
		return "", fmt.Errorf("module mismatch: base %q, head %q", base.Module, head.Module)
	}
	bm, hm := schemaMajor(base.SchemaVersion), schemaMajor(head.SchemaVersion)
	if bm != hm {
		return "", fmt.Errorf("schema major mismatch: base %s, head %s", base.SchemaVersion, head.SchemaVersion)
	}
	if base.BigoVersion != head.BigoVersion {
		return fmt.Sprintf(
			"bigo version differs (base %s, head %s): bound changes may reflect the engine, not the code",
			base.BigoVersion, head.BigoVersion), nil
	}
	return "", nil
}

// schemaMajor extracts the leading major component of a semver string.
// An unparseable version yields "" and thus compares equal only to itself.
func schemaMajor(v string) string {
	if i := strings.IndexByte(v, '.'); i >= 0 {
		return v[:i]
	}
	return v
}

// boundOf reconstructs a bound from its structured serialization. ok is false
// when no bound was recorded — a zero BoundJSON must never read as O(1).
//
// This is the inverse of boundJSON (document.go:97) and exists so the diff can
// hand real bounds to bound.Check instead of comparing prose.
func boundOf(bj BoundJSON) (bound.Bound, bool) {
	if bj.Top {
		return bound.Top(), true
	}
	if bj.Terms == nil {
		return bound.Bound{}, false
	}
	ms := make([]bound.Monomial, 0, len(bj.Terms))
	for _, t := range bj.Terms {
		m := bound.One()
		for v, f := range t {
			m = m.Mul(bound.Mono(bound.Var(v), f.Pow, f.Log))
		}
		ms = append(ms, m)
	}
	return bound.Of(ms...), true
}
