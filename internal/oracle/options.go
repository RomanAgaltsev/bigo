package oracle

import "github.com/RomanAgaltsev/bigo/internal/assume"

// Kind names which corpus a Report describes. It selects the golden's preamble
// and nothing else: the scoring rules are identical, and the two corpora differ
// in what their pins MEAN rather than in how they are classified.
type Kind string

const (
	// CanonicalCorpus pins literature bounds under the default cost model.
	CanonicalCorpus Kind = ""
	// KataCorpus pins human claims on submitted solutions under the kata model.
	KataCorpus Kind = "kata"
)

// Options carries the cost model a corpus is scored under.
//
// The canonical corpus passes the zero value: its pins are literature bounds,
// which are claims about an algorithm and not about any particular cost model.
// The kata corpus passes the kata profile, because a kata's graded claim is
// made IN that model — "one element comparison is one element operation" is
// what makes the author's O(n log n) the right answer for a comparison sort.
// Scoring a kata claim under the default model would answer a question nobody
// asked.
type Options struct {
	// Overlay prices calls on the TIME axis. Nil means the default model.
	Overlay *assume.Set
	// Corpus names which golden this run renders, so a reader cannot mistake a
	// human's average-case claim for a literature bound.
	Corpus Kind
	// SpaceOverlay prices calls on the SPACE axis.
	//
	// Separate from Overlay, and attached together or not at all: what a call
	// costs and what it allocates are two assertions, and a corpus scored with
	// kata time and default space would carry two cost models and say so
	// nowhere. That shape shipped once already (v1.52.1) and is why this is two
	// fields rather than one flag.
	SpaceOverlay *assume.Set
}
