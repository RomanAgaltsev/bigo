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
// them toward each other. Linear in total, although no single loop is
// individually bounded — which is why this function is bigo's standing
// inference commitment rather than a pricing question.
//
//oracle:time O(n) where n=len(participants)
//oracle:space O(1) where n=len(participants)
//oracle:source ya_algo sprint 3 final 2; the author's claim counts partition as the O(n) per-level element work inside quicksort's O(n log n)
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
