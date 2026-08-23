// Package expensivenetwork is the kata corpus's sprint-6 first final: Prim's
// algorithm over an adjacency list, taking the MAXIMUM spanning tree via a
// max-ordered priority queue.
//
// Reduced from the submitted solution; reading the edge list and printing the
// weight are not in this repository.
//
// This is a PRICING row rather than an inference row. The D7 baseline measured
// the whole traversal unverifiable because `container/heap.Push` and `.Pop`
// carry no price, while every one of the five queue methods below is O(1). The
// algorithm is not what defeats bigo here — two missing cost-table entries are.
package expensivenetwork

import (
	"container/heap"
	"errors"
)

var errGraphIsDisconnected = errors.New("graph is disconnected")

// Edge is a weighted undirected edge, stored once per endpoint.
type Edge struct {
	u, v, w int
}

// Graph is an adjacency list plus traversal bookkeeping.
type Graph struct {
	verticesCount   int
	verticesVisited int
	edgesCount      int
	edges           [][]Edge
}

// AddEdge records an edge in both endpoints' adjacency lists.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 6 final 1; two single-element appends, riding the amortization licence
func (g *Graph) AddEdge(u, v, w int) {
	g.edges[u] = append(g.edges[u], Edge{u, v, w})
	g.edges[v] = append(g.edges[v], Edge{v, u, w})
	g.edgesCount++
}

// Max returns the weight of the maximum spanning tree, or an error when the
// graph is disconnected.
//
//oracle:time O(n log n) where n=g.edgesCount
//oracle:space O(n) where n=g.edgesCount
//oracle:source ya_algo sprint 6 final 1; author's claim "Оценка для алгоритма Прима - O(E*log(V))". Pinned against the edge count, the only one of E and V this function names as a size; every edge enters the queue at most once and each queue operation is logarithmic.
func (g *Graph) Max() (int, error) {
	if g.verticesCount == 1 {
		return 0, nil
	}
	if g.verticesCount > 1 && g.edgesCount == 0 {
		return 0, errGraphIsDisconnected
	}
	visited := make([]bool, g.verticesCount+1)
	eq := &EdgeQueue{}
	heap.Push(eq, Edge{-1, 1, 0})
	maxWeight := 0
	for eq.Len() > 0 {
		e := heap.Pop(eq).(Edge)
		if visited[e.v] {
			continue
		}
		visited[e.v] = true
		g.verticesVisited++
		maxWeight += e.w
		for _, edge := range g.edges[e.v] {
			if !visited[edge.v] {
				heap.Push(eq, edge)
			}
		}
	}
	if g.verticesVisited != g.verticesCount {
		return 0, errGraphIsDisconnected
	}
	return maxWeight, nil
}

// EdgeQueue is a container/heap priority queue ordered by descending weight.
type EdgeQueue []Edge

// Len reports the queue length.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 6 final 1; one len, constant by inspection
func (eq EdgeQueue) Len() int { return len(eq) }

// Less orders edges by weight, heaviest first.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 6 final 1; one comparison, constant by inspection
func (eq EdgeQueue) Less(i, j int) bool { return eq[i].w > eq[j].w }

// Swap exchanges two queue positions.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 6 final 1; one exchange, constant by inspection
func (eq EdgeQueue) Swap(i, j int) { eq[i], eq[j] = eq[j], eq[i] }

// Push appends an edge; container/heap does the sifting.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 6 final 1; one single-element append, riding the amortization licence
func (eq *EdgeQueue) Push(x any) {
	*eq = append(*eq, x.(Edge))
}

// Pop removes the last element; container/heap has already moved the max there.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 6 final 1; a reslice and an index, constant by inspection
func (eq *EdgeQueue) Pop() any {
	old := *eq
	n := len(old)
	edge := old[n-1]
	*eq = old[0 : n-1]
	return edge
}
