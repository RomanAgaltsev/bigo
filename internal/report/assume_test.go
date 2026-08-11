package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/assume"
)

func TestCollectWithAssumptions(t *testing.T) {
	set, err := assume.Load("testdata/assumefix/fix.assume")
	if err != nil {
		t.Fatal(err)
	}
	var warns []string
	doc, err := Collect("testdata/assumefix", nil, Options{
		Version: "test", Now: fixedNow,
		Assume: set, Warn: func(w string) { warns = append(warns, w) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != "1.3.0" {
		t.Errorf("schema = %s, want 1.3.0", doc.SchemaVersion)
	}
	byName := map[string]Function{}
	for _, f := range doc.Functions {
		byName[f.Func] = f
	}
	for _, name := range []string{"Blocked", "Caller"} {
		f := byName[name]
		if f.Time.Top {
			t.Errorf("%s still top under assumption", name)
		}
		if f.Provenance != ProvenanceTainted {
			t.Errorf("%s provenance = %q, want %q", name, f.Provenance, ProvenanceTainted)
		}
	}
	if f := byName["Clean"]; f.Provenance != "" {
		t.Errorf("Clean provenance = %q, want absent", f.Provenance)
	}
	if len(doc.Assumptions) != 1 || doc.Assumptions[0].Key != "os.Getenv" || doc.Assumptions[0].Bound != "O(1)" {
		t.Errorf("assumptions surface = %+v", doc.Assumptions)
	}
	if len(warns) != 0 {
		t.Errorf("warnings = %v, want none", warns)
	}
}

func TestCollectWithoutAssumptionsHasNoProvenance(t *testing.T) {
	doc, err := Collect("testdata/assumefix", nil, Options{Version: "test", Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"provenance", "assumptions"} {
		if strings.Contains(string(data), banned) {
			t.Errorf("plain document contains %q — 1.1.0 byte-compatibility broken", banned)
		}
	}
}

func TestCollectUnmatchedAssumptionKeyFails(t *testing.T) {
	es, err := assume.ParseText("example.com/assumefix.NoSuch O(1)\n")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Collect("testdata/assumefix", nil, Options{Version: "test", Now: fixedNow, Assume: assume.NewSet(es)})
	if err == nil || !strings.Contains(err.Error(), "NoSuch") {
		t.Fatalf("err = %v, want hard unmatched-key error", err)
	}
}

func TestLoadOnceDocumentTwice(t *testing.T) {
	l, err := LoadModule("testdata/assumefix", nil)
	if err != nil {
		t.Fatal(err)
	}
	base, err := l.Document(Options{Version: "test", Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Collect("testdata/assumefix", nil, Options{Version: "test", Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	bj, _ := json.Marshal(base)
	pj, _ := json.Marshal(plain)
	if string(bj) != string(pj) {
		t.Fatal("Document over a shared load differs from a fresh Collect")
	}
	set, err := assume.Load("testdata/assumefix/fix.assume")
	if err != nil {
		t.Fatal(err)
	}
	withA, err := l.Document(Options{Version: "test", Now: fixedNow, Assume: set})
	if err != nil {
		t.Fatal(err)
	}
	if len(withA.Assumptions) != 1 {
		t.Fatal("assumption document missing its trust surface")
	}
	// The shared program must not leak assumption state back into a plain run.
	again, err := l.Document(Options{Version: "test", Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	aj, _ := json.Marshal(again)
	if string(aj) != string(pj) {
		t.Fatal("plain Document after an assumption Document differs — state leaked through the shared SSA program")
	}
}

func TestCollectInModuleAssumedTarget(t *testing.T) {
	es, err := assume.ParseText("example.com/assumefix.Blocked O(1)\n")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Collect("testdata/assumefix", nil, Options{Version: "test", Now: fixedNow, Assume: assume.NewSet(es)})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Function{}
	for _, f := range doc.Functions {
		byName[f.Func] = f
	}
	// The target itself keeps its INFERRED (top) bound — directive parity —
	// and is marked "assumed" so no instrument counts it as a graduation.
	if f := byName["Blocked"]; !f.Time.Top || f.Provenance != ProvenanceAssumed {
		t.Errorf("Blocked = top:%v provenance:%q, want top under provenance %q", f.Time.Top, f.Provenance, ProvenanceAssumed)
	}
	// Its caller graduates through the assumption and is tainted.
	if f := byName["Caller"]; f.Time.Top || f.Provenance != ProvenanceTainted {
		t.Errorf("Caller = top:%v provenance:%q", f.Time.Top, f.Provenance)
	}
}

func TestUnmatchedKeysReportedWhenCallbackSet(t *testing.T) {
	es, err := assume.ParseText("os.Getenv O(1)\nexample.com/assumefix.NoSuch O(1)\n")
	if err != nil {
		t.Fatal(err)
	}
	var reported []string
	doc, err := Collect("testdata/assumefix", nil, Options{
		Version: "test", Now: fixedNow, Assume: assume.NewSet(es),
		AssumeUnmatchedKeys: func(keys []string) { reported = append(reported, keys...) },
	})
	if err != nil {
		t.Fatalf("with the callback set an unmatched key must not fail the run: %v", err)
	}
	if len(reported) != 1 || reported[0] != "example.com/assumefix.NoSuch" {
		t.Fatalf("reported = %v, want the one absent key", reported)
	}
	// The matched key still did its job.
	for _, f := range doc.Functions {
		if f.Func == "Caller" && f.Time.Top {
			t.Error("Caller did not graduate — the matched assumption was dropped too")
		}
	}
}

// TestCrossPackageParamSummaryTaint is the half of review F2 that the
// same-package unit pin cannot cover. Here the parametric function lives in
// another package, so this resolver never runs InferTop on it and taint[Run]
// is never set — a memo-hit guard alone would leave B untainted forever.
func TestCrossPackageParamSummaryTaint(t *testing.T) {
	set, err := assume.Load("testdata/paramtaint/fix.assume")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Collect("testdata/paramtaint", nil, Options{Version: "test", Now: fixedNow, Assume: set})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Function{}
	for _, f := range doc.Functions {
		byName[f.Func] = f
	}
	for _, name := range []string{"A", "B"} {
		f, ok := byName[name]
		if !ok {
			t.Fatalf("%s missing from document", name)
		}
		if f.Time.Top {
			t.Errorf("%s is top; the assumption should have graduated it", name)
		}
		if f.Provenance != ProvenanceTainted {
			t.Errorf("%s provenance = %q, want %q — taint must not depend on which consumer ran first",
				name, f.Provenance, ProvenanceTainted)
		}
	}
}

// TestCauseCarriesCalleeKey pins schema 1.3.0's callee field. Without it
// `bigo trust init` would have to parse the human cause sentence to name a
// trust key, which the near-frontier design forbids: that parser was written
// once, during the fmt probe, and got it wrong.
//
// The ABSENCE of the field is equally load-bearing — see the interface case
// below. CalleeKey fails exactly for interface dispatch and function values,
// which are exactly the blockers a trust file cannot express, so the generator
// filters on presence and needs no second list of what is inexpressible.
func TestCauseCarriesCalleeKey(t *testing.T) {
	doc, err := Collect("testdata/trustinit", nil, Options{Version: "test", Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != "1.3.0" {
		t.Errorf("schema_version = %q, want 1.3.0", doc.SchemaVersion)
	}
	byName := map[string]Function{}
	for _, f := range doc.Functions {
		byName[f.Func] = f
	}

	// A static call to an unpriced stdlib function: the key is nameable.
	static := byName["BlockedByStatic"]
	if len(static.Causes) == 0 {
		t.Fatal("BlockedByStatic has no causes")
	}
	if got := static.Causes[0].Callee; got != "os.Getenv" {
		t.Errorf("callee = %q, want %q", got, "os.Getenv")
	}

	// An interface method call: no static callee, so no key, so no trust entry
	// is possible and the field must be absent.
	iface := byName["BlockedByInterface"]
	if len(iface.Causes) == 0 {
		t.Fatal("BlockedByInterface has no causes")
	}
	if got := iface.Causes[0].Callee; got != "" {
		t.Errorf("callee = %q, want empty — an interface method cannot be keyed", got)
	}
}
