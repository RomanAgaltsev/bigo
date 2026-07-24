package assume

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/costtable"
	"github.com/RomanAgaltsev/bigo/internal/normalize"
	"github.com/RomanAgaltsev/bigo/internal/size"
)

// Set is a loaded assumption file, compiled lazily per target signature.
// Safe for concurrent use: go/analysis runs packages in parallel.
type Set struct {
	mu       sync.Mutex
	entries  map[string]Entry
	order    []string // file order, for Entries()
	compiled map[string]*compiledEntry
	firstErr error
}

type compiledEntry struct {
	b     bound.Bound
	names []string
}

// NewSet indexes parsed entries. Duplicate keys were rejected at parse.
func NewSet(es []Entry) *Set {
	s := &Set{entries: map[string]Entry{}, compiled: map[string]*compiledEntry{}}
	for _, e := range es {
		s.entries[e.Key] = e
		s.order = append(s.order, e.Key)
	}
	return s
}

// Load reads, parses, and indexes an assumption file.
func Load(path string) (*Set, error) {
	es, err := ParseFile(path)
	if err != nil {
		return nil, err
	}
	return NewSet(es), nil
}

// Has reports whether key has an assumption, without compiling it.
func (s *Set) Has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.entries[key]
	return ok
}

// For returns the compiled bound and the argument-name vector for key against
// sig, compiling on first use. The names align with ssa.CallCommon.Args for a
// static call: a method's receiver is Args[0], so names[0] is a placeholder
// the bound cannot reference — receiver-dependent assumption bounds are not
// supported (parameters only).
//
// A compilation failure is recorded (Err) and reported as no-match here:
// CallCost cannot error, and the whole-module drivers surface it via Validate
// before any analysis runs.
func (s *Set) For(key string, sig *types.Signature) (bound.Bound, []string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if !ok {
		return bound.Bound{}, nil, false
	}
	if c, done := s.compiled[key]; done {
		if c == nil {
			return bound.Bound{}, nil, false // previously failed
		}
		return c.b, c.names, true
	}
	b, err := s.compile(e, sig)
	if err != nil {
		s.compiled[key] = nil
		if s.firstErr == nil {
			s.firstErr = fmt.Errorf("assumption for %s: %v", key, err)
		}
		return bound.Bound{}, nil, false
	}
	var names []string
	if sig.Recv() != nil {
		names = append(names, "$recv") // aligns with Args[0]; never referenceable
	}
	for i := 0; i < sig.Params().Len(); i++ {
		names = append(names, sig.Params().At(i).Name())
	}
	s.compiled[key] = &compiledEntry{b: b, names: names}
	return b, names, true
}

// compile normalizes the entry against sig and then insists every variable in
// the result is a plain size of one of sig's parameters. normalize does not
// check that a where-binding names a real parameter — a directive with a bogus
// binding produces a var no caller can interpret — and for assumptions that
// class is a hard error, never a silent Top or a leaked variable (spec §3).
func (s *Set) compile(e Entry, sig *types.Signature) (bound.Bound, error) {
	b, err := normalize.BudgetSig(e.Dir, sig)
	if err != nil {
		return bound.Bound{}, err
	}
	allowed := map[bound.Var]bool{}
	for i := 0; i < sig.Params().Len(); i++ {
		p := sig.Params().At(i).Name()
		allowed[size.Len(p)] = true
		allowed[size.Cap(p)] = true
		allowed[size.Num(p)] = true
	}
	for _, m := range b.Terms() {
		for _, v := range m.Vars() {
			if !allowed[v] {
				return bound.Bound{}, fmt.Errorf("bound variable %q does not name a parameter size", v)
			}
		}
	}
	return b, nil
}

// Err returns the first compilation error recorded by For, if any.
func (s *Set) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.firstErr
}

// Validate checks every entry against the loaded program: each key must match
// at least one function, and each matched entry must compile against its
// signature. Hard errors both ways (spec §3) — the whole-module drivers call
// this before analysis so a broken set never produces numbers.
func (s *Set) Validate(prog *ssa.Program) error {
	matched := map[string]bool{}
	for fn := range ssautil.AllFunctions(prog) {
		key, ok := costtable.FuncKey(fn)
		if !ok || !s.Has(key) {
			continue
		}
		matched[key] = true
		s.For(key, fn.Signature) // compile eagerly; failure lands in firstErr
	}
	var missing []string
	s.mu.Lock()
	for _, key := range s.order {
		if !matched[key] {
			missing = append(missing, key)
		}
	}
	s.mu.Unlock()
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("assumption keys match no function in the loaded module: %s", strings.Join(missing, ", "))
	}
	return s.Err()
}

// Entries returns the entries in file order (the document's trust surface).
func (s *Set) Entries() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, 0, len(s.order))
	for _, k := range s.order {
		out = append(out, s.entries[k])
	}
	return out
}
