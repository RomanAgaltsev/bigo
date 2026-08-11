package trust

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/assume"
	"github.com/RomanAgaltsev/bigo/internal/report"
)

// TestTrustInitRanksByGraduationCount: the number beside each key is the count
// of the user's functions that key ALONE unblocks — not call sites. This is
// ROADMAP §1's most-repeated principle at the point a user reads a number:
// fmt was 8,367 sites, 744 sole-blocker functions, 298 truthfully priceable.
func TestTrustInitRanksByGraduationCount(t *testing.T) {
	out, err := trustInit("../report/testdata/trustinit")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "os.Getenv") {
		t.Errorf("output does not offer os.Getenv:\n%s", out)
	}
	if !strings.Contains(out, "2 of your functions") {
		t.Errorf("output does not report the graduation count of 2:\n%s", out)
	}
}

// TestTrustInitOmitsUnkeyableBlockers: an interface method cannot be named in a
// trust file, so it must not be offered. The generator filters on the presence
// of the callee key rather than on a hand-maintained list.
func TestTrustInitOmitsUnkeyableBlockers(t *testing.T) {
	out, err := trustInit("../report/testdata/trustinit")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, ").Read") {
		t.Errorf("output offers an interface method, which no trust file can express:\n%s", out)
	}
}

// TestTrustInitProposesNoBound: bigo suggests KEYS. A tool that guessed bounds
// would be inventing the thing it exists to check, so the placeholder must not
// parse if someone uncomments a line without editing it.
func TestTrustInitProposesNoBound(t *testing.T) {
	out, err := trustInit("../report/testdata/trustinit")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "O(...)") {
		t.Fatalf("no placeholder in output:\n%s", out)
	}
	if _, err := assume.ParseText("os.Getenv O(...)\n"); err == nil {
		t.Error("the placeholder parses as a bound; it must fail loudly if left unedited")
	}
}

// TestTrustInitRoundTrips is the property that stops the generator ever
// emitting something its own loader rejects: uncomment everything, fill in any
// bound, and the result must parse.
func TestTrustInitRoundTrips(t *testing.T) {
	out, err := trustInit("../report/testdata/trustinit")
	if err != nil {
		t.Fatal(err)
	}
	var filled []string
	for _, line := range strings.Split(out, "\n") {
		rest, ok := strings.CutPrefix(line, "# ")
		if !ok || !strings.Contains(rest, "O(...)") {
			continue
		}
		filled = append(filled, strings.Replace(rest, "O(...)", "O(1)", 1))
	}
	if len(filled) == 0 {
		t.Fatal("no candidate lines found to round-trip")
	}
	if _, err := assume.ParseText(strings.Join(filled, "\n") + "\n"); err != nil {
		t.Errorf("generated file does not round-trip: %v\ninput:\n%s", err, strings.Join(filled, "\n"))
	}
}

// TestTrustInitIsDeterministic: two runs over one module are byte-identical, so
// a regenerated file produces a reviewable diff rather than noise.
func TestTrustInitIsDeterministic(t *testing.T) {
	a, err := trustInit("../report/testdata/trustinit")
	if err != nil {
		t.Fatal(err)
	}
	b, err := trustInit("../report/testdata/trustinit")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Error("two runs differ")
	}
}

