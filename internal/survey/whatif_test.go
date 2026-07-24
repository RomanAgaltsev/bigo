package survey

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/report"
)

func wfn(name string, top bool, prov string) report.Function {
	f := report.Function{Package: "p", Func: name, File: "f.go", Line: len(name), Provenance: prov}
	if top {
		f.Time = report.BoundJSON{Top: true}
	} else {
		f.Time = report.BoundJSON{Str: "O(1)"}
	}
	return f
}

func TestCompareCountsTaintedGraduationsOnly(t *testing.T) {
	base := report.Document{Module: "p", Functions: []report.Function{wfn("a", true, ""), wfn("bb", true, ""), wfn("ccc", false, "")}}
	cand := report.Document{Module: "p", Functions: []report.Function{wfn("a", false, report.ProvenanceTainted), wfn("bb", true, ""), wfn("ccc", false, "")}}
	grad, hand, blocked := compareDocs(base, cand, func(string) bool { return false })
	if blocked != "" || grad != 1 || hand != 1 {
		t.Fatalf("grad=%d hand=%d blocked=%q, want 1/1/none", grad, hand, blocked)
	}
}

func TestCompareBlocksUntaintedGraduation(t *testing.T) {
	base := report.Document{Module: "p", Functions: []report.Function{wfn("a", true, "")}}
	cand := report.Document{Module: "p", Functions: []report.Function{wfn("a", false, "")}} // graduated with NO taint
	_, _, blocked := compareDocs(base, cand, func(string) bool { return false })
	if !strings.Contains(blocked, "unattributed") {
		t.Fatalf("blocked = %q, want unattributed-graduation block", blocked)
	}
}

func TestCompareBlocksLostBound(t *testing.T) {
	base := report.Document{Module: "p", Functions: []report.Function{wfn("a", false, "")}}
	cand := report.Document{Module: "p", Functions: []report.Function{wfn("a", true, "")}}
	_, _, blocked := compareDocs(base, cand, func(string) bool { return false })
	if !strings.Contains(blocked, "lost") {
		t.Fatalf("blocked = %q, want lost-bound block", blocked)
	}
}

func TestCompareBlocksPopulationMismatch(t *testing.T) {
	base := report.Document{Module: "p", Functions: []report.Function{wfn("a", true, "")}}
	cand := report.Document{Module: "p", Functions: []report.Function{wfn("a", true, ""), wfn("xx", true, "")}}
	_, _, blocked := compareDocs(base, cand, func(string) bool { return false })
	if !strings.Contains(blocked, "population") {
		t.Fatalf("blocked = %q, want population-mismatch block", blocked)
	}
}

func TestCompareExcludesAssumedTargets(t *testing.T) {
	base := report.Document{Module: "p", Functions: []report.Function{wfn("a", true, "")}}
	cand := report.Document{Module: "p", Functions: []report.Function{wfn("a", false, report.ProvenanceAssumed)}}
	grad, _, blocked := compareDocs(base, cand, func(string) bool { return false })
	if blocked != "" || grad != 0 {
		t.Fatalf("grad=%d blocked=%q — an assumed target must not count and must not block", grad, blocked)
	}
}

func TestRunWhatIfOverFixture(t *testing.T) {
	abs, err := filepath.Abs("../report/testdata/assumefix")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fix.assume"), []byte("os.Getenv O(1)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cands := filepath.Join(dir, "cands.json")
	if err := os.WriteFile(cands, []byte(`{"candidates":[{"name":"getenv","file":"fix.assume"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	wc, sets, err := LoadWhatIfConfig(cands)
	if err != nil {
		t.Fatal(err)
	}
	rep := RunWhatIf(Config{Targets: []TargetConfig{{Name: "assumefix", Path: abs}}}, wc, sets, "test", t.Logf)
	if len(rep.Results) != 1 {
		t.Fatalf("results = %+v", rep.Results)
	}
	res := rep.Results[0]
	if res.Blocked != "" {
		t.Fatalf("blocked: %s", res.Blocked)
	}
	// Blocked and Caller graduate; Clean was already bounded.
	if res.Graduated != 2 || res.GraduatedHand != 2 {
		t.Fatalf("graduated = %d/%d, want 2/2", res.Graduated, res.GraduatedHand)
	}
	if !strings.Contains(RenderWhatIf(rep), "| getenv | 2 | 2 |") {
		t.Fatal("render missing the result row")
	}
}

func TestLoadWhatIfConfigErrors(t *testing.T) {
	dir := t.TempDir()
	writeCands := func(body string) string {
		p := filepath.Join(dir, "c.json")
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if _, _, err := LoadWhatIfConfig(writeCands(`{"candidates":[]}`)); err == nil {
		t.Error("empty candidates must error")
	}
	if _, _, err := LoadWhatIfConfig(writeCands(`{"candidates":[{"name":"x","file":"nosuch.assume"}]}`)); err == nil {
		t.Error("missing assumption file must error")
	}
	if _, _, err := LoadWhatIfConfig(writeCands(`{"candidates":[{"name":"","file":"a"}]}`)); err == nil {
		t.Error("empty candidate name must error")
	}
}
