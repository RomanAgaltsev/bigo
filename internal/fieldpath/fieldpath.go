// Package fieldpath proves which parameter-rooted field reads are stable —
// equal to their value at function entry — so they can serve as size
// variables without alias analysis. A wrong answer here is a wrong bound,
// the highest-severity bug class; every rule below is argued from SSA
// immutability (Rule V) or absence of mutation opportunity (Rule P).
// Stability reasoning assumes no data race (Go memory model); channel-typed
// fields are excluded because channel sync makes concurrent mutation legal.
package fieldpath

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/size"
)

// Stability holds the per-function taint facts Rule P needs. Rule V needs no
// facts at all — it is structural.
type Stability struct {
	taintIdx   map[*ssa.BasicBlock]int  // first dangerous instruction per block
	taintReach map[*ssa.BasicBlock]bool // blocks reachable FROM a tainted block
}

// Analyze computes the stability facts for fn. Safe on nil.
func Analyze(fn *ssa.Function) *Stability {
	s := &Stability{
		taintIdx:   map[*ssa.BasicBlock]int{},
		taintReach: map[*ssa.BasicBlock]bool{},
	}
	if fn == nil {
		return s
	}
	for _, b := range fn.Blocks {
		for i, instr := range b.Instrs {
			if taints(instr) {
				s.taintIdx[b] = i
				break
			}
		}
	}
	var stack []*ssa.BasicBlock
	for b := range s.taintIdx {
		stack = append(stack, b)
	}
	for len(stack) > 0 {
		b := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, succ := range b.Succs {
			if !s.taintReach[succ] {
				s.taintReach[succ] = true
				stack = append(stack, succ)
			}
		}
	}
	return s
}

// taints reports whether instr could mutate an object reachable through a
// pointer parameter — or synchronize with a goroutine that could (channels).
func taints(instr ssa.Instruction) bool {
	switch v := instr.(type) {
	case *ssa.Call:
		if b, ok := v.Call.Value.(*ssa.Builtin); ok {
			switch b.Name() {
			case "len", "cap":
				return false // pure
			}
		}
		return true
	case *ssa.Defer, *ssa.Go, *ssa.RunDefers, *ssa.Select, *ssa.Send, *ssa.MapUpdate:
		return true
	case *ssa.UnOp:
		return v.Op == token.ARROW // channel receive
	case *ssa.Store:
		// A store into a local Alloc cannot alias a caller-visible object;
		// any other store (through a parameter, global, or loaded pointer)
		// might.
		return !allocRooted(v.Addr)
	}
	return false
}

func allocRooted(addr ssa.Value) bool {
	for {
		switch a := addr.(type) {
		case *ssa.Alloc:
			return true
		case *ssa.FieldAddr:
			addr = a.X
		case *ssa.IndexAddr:
			addr = a.X
		default:
			return false
		}
	}
}

// VarFor maps an SSA value to its canonical field-path size variable when it
// is a provably entry-stable field read (or len/cap of one) rooted at a
// parameter or receiver. Nil-receiver safe: rejects everything.
func (s *Stability) VarFor(v ssa.Value) (bound.Var, bool) {
	if s == nil {
		return "", false
	}
	if c, ok := v.(*ssa.Call); ok {
		bi, okB := c.Call.Value.(*ssa.Builtin)
		if !okB || len(c.Call.Args) != 1 {
			return "", false
		}
		arg := c.Call.Args[0]
		path, ok := s.fieldPath(arg)
		if !ok {
			return "", false
		}
		switch bi.Name() {
		case "len":
			if lenEligible(arg.Type()) {
				return size.Len(path), true
			}
		case "cap":
			if isSlice(arg.Type()) {
				return size.Cap(path), true
			}
		}
		return "", false
	}
	if path, ok := s.fieldPath(v); ok && isInteger(v.Type()) {
		return size.Num(path), true
	}
	return "", false
}

// fieldPath returns the dotted path (e.g. "s.items") when v is a stable
// field read of depth <= 2 rooted at a parameter.
func (s *Stability) fieldPath(v ssa.Value) (string, bool) {
	if p, ok := pureFieldChain(v); ok {
		return p, true
	}
	if p, ok := valueSpillChain(v); ok {
		return p, true
	}
	return s.pointerFieldChain(v)
}

// pureFieldChain matches Field(...Field(param)) — Rule V. SSA values are
// immutable, so the read IS the entry value: stable by construction. Source
// that could destabilize the copy (&s, s.f = x) forces an Alloc spill, and
// the read becomes a Load this matcher rejects.
func pureFieldChain(v ssa.Value) (string, bool) {
	var fields []string
	for {
		f, ok := v.(*ssa.Field)
		if !ok {
			break
		}
		st, ok := f.X.Type().Underlying().(*types.Struct)
		if !ok {
			return "", false
		}
		fields = append(fields, st.Field(f.Field).Name())
		v = f.X
	}
	p, ok := v.(*ssa.Parameter)
	if !ok || len(fields) == 0 || len(fields) > 2 {
		return "", false
	}
	return joinPath(p.Name(), fields), true
}

// pointerFieldChain matches loads through FieldAddr chains rooted at a
// pointer parameter — Rule P. Every load in the chain must pass cleanAt.
func (s *Stability) pointerFieldChain(v ssa.Value) (string, bool) {
	ld, ok := v.(*ssa.UnOp)
	if !ok || ld.Op != token.MUL || !s.cleanAt(ld) {
		return "", false
	}
	var fields []string
	addr := ld.X
	for {
		switch a := addr.(type) {
		case *ssa.FieldAddr:
			if len(fields) >= 2 {
				return "", false
			}
			name, ok := fieldNameAt(a.X, a.Field)
			if !ok {
				return "", false
			}
			fields = append(fields, name)
			addr = a.X
		case *ssa.UnOp: // intermediate load of a pointer-typed field
			if a.Op != token.MUL || !s.cleanAt(a) {
				return "", false
			}
			addr = a.X
		case *ssa.Parameter:
			if _, ok := a.Type().Underlying().(*types.Pointer); !ok {
				return "", false
			}
			if len(fields) == 0 {
				return "", false
			}
			return joinPath(a.Name(), fields), true
		default:
			return "", false
		}
	}
}

