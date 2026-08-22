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
	out, _, err := trustInit("../report/testdata/trustinit", false)
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
	out, _, err := trustInit("../report/testdata/trustinit", false)
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
	out, _, err := trustInit("../report/testdata/trustinit", false)
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
	out, _, err := trustInit("../report/testdata/trustinit", false)
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
	a, _, err := trustInit("../report/testdata/trustinit", false)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := trustInit("../report/testdata/trustinit", false)
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
	out, _, err := trustInit("../report/testdata/trustinit", false)
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

// TestTrustInitOmitsFirstPartyGenerics is the product half of the 2026-08-12
// review's F2, pinned end to end through a real module rather than through a
// hand-built document.
//
// A trust file is for code the user CANNOT edit. Offering them a generic
// function from their own module is wrong twice: it contradicts the premise,
// and it points at the wrong tool — //bigo:cost lives beside the code and is
// checked against the signature, which an assertion in a side file is not.
//
// The control matters as much as the assertion: os.Getenv must still be
// offered, or a fix that suppressed every first-party-looking key would pass.
func TestTrustInitOmitsFirstPartyGenerics(t *testing.T) {
	out, _, err := trustInit("../report/testdata/trustinit", false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "trustinit.Drain") {
		t.Errorf("output offers a first-party generic from the user's own module:\n%s", out)
	}
	if !strings.Contains(out, "os.Getenv") {
		t.Errorf("control failed: os.Getenv is genuinely unpriced and must still be offered:\n%s", out)
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

	out, _, err := trustInit(dir, false)
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
	got := headerFor(true)
	for _, want := range []string{
		"MEASURED, not predicted",
		"UPPER BOUND",
		"CONSTANT bound",
		"resolve at the call site",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("header does not mention %q:\n%s", want, got)
		}
	}
}

// TestReadmeCarriesTheSameCaveats extends the header test to the document a
// user actually reads first (2026-08-12 review F3).
//
// v1.44.2 existed because a measured caveat shipped in a PR description instead
// of in the file, and its durable half was a test on the generated header. The
// README kept describing the PRE-measurement semantics for two releases anyway,
// because the test was scoped to the one file the fix had touched. The lesson
// was "checking a claim a user will ACT ON is worth a test", and the README
// carries the same claim to more people than the header does.
//
// Reading the file from disk is the point: a constant in this package would
// pin a copy, not the thing shipped.
func TestReadmeCarriesTheSameCaveats(t *testing.T) {
	b, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	readme := string(b)
	for _, want := range []string{
		// F3 — the counts are measured and bounded.
		"measured, not predicted",
		"upper bound, and only a constant bound reaches it",
		"resolve at the call site",
		// F6 — generated code is scored.
		"Generated code counts",
		// F5 — the gate can be silenced by a trust edit.
		"never fails `-fail-on`",
	} {
		if !strings.Contains(readme, want) {
			t.Errorf("README does not state %q — the header says it and the README is where a user reads it first", want)
		}
	}
}

// TestHeaderSaysWhichNumberItCarries is the other half, added 2026-08-12
// (review F4). The measured caveat was asserted UNCONDITIONALLY while the
// measurement pass failed soft to the predicted count, so on that path the file
// claimed a property of itself that was false.
//
// Both directions, because a fix that always said "predicted" would satisfy a
// one-sided test and throw away the number the pass exists to produce.
func TestHeaderSaysWhichNumberItCarries(t *testing.T) {
	measured, predicted := headerFor(true), headerFor(false)
	if strings.Contains(measured, "PREDICTED") {
		t.Error("a measured file warns about predicted counts")
	}
	if !strings.Contains(predicted, "PREDICTED, not measured") {
		t.Errorf("an unmeasured file does not say so:\n%s", predicted)
	}
	if strings.Contains(predicted, "MEASURED, not predicted") {
		t.Errorf("an unmeasured file still claims its counts are measured:\n%s", predicted)
	}
	// Both must carry the notes that do not depend on which number it is.
	for _, h := range []string{measured, predicted} {
		if !strings.Contains(h, "Only single blockers are listed") {
			t.Error("header lost the single-blocker note")
		}
		if !strings.Contains(h, "GENERATED CODE COUNTS") {
			t.Error("header does not warn that generated code is scored (review F6)")
		}
	}
}

// bigo trust init -kata used to fail outright with "flag provided but not
// defined", and that was the smaller half of the defect: trustInit built its
// resolver with no overlay, so even once the flag parsed, the blocker ranking
// answered the DEFAULT cost model. A kata user was being told which keys to
// reason about under a model they had explicitly not chosen.
func TestInitMainAcceptsKata(t *testing.T) {
	if code := InitMain([]string{"-kata", "-C", "../report/testdata/trustinitkata"}); code != 0 {
		t.Errorf("exit = %d with -kata, want 0", code)
	}
}

// The ranking must answer the model the user asked for. Under the default
// model the bufio write is a real, offerable trust candidate. Under -kata the
// profile already prices it, so the function is not blocked at all and offering
// the key would spend the user's reasoning on a line that changes nothing --
// exactly what TestTrustInitOmitsAlreadyCuratedKeys refuses for curated keys.
func TestTrustInitUnderKataOmitsProfiledKeys(t *testing.T) {
	const key = "(*bufio.Writer).WriteString"

	base, _, err := trustInit("../report/testdata/trustinitkata", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(base, key) {
		t.Fatalf("default model must offer %s, got:\n%s", key, base)
	}

	kataText, _, err := trustInit("../report/testdata/trustinitkata", true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(kataText, key) {
		t.Errorf("-kata must not offer %s: the profile already prices it\ngot:\n%s", key, kataText)
	}
}
