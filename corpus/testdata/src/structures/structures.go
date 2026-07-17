// Package structures is the canonical-corpus data-structure-operations family.
// Array-backed only: pointer-structure sizes are inexpressible in canonical
// size variables and live in the exclusion register instead (spec §3.3.1).
package structures

// SiftDown restores the max-heap property from index i in h.
//
//oracle:time O(log n) where n=len(h)
//oracle:space O(1) where n=len(h)
//oracle:source CLRS §6.2 (MAX-HEAPIFY, iterative)
func SiftDown(h []int, i int) {
	for {
		child := 2*i + 1
		if child >= len(h) {
			return
		}
		if child+1 < len(h) && h[child+1] > h[child] {
			child++
		}
		if h[i] >= h[child] {
			return
		}
		h[i], h[child] = h[child], h[i]
		i = child
	}
}

// BuildHeap heapifies h bottom-up — the classic linear-time bound, a prime
// oracle entry: n log n from inference would be loose, n exact.
//
//oracle:time O(n) where n=len(h)
//oracle:space O(1) where n=len(h)
//oracle:source CLRS §6.3 (BUILD-MAX-HEAP is Θ(n))
func BuildHeap(h []int) {
	for i := len(h)/2 - 1; i >= 0; i-- {
		SiftDown(h, i)
	}
}

// HeapPush appends v and sifts it up; returns the grown heap.
//
// Space pin corrected from O(n) to O(1) (oracle run, PR 2). CLRS §6.5's
// MAX-HEAP-INSERT writes into a pre-allocated array and uses O(1) auxiliary
// space; the O(n) originally pinned here was Go's `append` reallocating, which
// is a language implementation detail the citation does not claim. A pin must
// state its citation's bound (spec §3.3–3.4), and bigo models append as
// amortized O(1) on both axes by documented design (README, "What bigo does not
// count (yet)"), so O(1) is also what the engine's own contract says. Pinning the
// single-op realloc worst case would test Go's slice growth, not the algorithm.
//
//oracle:time O(log n) where n=len(h)
//oracle:space O(1) where n=len(h)
//oracle:source CLRS §6.5 (MAX-HEAP-INSERT — O(1) auxiliary space)
func HeapPush(h []int, v int) []int {
	h = append(h, v)
	i := len(h) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if h[parent] >= h[i] {
			break
		}
		h[parent], h[i] = h[i], h[parent]
		i = parent
	}
	return h
}

// HeapPop removes and returns the maximum; returns the shrunk heap.
//
//oracle:time O(log n) where n=len(h)
//oracle:space O(1) where n=len(h)
//oracle:source CLRS §6.5 (HEAP-EXTRACT-MAX)
func HeapPop(h []int) (int, []int) {
	top := h[0]
	h[0] = h[len(h)-1]
	h = h[:len(h)-1]
	SiftDown(h, 0)
	return top, h
}

// LinearProbeInsert inserts key into an open-addressing table (-1 = empty),
// in place. Worst case probes the whole table.
//
//oracle:time O(n) where n=len(table)
//oracle:space O(1) where n=len(table)
//oracle:source CLRS §11.4 (linear probing, worst case)
func LinearProbeInsert(table []int, key int) bool {
	if len(table) == 0 {
		return false
	}
	start := key % len(table)
	if start < 0 {
		start += len(table)
	}
	for i := 0; i < len(table); i++ {
		j := (start + i) % len(table)
		if table[j] == -1 || table[j] == key {
			table[j] = key
			return true
		}
	}
	return false
}

// StackOps pushes n values and pops them all through a slice stack: 2n
// slice operations total.
//
//oracle:time O(n) where n=n
//oracle:space O(n) where n=n
//oracle:source sequential stack ops; en.wikipedia.org/wiki/Stack_(abstract_data_type)
func StackOps(n int) int {
	stack := make([]int, 0, n)
	for i := 0; i < n; i++ {
		stack = append(stack, i)
	}
	sum := 0
	for len(stack) > 0 {
		sum += stack[len(stack)-1]
		stack = stack[:len(stack)-1]
	}
	return sum
}

// DSUFind returns x's root in an array-backed disjoint-set forest WITHOUT
// path compression — the plain worst case is a chain of length n. (With
// compression the bound is amortized and inexpressible; see EXCLUSIONS.md.)
//
//oracle:time O(n) where n=len(parent)
//oracle:space O(1) where n=len(parent)
//oracle:source CLRS §21.2 (linked-list forests, un-amortized worst case)
func DSUFind(parent []int, x int) int {
	for parent[x] != x {
		x = parent[x]
	}
	return x
}

// DSUUnion merges the sets of a and b by root relinking (no rank).
//
//oracle:time O(n) where n=len(parent)
//oracle:space O(1) where n=len(parent)
//oracle:source CLRS §21.2 (un-amortized worst case)
func DSUUnion(parent []int, a, b int) {
	ra, rb := DSUFind(parent, a), DSUFind(parent, b)
	if ra != rb {
		parent[ra] = rb
	}
}
