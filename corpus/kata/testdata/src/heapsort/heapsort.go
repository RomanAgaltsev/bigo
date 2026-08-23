// Package heapsort is the kata corpus's sprint-5 first final: heapsort over
// contest participants, with the sift-down step available in both recursive and
// iterative form.
//
// Reduced from the submitted solution; reading and printing the participants
// are not in this repository.
//
// The comparator is held as a FUNC VALUE on the heap struct, which is why
// `Largest` is unverifiable for a reason unrelated to the sort: the D7 baseline
// measured it top on a call through a variable, the known funcvalue frontier.
// That is a pricing-and-resolution gap rather than a missing trip-count rule.
package heapsort

import (
	"cmp"
	"strings"
)

// Participant is one contestant's graded record.
type Participant struct {
	Login   string
	Solve   int
	Penalty int
}

// Heap is a max-heap over participants, rooted at index 1.
type Heap struct {
	arr       []Participant
	length    int
	size      int
	iterative bool
	less      func(a, b Participant) bool
}

// Build turns the array into a heap by sifting every internal node down.
//
//oracle:time O(n) where n=h.length
//oracle:space O(1) where n=h.length
//oracle:source ya_algo sprint 5 final 1; author's claim "Построение кучи - O(n) - при построении кучи обрабатываются все элементы исходного массива"
func (h *Heap) Build() {
	for i := h.length / 2; i >= 1; i-- {
		if h.iterative {
			h.HeapifyIterative(i)
		} else {
			h.HeapifyRecursive(i)
		}
	}
}

// HeapifyRecursive sifts index i down until the heap property holds.
//
//oracle:time O(log n) where n=h.length
//oracle:space O(log n) where n=h.length
//oracle:source ya_algo sprint 5 final 1; author's claim "перестановка (просеивание) элемент зависит от высоты кучи, это O(log n)". Space is the recursion stack, one frame per level.
func (h *Heap) HeapifyRecursive(i int) {
	largest := h.Largest(i)
	if largest != i {
		h.Swap(i, largest)
		h.HeapifyRecursive(largest)
	}
}

// HeapifyIterative is HeapifyRecursive without the recursion.
//
//oracle:time O(log n) where n=h.length
//oracle:space O(1) where n=h.length
//oracle:source ya_algo sprint 5 final 1; author's claim "просеивание ... зависит от высоты кучи, это O(log n)", no recursion stack in this form
func (h *Heap) HeapifyIterative(i int) {
	for {
		largest := h.Largest(i)
		if largest == i {
			break
		}
		h.Swap(i, largest)
		i = largest
	}
}

// Sort heapsorts the array in place.
//
//oracle:time O(n log n) where n=h.length
//oracle:space O(log n) where n=h.length
//oracle:source ya_algo sprint 5 final 1; author's claim "получаем O(n + n * log n) или O(n * log n)". Space is the sift recursion; the array itself is the input, not auxiliary.
func (h *Heap) Sort() {
	h.Build()
	for i := h.length; i >= 2; i-- {
		h.Swap(1, i)
		h.size--
		if h.iterative {
			h.HeapifyIterative(1)
		} else {
			h.HeapifyRecursive(1)
		}
	}
}

// Swap exchanges two heap positions.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 5 final 1; one exchange, constant by inspection
func (h *Heap) Swap(i, j int) {
	h.arr[i], h.arr[j] = h.arr[j], h.arr[i]
}

// Largest returns the index of the largest among i and its two children.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 5 final 1; two comparisons through the stored comparator, constant per call under the kata cost model
func (h *Heap) Largest(i int) int {
	largest := i
	l, r := Left(i), Right(i)
	if l <= h.size && h.less(h.arr[i], h.arr[l]) {
		largest = l
	}
	if r <= h.size && h.less(h.arr[largest], h.arr[r]) {
		largest = r
	}
	return largest
}

// NewHeap builds a heap over arr, one-indexed by prepending a zero element.
//
//oracle:time O(n) where n=len(arr)
//oracle:space O(n) where n=len(arr)
//oracle:source ya_algo sprint 5 final 1; the array is copied once into the one-indexed backing slice
func NewHeap(arr []Participant, iterative bool) *Heap {
	return &Heap{
		arr:       append([]Participant{{}}, arr...),
		length:    len(arr),
		size:      len(arr),
		iterative: iterative,
		less: func(a, b Participant) bool {
			if res := cmp.Compare(b.Solve, a.Solve); res != 0 {
				return res == -1
			}
			if res := cmp.Compare(a.Penalty, b.Penalty); res != 0 {
				return res == -1
			}
			return strings.Compare(a.Login, b.Login) == -1
		},
	}
}

// Left returns the left child index of i.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 5 final 1; index arithmetic, constant by inspection
func Left(i int) int {
	return 2 * i
}

// Right returns the right child index of i.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 5 final 1; index arithmetic, constant by inspection
func Right(i int) int {
	return 2*i + 1
}
