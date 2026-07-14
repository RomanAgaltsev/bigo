package recurrence

// Package-internal two-function recursion cycles. A mutual pair is exactly:
// fn -> g and g -> fn by static calls, same package, neither self-recursive,
// and g unique. Anything else (3-cycles, multi-cycles, cross-package, dynamic
// edges) reads as "no partner" and stays ⊤ — see the mutual-recursion spec's
// §2 non-goals.

import (
	"golang.org/x/tools/go/ssa"
)

// callsTo returns fn's static call sites whose callee is target.
func callsTo(fn, target *ssa.Function) []*ssa.CallCommon {
	var out []*ssa.CallCommon
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			if cc := callCommon(instr); cc != nil && cc.StaticCallee() == target {
				out = append(out, cc)
			}
		}
	}
	return out
}

// MutualPartner returns the unique two-cycle partner of fn, if any: the same-
// package function g such that fn statically calls g, g statically calls fn,
// neither is self-recursive, and no other function also forms a two-cycle with
// fn. A second distinct partner makes the SCC larger than two, which is out of
// scope, so the result is (nil, false).
func MutualPartner(fn *ssa.Function) (*ssa.Function, bool) {
	if fn == nil || len(fn.Blocks) == 0 || fn.Pkg == nil || IsSelfRecursive(fn) {
		return nil, false
	}
	var partner *ssa.Function
	seen := map[*ssa.Function]bool{}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			cc := callCommon(instr)
			if cc == nil {
				continue
			}
			g := cc.StaticCallee()
			if g == nil || g == fn || seen[g] {
				continue
			}
			seen[g] = true
			if g.Pkg != fn.Pkg || len(g.Blocks) == 0 || IsSelfRecursive(g) {
				continue
			}
			if len(callsTo(g, fn)) == 0 {
				continue // g does not call back: not a two-cycle member
			}
			if partner != nil {
				return nil, false // two distinct two-cycles through fn: ambiguous
			}
			partner = g
		}
	}
	return partner, partner != nil
}
