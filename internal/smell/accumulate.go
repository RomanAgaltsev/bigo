package smell

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

func init() { register("SM1", smConcatInLoop) }

// smConcatInLoop fires when a string-typed phi in a data-dependent loop header
// has a back-edge value that — through a chain of string + BinOps or a
// fmt.Sprintf call — includes the phi itself. That is the SSA shape of
// `s += x` / `s = s + x` / `s = fmt.Sprintf("%s", s, ...)` in a loop, the
// quadratic string-building pattern strings.Builder replaces.
func smConcatInLoop(fn *ssa.Function, ctx *fnContext) []Finding {
	var out []Finding
	for _, lp := range dataDependentLoops(fn, ctx) {
		for _, instr := range lp.Header.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok || !isString(phi.Type()) {
				continue
			}
			if phi.Block() != lp.Header {
				continue
			}
			// Check each back-edge (from inside the loop) for a self-referential
			// string accumulation.
			for i, edge := range phi.Edges {
				if i == 0 {
					continue // entry edge
				}
				if lp.Blocks[phi.Block().Preds[i]] && stringChainReaches(edge, phi, nil) {
					out = append(out, Finding{
						Pos:     phi.Pos(),
						Rule:    "SM1",
						Message: "string built by repeated concatenation in a loop (quadratic); use strings.Builder",
					})
					break
				}
			}
		}
	}
	return out
}

// stringChainReaches reports whether v — through a bounded chain of string +
// BinOps and fmt.Sprintf varargs — transitively references target. visited
// bounds recursion over cyclic value graphs.
func stringChainReaches(v ssa.Value, target *ssa.Phi, visited map[ssa.Value]bool) bool {
	if v == target {
		return true
	}
	if visited == nil {
		visited = map[ssa.Value]bool{}
	}
	if visited[v] {
		return false
	}
	visited[v] = true
	switch v := v.(type) {
	case *ssa.BinOp:
		if v.Op != token.ADD || !isString(v.Type()) {
			return false
		}
		return stringChainReaches(v.X, target, visited) || stringChainReaches(v.Y, target, visited)
	case *ssa.Call:
		// fmt.Sprintf("%s", s, ...) wraps the varargs in an Alloc-backed slice.
		// The phi appears (boxed in a MakeInterface) as a stored element; chase
		// through the Alloc's stores to find it.
		if name, ok := calleeOrigin(&v.Call); ok && name == "fmt.Sprintf" {
			for _, arg := range v.Call.Args {
				if stringChainReaches(arg, target, visited) {
					return true
				}
			}
		}
	case *ssa.Slice:
		// varargs slice over an Alloc: chase the alloc's stored elements.
		return stringChainReaches(v.X, target, visited)
	case *ssa.Alloc:
		for _, ref := range *v.Referrers() {
			switch r := ref.(type) {
			case *ssa.Store:
				if stringChainReaches(r.Val, target, visited) {
					return true
				}
			case *ssa.IndexAddr:
				// varargs aggregate: stores flow through &alloc[i]; chase its stores.
				if allocStoresReach(r, target, visited) {
					return true
				}
			}
		}
	case *ssa.MakeInterface:
		// boxing a string into any for the varargs aggregate
		return stringChainReaches(v.X, target, visited)
	}
	return false
}

// allocStoresReach reports whether any store through addr (an &alloc[i] derived
// from the varargs aggregate) reaches target. This handles the Sprintf varargs
// shape where the phi is boxed in a MakeInterface and stored via an IndexAddr.
func allocStoresReach(addr *ssa.IndexAddr, target *ssa.Phi, visited map[ssa.Value]bool) bool {
	for _, ref := range *addr.Referrers() {
		st, ok := ref.(*ssa.Store)
		if !ok {
			continue
		}
		if stringChainReaches(st.Val, target, visited) {
			return true
		}
	}
	return false
}
