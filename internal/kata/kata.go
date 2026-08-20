// Package kata supplies the algorithm-kata cost model as a cost-model overlay.
//
// The profile deliberately contradicts the curated cost table and outranks it
// (see callsummary.UseOverlay); it is active only under the -kata flag. Each
// entry is a written claim with a stated axiom — never a wildcard and never
// generated, the same discipline the trust file requires.
package kata

import (
	_ "embed"

	"github.com/RomanAgaltsev/bigo/internal/assume"
)

//go:embed kata.assume
var profile string

//go:embed kata-space.assume
var spaceProfile string

// Profile parses the embedded kata TIME cost model. A malformed profile is a
// hard error, never a skip: a silently dropped entry would make bigo report a
// bound under a cost model the user did not get.
func Profile() (*assume.Set, error) { return parse(profile) }

// SpaceProfile parses the embedded kata SPACE model.
//
// Deliberately a second file rather than a second column on each time entry:
// what a call costs and what it allocates are two claims, and one line
// asserting both would leave half of every entry unreasoned. The answers do
// diverge — strconv.Atoi allocates nothing on any model, while strings.Split
// is constant here only because K-1 says input is not graded.
func SpaceProfile() (*assume.Set, error) { return parse(spaceProfile) }

func parse(text string) (*assume.Set, error) {
	es, err := assume.ParseText(text)
	if err != nil {
		return nil, err
	}
	return assume.NewSet(es), nil
}
