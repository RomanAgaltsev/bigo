// Package annotation parses bigo directive comments (//bigo:...) into structured
// verbs and their associated asymptotic bounds.
package annotation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/RomanAgaltsev/bigo/internal/bound"
)

// Verb is the bigo directive verb.
type Verb int

const (
	// Max declares a budget the inferred bound must not exceed.
	Max Verb = iota
	// Cost asserts the cost of a function or interface method.
	Cost
	// Ignore trusts a construct, treating it as O(1).
	Ignore
	// Space is reserved for Phase 2 (space complexity); parsed but inert.
	Space
)

func (v Verb) String() string {
	switch v {
	case Max:
		return "max"
	case Cost:
		return "cost"
	case Ignore:
		return "ignore"
	case Space:
		return "space"
	default:
		return "unknown"
	}
}

// SizeKind is how a size variable binds to a function parameter.
type SizeKind int

const (
	// Len binds to len(param).
	Len SizeKind = iota
	// Cap binds to cap(param).
	Cap
	// Num binds to a numeric parameter's value.
	Num
)

// SizeRef binds a size variable to a source of size (a parameter).
type SizeRef struct {
	Kind  SizeKind
	Param string
}

// Directive is a parsed //bigo: comment.
type Directive struct {
	Verb     Verb
	Budget   bound.Bound
	Bindings map[bound.Var]SizeRef
	Raw      string
}

// whereWord matches the where keyword with mandatory surrounding whitespace.
var whereWord = regexp.MustCompile(`\swhere\s`)

// Parse parses a single //bigo: comment line. Like //go: directives, the
// prefix must be exact — `// bigo:` with a space is prose, not a directive.
func Parse(text string) (Directive, error) {
	const prefix = "//bigo:"
	if !strings.HasPrefix(text, prefix) {
		return Directive{}, fmt.Errorf("not a bigo directive: %q", text)
	}
	s := text[len(prefix):]

	verbTok, rest, _ := strings.Cut(strings.TrimSpace(s), " ")
	rest = strings.TrimSpace(rest)

	d := Directive{
		Raw:      text,
		Bindings: map[bound.Var]SizeRef{},
	}
	switch verbTok {
	case "ignore":
		d.Verb = Ignore
		if rest != "" {
			return Directive{}, fmt.Errorf("ignore takes no argument, got %q", rest)
		}
		return d, nil
	case "max":
		d.Verb = Max
	case "cost":
		d.Verb = Cost
	case "space":
		d.Verb = Space
	default:
		return Directive{}, fmt.Errorf("unknown bigo verb %q", verbTok)
	}

	// The grammar is `bigexpr SP "where" SP bindings` — split on the word,
	// not the substring, so O(nowhere) parses as a variable.
	exprPart, wherePart := rest, ""
	if loc := whereWord.FindStringIndex(rest); loc != nil {
		exprPart, wherePart = rest[:loc[0]], rest[loc[1]:]
	}
	b, err := parseBigO(strings.TrimSpace(exprPart))

	if err != nil {
		return Directive{}, err
	}
	d.Budget = b

	if strings.TrimSpace(wherePart) != "" {
		binds, err := parseWhere(strings.TrimSpace(wherePart))
		if err != nil {
			return Directive{}, err
		}
		d.Bindings = binds
	}
	return d, nil
}
