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
	if doc.SchemaVersion != "1.2.0" {
		t.Errorf("schema = %s, want 1.2.0", doc.SchemaVersion)
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
