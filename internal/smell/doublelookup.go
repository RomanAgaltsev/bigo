package smell

import (
	"golang.org/x/tools/go/ssa"
)

// sm7ContainsNames are the (Contains, Index) callee pairs whose redundant use
// is the slice variant of SM7.
var sm7ContainsNames = map[string]bool{
	"slices.Contains": true,
}
var sm7IndexNames = map[string]bool{
	"slices.Index": true,
}

func init() { register("SM7", smDoubleLookup) }

// smDoubleLookup fires on a redundant second lookup that the first already
// answered:
//  1. Map: a comma-ok lookup (m[k], ok) followed, on a dominated path, by a
//     plain lookup (m[k]) with the same SSA X and Index, when the function
//     contains NO MapUpdate on X anywhere (conservative, function-wide).
//  2. Slice: a slices.Contains(s, x) followed by slices.Index(s, x) on a
//     dominated path with identical SSA arguments.
//
// SSA identity is the rule: different key values (even if provably equal) do
// not fire — we never invent a claim from value equality.
func smDoubleLookup(fn *ssa.Function, _ *fnContext) []Finding {
	var out []Finding
	out = append(out, mapDoubleLookups(fn)...)
	out = append(out, sliceContainsIndexDouble(fn)...)
	return out
}

// mapDoubleLookups implements the map variant of SM7.
func mapDoubleLookups(fn *ssa.Function) []Finding {
	// Gather all Lookup instructions and the set of maps updated anywhere.
	updated := make(map[ssa.Value]bool, len(fn.Blocks))
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			if upd, ok := instr.(*ssa.MapUpdate); ok {
				updated[upd.Map] = true
			}
		}
	}
	var lookups []*ssa.Lookup
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			lu, ok := instr.(*ssa.Lookup)
			if !ok {
				continue
			}
			lookups = append(lookups, lu)
		}
	}
	var out []Finding
	seen := map[*ssa.Lookup]bool{}
	for _, first := range lookups {
		if !first.CommaOk || seen[first] {
			continue
		}
		for _, second := range lookups {
			if second.CommaOk || first == second {
				continue
			}
			if second.X != first.X || second.Index != first.Index {
				continue
			}
			if updated[first.X] {
				continue // map mutated anywhere: conservatively stay silent
			}
			// The second must be dominated by the first (so the first's result
			// is available); ordering by block, second's block must be reachable
			// from first's block (first dominates second).
			if !dominates(first.Block(), second.Block()) {
				continue
			}
			if seen[second] {
				continue
			}
			seen[first] = true
			seen[second] = true
			out = append(out, Finding{
				Pos:     first.Pos(),
				Rule:    "SM7",
				Message: "redundant map lookup; combine with the preceding comma-ok check (use the value the ok already fetched)",
			})
			break
		}
	}
	return out
}

// sliceContainsIndexDouble implements the slices.Contains -> slices.Index variant.
func sliceContainsIndexDouble(fn *ssa.Function) []Finding {
	type call struct {
		instr  *ssa.Call
		origin string
		slice  ssa.Value
		needle ssa.Value
	}
	var contains, indexes []call
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			c, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			origin, ok := calleeOrigin(&c.Call)
			if !ok {
				continue
			}
			if len(c.Call.Args) != 2 {
				continue
			}
			entry := call{instr: c, origin: origin, slice: c.Call.Args[0], needle: c.Call.Args[1]}
			switch {
			case sm7ContainsNames[origin]:
				contains = append(contains, entry)
			case sm7IndexNames[origin]:
				indexes = append(indexes, entry)
			}
		}
	}
	var out []Finding
	for _, c := range contains {
		for _, ix := range indexes {
			if c.slice != ix.slice || c.needle != ix.needle {
				continue
			}
			if !dominates(c.instr.Block(), ix.instr.Block()) {
				continue
			}
			out = append(out, Finding{
				Pos:     c.instr.Pos(),
				Rule:    "SM7",
				Message: "redundant linear scan; slices.Contains then slices.Index re-scans — use Index directly and test its result against -1",
			})
			break
		}
	}
	return out
}

// dominates reports whether a dominates b (a is on every path from entry to b).
// A block dominates itself.
func dominates(a, b *ssa.BasicBlock) bool {
	for x := b; x != nil; x = x.Idom() {
		if x == a {
			return true
		}
	}
	return false
}