// fieldNameAt returns the name of field #field of the struct that x (an
// address value: *T) points at. FieldAddr always has a pointer-to-struct X.
func fieldNameAt(x ssa.Value, field int) (string, bool) {
	pt, ok := x.Type().Underlying().(*types.Pointer)
	if !ok {
		return "", false
	}
	st, ok := pt.Elem().Underlying().(*types.Struct)
	if !ok {
		return "", false
	}
	return st.Field(field).Name(), true
}

// valueSpillChain matches a load through a FieldAddr chain rooted at the Alloc
// that spills a value parameter — Rule V in the form go/ssa actually emits. A
// struct value parameter whose field's address is taken (which len(s.items)
// requires) is always spilled to a local Alloc; the pure ssa.Field form never
// appears for such reads. The read is entry-stable because the alloc does not
// escape (Heap == false, so no call can reach it — calls are harmless) and is
// written only by the entry spill of the parameter, so no taint scan is needed.
func valueSpillChain(v ssa.Value) (string, bool) {
	ld, ok := v.(*ssa.UnOp)
	if !ok || ld.Op != token.MUL {
		return "", false
	}
	var fields []string
	addr := ld.X
	for {
		switch a := addr.(type) {
		case *ssa.FieldAddr:
			if len(fields) >= 2 {
				return "", false
			}
			name, ok := fieldNameAt(a.X, a.Field)
			if !ok {
				return "", false
			}
			fields = append(fields, name)
			addr = a.X
		case *ssa.Alloc:
			root, ok := paramSpill(a)
			if !ok || len(fields) == 0 {
				return "", false
			}
			return joinPath(root, fields), true
		default:
			return "", false
		}
	}
}

// paramSpill reports the source name when alloc is a non-escaping spill of a
// value parameter that is never mutated after the spill. Non-heap guarantees
// the alloc's address never leaves the function, so no call can mutate it; the
// single whole-struct store of the *ssa.Parameter is the only write, and no
// address derived from the alloc is ever stored through or passed on. Under
// those conditions every field read equals the value at entry (Rule V).
func paramSpill(alloc *ssa.Alloc) (string, bool) {
	if alloc.Heap {
		return "", false // escapes: a call could hold and mutate it
	}
	root := ""
	for _, ref := range *alloc.Referrers() {
		st, ok := ref.(*ssa.Store)
		if !ok || st.Addr != alloc {
			continue
		}
		p, ok := st.Val.(*ssa.Parameter)
		if !ok || root != "" {
			return "", false // not a param spill, or a second whole-struct write
		}
		root = p.Name()
	}
	if root == "" {
		return "", false
	}
	if addrEscapesOrMutated(alloc, true) {
		return "", false
	}
	return root, true
}

// addrEscapesOrMutated walks the uses of an address rooted at the spill alloc
// and reports whether any field could change or the address could leave the
// function. atAllocRoot marks the alloc itself, whose single whole-struct spill
// store (already validated by paramSpill) is the one permitted write.
func addrEscapesOrMutated(v ssa.Value, atAllocRoot bool) bool {
	for _, ref := range *v.Referrers() {
		switch r := ref.(type) {
		case *ssa.Store:
			if r.Addr == v {
				if atAllocRoot {
					continue // the permitted entry spill
				}
				return true // s.field = ... : the field is mutated
			}
			return true // address stored as a value: escapes into memory
		case *ssa.FieldAddr:
			if addrEscapesOrMutated(r, false) {
				return true
			}
		case *ssa.IndexAddr:
			if addrEscapesOrMutated(r, false) {
				return true
			}
		case *ssa.UnOp:
			if r.Op != token.MUL {
				return true // only loads are harmless
			}
		case *ssa.DebugRef:
			// debug metadata: harmless
		default:
			return true // call argument, phi, convert, ... : may escape/mutate
		}
	}
	return false
}

// cleanAt reports whether no dangerous instruction can execute before instr:
// instr's block is unreachable from every tainted block, and any taint in its
// own block comes strictly after it.
func (s *Stability) cleanAt(instr ssa.Instruction) bool {
	b := instr.Block()
	if s.taintReach[b] {
		return false
	}
	ti, tainted := s.taintIdx[b]
	if !tainted {
		return true
	}
	for i, in := range b.Instrs {
		if in == instr {
			return i < ti
		}
	}
	return false
}

func joinPath(root string, fields []string) string {
	// fields were collected leaf-first; render root.outer.inner.
	out := root
	for i := len(fields) - 1; i >= 0; i-- {
		out += "." + fields[i]
	}
	return out
}

func lenEligible(t types.Type) bool {
	switch u := t.Underlying().(type) {
	case *types.Slice, *types.Array, *types.Map:
		return true
	case *types.Basic:
		return u.Info()&types.IsString != 0
	}
	return false // channels excluded by design — see package comment
}

func isSlice(t types.Type) bool {
	_, ok := t.Underlying().(*types.Slice)
	return ok
}

func isInteger(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsInteger != 0
}

// PathFor exposes the dotted path of a stable field read (see fieldPath) for
// callers that need to name the operand itself — e.g. a ranged map field.
// Nil-safe.
func (s *Stability) PathFor(v ssa.Value) (string, bool) {
	if s == nil {
		return "", false
	}
	return s.fieldPath(v)
}
