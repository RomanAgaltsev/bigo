package annotation

import (
	"fmt"
	"strings"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// parseWhere parses a comma-separated list of size bindings, e.g.
// "n=len(a), m=cap(b), k=count".
func parseWhere(s string) (map[bound.Var]SizeRef, error) {
	out := make(map[bound.Var]SizeRef)
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
		if !isIdentifier(p) {
			return SizeRef{}, fmt.Errorf("bad len() argument %q", s)
		}
		return SizeRef{Kind: Len, Param: p}, nil
	case strings.HasPrefix(s, "cap(") && strings.HasSuffix(s, ")"):
		p := s[len("cap(") : len(s)-1]
		if !isIdentifier(p) {
			return SizeRef{}, fmt.Errorf("bad cap() argument %q", s)
		}
		return SizeRef{Kind: Cap, Param: p}, nil
	default:
		if !isIdentifier(s) {
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
