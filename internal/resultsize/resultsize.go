// Package resultsize curates the RESULT length of stdlib calls that return a
// collection, expressed as "bounded by the size of argument i".
//
// # Why this is its own package
//
// This is a SIZE fact, not a COST fact, and the two tables cannot live
// together: internal/costtable already imports internal/sizefacts, and
// sizefacts is the consumer that needs this. A leaf package with no bigo
// dependencies breaks the cycle and keeps each table single-purpose.
//
// # What an entry claims
//
// An entry says: len(result) is O(size of args[i]). It is an UPPER bound, the
// sound direction — over-approximating degrades a Within to Unknown and never
// fabricates a tighter-than-true bound.
//
// Entries are asymptotic, so an additive constant is not a defect:
// strings.Split(s, sep) returns at most len(s)+1 substrings, and the +1 (the
// empty-input case, which yields one empty element) vanishes in O(len(s)).
// sizefacts.UpperExtent already discards constants on the same reasoning.
//
// # What must NOT be here
//
// A result whose length is a PRODUCT of two sizes, because an extent is one
// variable: strings.Repeat(s, n) has length len(s)*n and stays unnamed. A
// result whose length depends on a callback, on file or network content, or on
// anything the caller cannot name, likewise. Refusing is always safe — the
// consumer falls back to ⊤.
package resultsize

import (
	"golang.org/x/tools/go/ssa"
)

// ArgIndex returns the index of the argument whose size upper-bounds the
// length of c's result. ok=false means the callee has no curated result size,
// which is the answer for everything not listed below.
func ArgIndex(c *ssa.CallCommon) (int, bool) {
	callee := c.StaticCallee()
	if callee == nil {
		return 0, false
	}
	if orig := callee.Origin(); orig != nil {
		callee = orig
	}
	if callee.Pkg == nil || callee.Pkg.Pkg == nil || callee.Signature.Recv() != nil {
		return 0, false
	}
	i, ok := table[callee.Pkg.Pkg.Path()+"."+callee.Name()]
	return i, ok
}

// table maps "pkgpath.Name" to the argument index whose size bounds the
// result's length.
//
// Every entry below splits or transforms ONE string or byte slice, and the
// pieces are drawn from that operand — so the piece COUNT (for the splitters)
// and the byte length (for the transformers) are both bounded by the operand's
// length. Read from the stdlib's own documented contracts, per the Trim rule.
// Every entry is REACHABLE end to end and pinned by a test: the callee also has
// a cost-table entry, and its result is a SLICE, so `range` over it lowers to an
// index loop whose limit is len(result). Entries failing either condition were
// deliberately left out rather than written on faith — see the two deferrals
// below.
var table = map[string]int{
	// At most len(s)+1 pieces, and exactly len(s)+1 only for the degenerate
	// empty separator. The piece COUNT is what a caller ranges over.
	"strings.Split":  0,
	"strings.SplitN": 0, // n caps it lower; len(s) is still an upper bound
	"strings.Fields": 0,
}

// Two DEFERRALS, recorded so neither reads as an oversight:
//
//  1. String-returning transformers — strings.ToLower, ToUpper, TrimSpace. Their
//     result length really is bounded by len(s), but `range` over a STRING
//     lowers to *ssa.Range/*ssa.Next, which tripcount answers with ruleRangeNext
//     — a path that never consults sizefacts.lenExtent and so never reaches this
//     table. Adding entries would change nothing. Wiring that path is a separate
//     change, and nothing measured asks for it.
//
//  2. Unpriced splitters — strings.SplitAfter, SplitAfterN, and the whole bytes
//     family. costtable prices only bytes.Contains/Index/Count, so a call to any
//     of the others is ⊤ on COST no matter what its result measures, and the row
//     stays unverifiable. A size fact here would be untestable end to end, which
//     is how unmeasured surface accumulates. Add each one WITH its cost entry,
//     if a measured row ever asks.
//
// And the permanent refusals, which are refusals rather than deferrals:
//
//   - strings.Repeat(s, n): length is len(s)*n, a PRODUCT. An extent is one
//     variable, so naming either factor alone would be a wrong bound.
//   - strings.Join(elems, sep): length is the SUM of element lengths, which the
//     grammar cannot name (costtable's unit-element axiom — "the total CONTENT
//     of a collection has no name").
//   - strings.Map / strings.FieldsFunc: a callback decides the result.
//   - os.ReadFile / io.ReadAll: the result is file or stream content, not a
//     function of any argument's size.
//   - slices.Collect / maps.Keys: the source is an iterator, whose yield count
//     is costtable's iteratorProducers table's business, not this one.
