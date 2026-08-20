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

// Profile parses the embedded kata cost model. A malformed profile is a hard
// error, never a skip: a silently dropped entry would make bigo report a bound
// under a cost model the user did not get.
func Profile() (*assume.Set, error) {
	es, err := assume.ParseText(profile)
	if err != nil {
		return nil, err
	}
	return assume.NewSet(es), nil
}
