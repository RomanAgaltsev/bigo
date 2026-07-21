package survey

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// generatedMarker is Go's canonical generated-code line, per
// https://go.dev/s/generatedcode. Matched as a WHOLE line: the convention is a
// declaration, not a phrase that happens to appear in prose.
var generatedMarker = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// generatedDetector answers whether a reported function's file is machine
// generated, reading each file at most once per run.
//
// Why this exists: firstParty is a module-path-prefix test, so a generated
// .pb.go inside the module is first-party by construction. That is correct as
// written and misleading as a WORK QUEUE — nobody hand-tunes generated
// protobuf accessors, and their unverifiability is usually the CORRECT answer.
// The 2026-07-21 (*sync.Once).Do probe measured 239 of that class's 326
// sole-blocker functions as generated code whose ⊤ is right.
type generatedDetector struct {
	root string
	memo map[string]bool
}

// newGeneratedDetector returns a detector rooted at a target's module
// directory.
func newGeneratedDetector(root string) *generatedDetector {
	return &generatedDetector{root: root, memo: map[string]bool{}}
}

// isGenerated reports whether relFile carries the marker before its package
// clause. relFile is module-relative with forward slashes, as report.Collect's
// relPath emits it.
//
// A nil detector, an empty path, and any file that cannot be read or scanned
// all answer FALSE — hand-written. The asymmetry is deliberate and
// load-bearing: a false EXCLUSION silently drops real work out of the queue
// where nobody will look for it again, while a false inclusion merely leaves
// noise visible in a table someone reads. When in doubt, keep it in the queue.
func (d *generatedDetector) isGenerated(relFile string) bool {
	if d == nil || relFile == "" {
		return false
	}
	if v, ok := d.memo[relFile]; ok {
		return v
	}
	v := scanGenerated(filepath.Join(d.root, filepath.FromSlash(relFile)))
	d.memo[relFile] = v
	return v
}

// scanGenerated reads the leading comment block only, stopping at the first
// non-comment, non-blank line.
//
// Stopping there is both cheaper and STRICTER than scanning the whole file:
// the convention requires the marker before the first non-comment line, so a
// matching line further down is prose, not a declaration. It also means a
// generated file's megabyte-long rawDesc literals are never read, since they
// sit after the package clause.
func scanGenerated(path string) bool {
	// #nosec G304 -- path is a source file named by bigo's own report document,
	// under a repository path from survey/targets.json, an operator-authored
	// file committed to this repo. The survey is a manual developer tool that
	// never runs in CI or on untrusted input.
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if generatedMarker.MatchString(line) {
			return true
		}
		if !strings.HasPrefix(line, "//") {
			return false // package clause, build constraint block, or code
		}
	}
	// A scan error (including a line over bufio's limit) lands here and answers
	// hand-written, per the conservative default above.
	return false
}