// TestInitMainRefusesToClobber: a trust file is hand-curated and carries the
// justifications someone reasoned out. Regenerating over it would destroy
// exactly the part that took thought, so overwriting needs -force.
func TestInitMainRefusesToClobber(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bigo.trust")
	const curated = "# os.Getenv is O(1): it reads a preparsed environ slice.\nos.Getenv O(1)\n"
	if err := os.WriteFile(path, []byte(curated), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := InitMain([]string{"-C", "../report/testdata/trustinit", "-o", path}); code != 1 {
		t.Errorf("exit = %d, want 1 — an existing file must not be clobbered", code)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != curated {
		t.Error("the curated file was overwritten")
	}
	if code := InitMain([]string{"-C", "../report/testdata/trustinit", "-o", path, "-force"}); code != 0 {
		t.Errorf("exit = %d with -force, want 0", code)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) == curated {
		t.Error("-force did not overwrite")
	}
}

// TestTrustInitOmitsAlreadyCuratedKeys is the defect found the first time this
// generator was pointed at real repositories, hours after it shipped.
//
// A blocker reads "unresolved ARGUMENT SIZE at call to strconv.Atoi" when the
// curated table already prices the callee and only the argument's size is
// unknown. The key is real, so the generator offered it — but the plain table
// outranks a trust entry, so writing one is shadowed and contributes nothing.
// The user gets a warning and no verdict change, for a line they reasoned about.
//
// Filtering asks the cost table whether it already answers the key, rather than
// parsing the cause sentence: parsing prose is what the callee key was added to
// avoid, and a table-driven answer stays correct as entries are added.
func TestTrustInitOmitsAlreadyCuratedKeys(t *testing.T) {
	out, err := trustInit("../report/testdata/trustinit")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "strconv.Atoi") {
		t.Errorf("output offers strconv.Atoi, which the curated table already prices — a trust entry for it is shadowed:\n%s", out)
	}
	// The control: an UNPRICED callee is still offered, so the filter cannot be
	// satisfied by suppressing everything.
	if !strings.Contains(out, "os.Getenv") {
		t.Errorf("output no longer offers os.Getenv, which is genuinely unpriced:\n%s", out)
	}
}

// TestTrustInitCountsAreMeasuredNotPredicted pins the property the whole
// measurement pass exists to establish: the number beside a key equals what
// asserting that key actually delivers.
//
// It is checked by doing the thing the user would do — assert the key, count
// what became verifiable — and comparing against what the generator advertised.
// First contact found the predicted number overstating reality by a third on
// the largest real row (goldmark's bytes.Repeat, advertised 19, delivers 13).
func TestTrustInitCountsAreMeasuredNotPredicted(t *testing.T) {
	const dir = "../report/testdata/trustinit"

	out, err := trustInit(dir)
	if err != nil {
		t.Fatal(err)
	}
	advertised := countFor(t, out, "os.Getenv")
	if advertised == 0 {
		t.Fatalf("os.Getenv not advertised:\n%s", out)
	}

	// Now do what the user would do, independently of the generator.
	l, err := report.LoadModule(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	base, err := l.Document(report.Options{Version: "t"})
	if err != nil {
		t.Fatal(err)
	}
	es, err := assume.ParseText("os.Getenv O(1)\n")
	if err != nil {
		t.Fatal(err)
	}
	head, err := l.Document(report.Options{Version: "t", Assume: assume.NewSet(es)})
	if err != nil {
		t.Fatal(err)
	}

	wasTop := map[string]bool{}
	for _, f := range base.Functions {
		if f.Time.Top {
			wasTop[f.Package+f.File+f.Func] = true
		}
	}
	delivered := 0
	for _, f := range head.Functions {
		if !f.Time.Top && wasTop[f.Package+f.File+f.Func] {
			delivered++
		}
	}

	if advertised != delivered {
		t.Errorf("generator advertised %d, asserting the key delivers %d — the count must be measured, not predicted",
			advertised, delivered)
	}
}

// countFor extracts the advertised count for a key from generated output.
func countFor(t *testing.T, out, key string) int {
	t.Helper()
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "# "+key+" O(") {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(lines[i-1], "# %d of your functions", &n); err != nil {
			t.Fatalf("could not read the count above %q: %v", key, err)
		}
		return n
	}
	return 0
}

// TestHeaderCarriesTheMeasuredCaveat pins the two things the header must tell a
// user about the number beside each key: that it is measured, and that it is an
// upper bound only a constant bound reaches.
//
// It exists because the header did NOT say either for one release. The edit
// that was supposed to add them was scripted, matched nothing, and reported
// success — gofmt and the compiler were both happy, no test looked at the text,
// and the claim shipped in a PR description instead of in the file.
//
// Prose that carries a measured caveat is load-bearing, so it gets a test.
func TestHeaderCarriesTheMeasuredCaveat(t *testing.T) {
	for _, want := range []string{
		"MEASURED, not predicted",
		"UPPER BOUND",
		"CONSTANT bound",
		"resolve at the call site",
	} {
		if !strings.Contains(trustHeader, want) {
			t.Errorf("header does not mention %q:\n%s", want, trustHeader)
		}
	}
}
