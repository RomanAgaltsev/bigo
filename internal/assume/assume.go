// Package assume loads external assumption files: bounds the engine treats as
// trusted user claims for named functions, without touching the target's
// source. Semantically an entry is a //bigo:cost directive applied from the
// outside; it reuses the directive grammar and never introduces a second
// bound syntax.
//
// Malformed or unresolvable entries are hard errors, never skips: a silently
// dropped assumption corrupts a what-if measurement (spec §3).
//
// Three limits are measured, not theoretical: interface methods and builtins
// cannot be named at all (campaign 1, proven on a fixture), and a key is
// separated from its bound at the first " O(" so that keys containing spaces —
// a method on a generic type with two or more type parameters — stay
// expressible (2026-08-11 review F5).
package assume

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
)

// Entry is one parsed assumption: treat the function named Key as having the
// time bound Expr. Key is the cost table's vocabulary — package-qualified
// name, receiver-qualified for methods (costtable.FuncKey).
type Entry struct {
	Key  string
	Expr string
	Line int
	Dir  annotation.Directive
}

// ParseFile reads and validates an assumption file.
func ParseFile(path string) ([]Entry, error) {
	f, err := os.Open(path) // #nosec G304 -- user-supplied config path, like -i on badge
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return parse(path, f)
}

// ParseText parses assumption-file content from a string (tests, whatif).
func ParseText(text string) ([]Entry, error) { return parse("inline", strings.NewReader(text)) }

func parse(name string, r io.Reader) ([]Entry, error) {
	var es []Entry
	seen := map[string]int{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split at the first " O(" rather than the first space: a KEY may
		// contain spaces (types.Func.FullName renders multiple type parameters
		// comma-space separated, e.g. "(*pkg.Pair[K, V]).Get"), while every
		// bound goes through parseBigO, which requires a literal "O(" — so this
		// separator is total where a space is not (2026-08-11 review F5).
		i := strings.Index(line, " O(")
		if i < 0 {
			return nil, fmt.Errorf("%s: line %d: want '<key> O(<expr>)', got %q", name, n, line)
		}
		key := strings.TrimSpace(line[:i])
		expr := strings.TrimSpace(line[i+1:])
		if key == "" {
			return nil, fmt.Errorf("%s: line %d: want '<key> O(<expr>)', got %q", name, n, line)
		}
		dir, err := annotation.Parse("//bigo:cost " + expr)
		if err != nil {
			return nil, fmt.Errorf("%s: line %d: invalid bound %q: %v", name, n, expr, err)
		}
		if prev, dup := seen[key]; dup {
			return nil, fmt.Errorf("%s: line %d: duplicate key %s (first at line %d)", name, n, key, prev)
		}
		seen[key] = n
		es = append(es, Entry{Key: key, Expr: expr, Line: n, Dir: dir})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return es, nil
}
