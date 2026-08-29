package annotation

import (
	"fmt"
	"strings"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// parseWhere parses a comma-separated list of size bindings, e.g.
// "n=len(a), m=cap(b), k=count".
func parseWhere(s string) (map[bound.Var]SizeRef, error) {
	// NO SIZE HINT, deliberately. bigo's own SM6 fires here — the result-size
	// model (2026-08-29) gave the loop a trip count, so the rule could finally
	// see it — and taking its advice was MEASURED as a regression:
	//
	//	input            no hint              make(map, len(parts))
	//	1,000 commas     14.5us   16.5 KB     31.2us    115 KB   (2.1x, 7.0x)
	//	100,000 commas   1.32ms   1.61 MB     2.65ms   7.90 MB   (2.0x, 4.9x)
	//
	// SM6 assumes the loop RUNS TO COMPLETION, so the entry count equals the
	// trip count. This loop returns an error on the first malformed binding, so
	// on hostile input it stores nothing while the hint reserves a bucket per
	// comma — attacker-controlled amplification, and the input here is a source
	// comment. The hint is right only for input already known valid, which is
	// what the parse is deciding.
	//
	// Found by CI's fuzz-smoke after the hinted version was merged; SM6's
	// early-exit blindness is filed as a rule finding.
	out := make(map[bound.Var]SizeRef) //nolint:bigo // SM6: loop exits early on the error path; see above
	for _, part := range strings.Split(s, ",") {
		name, val, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("bad binding %q: expected name=source", part)
		}
		name = strings.TrimSpace(name)
		val = strings.TrimSpace(val)
		if !isIdentifier(name) {
			return nil, fmt.Errorf("bad binding name %q", name)
		}
		ref, err := parseSizeRef(val)
		if err != nil {
			return nil, err
		}
		out[bound.Var(name)] = ref
	}
	return out, nil
}

func parseSizeRef(s string) (SizeRef, error) {
	switch {
	case strings.HasPrefix(s, "len(") && strings.HasSuffix(s, ")"):
		p := s[len("len(") : len(s)-1]
		if !isFieldPathIdent(p) {
			return SizeRef{}, fmt.Errorf("bad len() argument %q", s)
		}
		return SizeRef{Kind: Len, Param: p}, nil
	case strings.HasPrefix(s, "cap(") && strings.HasSuffix(s, ")"):
		p := s[len("cap(") : len(s)-1]
		if !isFieldPathIdent(p) {
			return SizeRef{}, fmt.Errorf("bad cap() argument %q", s)
		}
		return SizeRef{Kind: Cap, Param: p}, nil
	default:
		if !isFieldPathIdent(s) {
			return SizeRef{}, fmt.Errorf("bad size reference %q", s)
		}
		return SizeRef{Kind: Num, Param: s}, nil
	}
}

// isIdentifier reports whether s is a non-empty Go-like identifier.
func isIdentifier(s string) bool {
	if s == "" || !isIdentStart(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isIdentPart(s[i]) {
			return false
		}
	}
	return true
}

// isFieldPathIdent reports whether s is an identifier or a dotted field path
// of at most two selections (root, root.f, or root.f.g) — the same depth
// limit fieldpath enforces on the inference side.
func isFieldPathIdent(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return false
	}
	for _, p := range parts {
		if !isIdentifier(p) {
			return false
		}
	}
	return true
}
