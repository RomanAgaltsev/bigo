// Package oracle drives the canonical algorithm corpus — bigo's prime-directive
// instrument: real algorithms whose worst-case bounds are known from the
// literature, pinned in-source, compared against unaided inference by bound
// domination. An emitted bound that does not dominate its pin is a wrong bound
// and fails the build. Spec: vault
// superpowers/specs/2026-07-16-bigo-canonical-corpus-design.md.
package oracle

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
)

// Pin is one corpus entry's parsed ground truth.
type Pin struct {
	// Time is the literature worst-case time bound (Budget + where-Bindings).
	Time annotation.Directive
	// Space is the literature auxiliary-space/stack bound; nil when the
	// literature states none (then only time is scored).
	Space *annotation.Directive
	// Source is the mandatory citation. An entry without a citation does not exist.
	Source string
}

// prefix is exact, like //bigo: — "// oracle:" with a space is prose.
const prefix = "//oracle:"

// ExtractPins returns the pinned function declarations of one file. Malformed
// pins are errors, never skips: a skipped pin would be a silently missing
// oracle entry, and the reconciliation count could not catch it.
func ExtractPins(file *ast.File) (map[*ast.FuncDecl]Pin, error) {
	out := map[*ast.FuncDecl]Pin{}
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Doc == nil {
			continue
		}
		pin, found, err := parseDoc(fd.Doc)
		if err != nil {
			return nil, fmt.Errorf("func %s: %w", fd.Name.Name, err)
		}
		if found {
			out[fd] = pin
		}
	}
	return out, nil
}

// parseDoc scans one doc comment group. found reports whether any //oracle:
// line was present; if so, time and source are mandatory.
func parseDoc(doc *ast.CommentGroup) (Pin, bool, error) {
	var pin Pin
	var hasTime, found bool
	for _, c := range doc.List {
		if !strings.HasPrefix(c.Text, prefix) {
			continue
		}
		found = true
		verb, rest, _ := strings.Cut(c.Text[len(prefix):], " ")
		rest = strings.TrimSpace(rest)
		switch verb {
		case "time":
			if hasTime {
				return Pin{}, true, fmt.Errorf("duplicate //oracle:time")
			}
			d, err := annotation.Parse("//bigo:max " + rest)
			if err != nil {
				return Pin{}, true, fmt.Errorf("//oracle:time: %w", err)
			}
			pin.Time, hasTime = d, true
		case "space":
			if pin.Space != nil {
				return Pin{}, true, fmt.Errorf("duplicate //oracle:space")
			}
			d, err := annotation.Parse("//bigo:max " + rest)
			if err != nil {
				return Pin{}, true, fmt.Errorf("//oracle:space: %w", err)
			}
			pin.Space = &d
		case "source":
			if pin.Source != "" {
				return Pin{}, true, fmt.Errorf("duplicate //oracle:source")
			}
			if rest == "" {
				return Pin{}, true, fmt.Errorf("//oracle:source is empty")
			}
			pin.Source = rest
		default:
			return Pin{}, true, fmt.Errorf("unknown //oracle: verb %q", verb)
		}
	}
	if !found {
		return Pin{}, false, nil
	}
	if !hasTime {
		return Pin{}, true, fmt.Errorf("//oracle:time is mandatory")
	}
	if pin.Source == "" {
		return Pin{}, true, fmt.Errorf("//oracle:source is mandatory")
	}
	return pin, true, nil
}
