// Package report builds the machine-readable bigo document — the Stage-1
// integration contract of the ecosystem spec. Output plumbing only: all
// verdict logic is consumed from bound/engine/callsummary, never reimplemented.
//
// Stability: SchemaVersion is semver, independent of bigo's release version.
// Changes within a major are additive-only; no field is ever reinterpreted.
// The normative artifact is schema/report.schema.json.
package report

import (
	"sort"

	"golang.org/x/tools/go/ssa"

	"github.com/RomanAgaltsev/bigo/internal/annotation"
	"github.com/RomanAgaltsev/bigo/internal/bound"
	"github.com/RomanAgaltsev/bigo/internal/normalize"
)

// SchemaVersion is the version of the document format.
// 1.1.0 added the top-level smells array (additive; 1.0.0 documents remain valid).
const SchemaVersion = "1.1.0"

// Document is one `bigo json` run over one module.
type Document struct {
	SchemaVersion string       `json:"schema_version"`
	BigoVersion   string       `json:"bigo_version"`
	Module        string       `json:"module"`
	Generated     string       `json:"generated"` // RFC 3339, UTC
	Functions     []Function   `json:"functions"`
	Trusted       []TrustEntry `json:"trusted,omitempty"`

	// Smells are advisory findings, deliberately a top-level array rather than
	// a field on Function: the document mirrors the engine's firewall, where a
	// smell can never influence a verdict. Present since schema 1.1.0.
	Smells []SmellJSON `json:"smells,omitempty"`
}

// Function is one analyzed function or method.
type Function struct {
	Package  string      `json:"package"`
	Func     string      `json:"func"`
	Receiver string      `json:"receiver,omitempty"` // e.g. "*Tree"; empty for plain functions
	File     string      `json:"file"`               // module-relative, forward slashes
	Line     int         `json:"line"`
	Time     BoundJSON   `json:"time"`
	Causes   []CauseJSON `json:"causes,omitempty"` // present exactly when time.top
	Budget   *BudgetJSON `json:"budget,omitempty"` // //bigo:max, when declared
	Space    *SpaceJSON  `json:"space,omitempty"`  // //bigo:space, when declared
	Trust    []string    `json:"trust,omitempty"`  // raw //bigo:cost / //bigo:ignore on this decl
}

// BoundJSON is an asymptotic bound: top (unverifiable), or a canonical string
// plus the antichain of poly-log monomials. A monomial maps each size variable
// to its exponents; the empty object is O(1).
type BoundJSON struct {
	Top   bool                    `json:"top,omitempty"`
	Str   string                  `json:"str,omitempty"`
	Terms []map[string]FactorJSON `json:"terms,omitempty"`
}

// FactorJSON is one variable's exponents within a monomial: v^pow · (log v)^log.
type FactorJSON struct {
	Pow int `json:"pow,omitempty"`
	Log int `json:"log,omitempty"`
}

// CauseJSON is one reason a bound is unverifiable (engine.Cause, serialized).
type CauseJSON struct {
	Kind   string `json:"kind"` // engine.CauseKind string: call, defer, go, loop, irreducible, nobody
	Detail string `json:"detail"`
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
}

// BudgetJSON is a declared budget and its verdict. Verdict vocabulary:
// within | exceeds | unverifiable | invalid (budget failed to normalize).
type BudgetJSON struct {
	Raw     string     `json:"raw"`             // the directive as written
	Bound   *BoundJSON `json:"bound,omitempty"` // normalized budget; nil when invalid
	Verdict string     `json:"verdict"`
}

// SpaceJSON is the //bigo:space picture: heap (total-allocation upper bound,
// proves Within only) and stack (true peak depth, proves both).
type SpaceJSON struct {
	Heap   BoundJSON   `json:"heap"`
	Stack  BoundJSON   `json:"stack"`
	Budget *BudgetJSON `json:"budget,omitempty"`
}

// TrustEntry is one //bigo:cost or //bigo:ignore assertion — the document's
// trust surface. Any entry may have influenced any verdict in the document.
type TrustEntry struct {
	Package   string `json:"package"`
	Func      string `json:"func"`
	Receiver  string `json:"receiver,omitempty"`
	Directive string `json:"directive"` // the directive as written
}

// SmellJSON is one advisory smell finding (internal/smell.Finding, serialized).
// Rule is the canonical ID (SM1..SM8); Message carries no "smell(SMn):" prefix
// — that is the analyzer's diagnostic presentation, not part of the data.
type SmellJSON struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
	File    string `json:"file"` // module-relative, forward slashes
	Line    int    `json:"line"`
}

// boundJSON serializes a bound. Terms are sorted by canonical monomial string
// so documents are deterministic.
func boundJSON(b bound.Bound) BoundJSON {
	if b.IsTop() {
		return BoundJSON{Top: true}
	}
	ms := b.Terms()
	type keyed struct {
		key  string
		term map[string]FactorJSON
	}
	ks := make([]keyed, 0, len(ms))
	for _, m := range ms {
		t := map[string]FactorJSON{}
		for _, v := range m.Vars() {
			pow, log := m.FactorOf(v)
			t[string(v)] = FactorJSON{Pow: pow, Log: log}
		}
		ks = append(ks, keyed{key: m.String(), term: t})
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i].key < ks[j].key })
	out := BoundJSON{Str: b.String(), Terms: make([]map[string]FactorJSON, 0, len(ks))}
	for _, k := range ks {
		out.Terms = append(out.Terms, k.term)
	}
	return out
}

// verdictString maps bound.Verdict to the schema vocabulary. bound.Unknown is
// rendered "unverifiable" — never Verdict.String()'s "unknown".
func verdictString(v bound.Verdict) string {
	switch v {
	case bound.Within:
		return "within"
	case bound.Exceeds:
		return "exceeds"
	default:
		return "unverifiable"
	}
}

// budgetJSON records a declared budget. verdictOf compares the normalized
// budget to whatever the caller inferred (time: bound.Check against the
// inferred bound; space: engine.SpaceVerdict). A budget that fails to
// normalize is recorded with Verdict "invalid", never silently dropped.
func budgetJSON(dir annotation.Directive, fn *ssa.Function, verdictOf func(budget bound.Bound) bound.Verdict) *BudgetJSON {
	bj := &BudgetJSON{Raw: dir.Raw}
	budget, err := normalize.Budget(dir, fn)
	if err != nil {
		bj.Verdict = "invalid"
		return bj
	}
	b := boundJSON(budget)
	bj.Bound = &b
	bj.Verdict = verdictString(verdictOf(budget))
	return bj
}
