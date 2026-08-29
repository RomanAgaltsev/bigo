package annotation

import (
	"fmt"
	"strings"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// parseWhere parses a comma-separated list of size bindings, e.g.
// "n=len(a), m=cap(b), k=count".
func parseWhere(s string) (map[bound.Var]SizeRef, error) {
	// Preallocated with the binding count. bigo's own SM6 flagged this on the
	// 2026-08-29 result-size branch: until len(strings.Split(s, sep)) was
	// curated as O(len(s)) the loop had no trip count, so the rule could not
	// fire. First finding the result-size model produced on bigo itself.
	parts := strings.Split(s, ",")
	out := make(map[bound.Var]SizeRef, len(parts))
	for _, part := range parts {
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
