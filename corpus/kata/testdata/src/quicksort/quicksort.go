// Package quicksort is the kata corpus's sprint-3 final: an in-place quicksort
// over contest participants, ordered by solved-count desc, penalty asc, login
// asc.
//
// Reduced from the submitted solution. The I/O scaffolding and the task
// statement are not part of the algorithm and are not in this repository; the
// author's own complexity section excludes them from the claim as well.
package quicksort

import (
	"cmp"
	"math/rand"
	"strings"
)

// Participant is one contestant's graded record.
type Participant struct {
	Login   string
	Solve   int
	Penalty int
}

// Less orders participants: solved-count descending, penalty ascending, login
// ascending. It returns -1 when a sorts before b.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 3 final 2; the author's claim states "сравнение структур-участников - это O(1)" — one participant comparison is one element operation under the kata cost model
func Less(a, b Participant) int {
	if res := cmp.Compare(b.Solve, a.Solve); res != 0 {
		return res
	}
	if res := cmp.Compare(a.Penalty, b.Penalty); res != 0 {
		return res
	}
	return strings.Compare(a.Login, b.Login)
}

// Partition splits participants[left:right] around a randomly chosen pivot and
// returns the right pointer's resting index.
//
// The shape is the classic amortized two-pointer partition: an outer loop that
// runs while the pointers have not met, wrapping two inner scans that advance
// them toward each other.
//
// TIME IS DELIBERATELY UNPINNED, since 2026-08-29, and this closes bigo's
// standing inference commitment with a measurement rather than a rule.
//
// THIS FUNCTION DOES NOT TERMINATE ON AN ALL-EQUAL SLICE. `Less` is a three-way
// comparator, so `Less(x, x)` is 0, not -1. On a slice whose elements all
// compare equal, neither inner scan advances, the `left < right` swap exchanges
// an element with an equal one, and the outer loop repeats forever. Classic
// Hoare partition advances both pointers after the swap; this one does not.
//
// PROVED by running it, 2026-08-29: three identical Participants, Partition
// did not return within 5s, while the same test on three distinct Participants
// returned immediately. The reduction is faithful — sprint3/final2's
// `partition` is character-for-character the same loop.
//
// So the honest worst-case time is UNBOUNDED, and `O(n)` was a claim about the
// function's INTENDED INPUT rather than about the function. On a real scoreboard
// logins are unique, `Less` is then a strict total order, and the linear
// amortized argument holds — which is exactly the caveat bigo's own recognition
// channel already prints: "true when the comparison is strict on distinct
// elements".
//
// **bigo's ⊤ here is the RIGHT ANSWER, not a gap.** The prime directive is a
// worst-case upper bound; the worst case is non-termination. No rule should ever
// be extended to bound this, and the pin had to go because it would have scored
// such a rule `exact`. Space O(1) is sound and stays: the loop allocates nothing
// however long it runs.
//
//oracle:space O(1) where n=len(participants)
//oracle:source ya_algo sprint 3 final 2; the author's claim counts partition as the O(n) per-level element work inside quicksort's O(n log n). That claim holds only for pairwise-distinct participants; see the note above.
func Partition(participants []Participant, left, right int) int {
	pivotIndex := rand.Intn(right+1-left) + left
	pivot := participants[pivotIndex]
	for left < right {
		for Less(participants[left], pivot) == -1 {
			left++
		}
		for Less(pivot, participants[right]) == -1 {
			right--
		}
		if left < right {
			participants[left], participants[right] = participants[right], participants[left]
		}
	}
	return right
}

// QuickSort sorts participants[left:right] in place.
//
// The pin is the author's AVERAGE-case claim. bigo reasons about the worst
// case, which for this shape is O(n^2) — the author says so too, and chose to
// present the average case as the answer. A worst-case emission dominates this
// pin and classifies as `loose`, not `wrong`: the two are answering different
// questions, and the corpus README says so rather than letting a future reader
// read `loose` as a graduation target.
//
// OPEN, 2026-08-29: this pin inherits Partition's non-termination on an
// all-equal slice (see the note there), so O(n log n) rests on the same
// unstated distinct-elements precondition. It is LEFT PINNED because it is
// already labelled as answering a different question and bigo emits ⊤ for it
// anyway — but whether an average-case pin may rest on an unstated precondition
// is a corpus-policy question, not a measurement, and it is open.
//
//oracle:time O(n log n) where n=len(participants)
//oracle:space O(log n) where n=len(participants)
//oracle:source ya_algo sprint 3 final 2; author's claim "решение работает за O(n * log n), свойственное среднему случаю быстрой сортировки", auxiliary space "O(log n) дополнительной памяти" for the recursion stack
func QuickSort(participants []Participant, left, right int) {
	if left >= right {
		return
	}
	p := Partition(participants, left, right)
	QuickSort(participants, left, p-1)
	QuickSort(participants, p+1, right)
}
