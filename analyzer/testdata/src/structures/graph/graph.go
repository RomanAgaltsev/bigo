// Package graph is the canonical-structures corpus for adjacency-list graphs.
package graph

// Degrees is O(V): one bounded pass over the adjacency slice. Bounded today
// (range over a slice parameter).
//
//bigo:max O(n)
func Degrees(adj [][]int) []int {
	out := make([]int, len(adj))
	for i := range adj {
		out[i] = len(adj[i])
	}
	return out
}

// BFS is O(V+E). Unverifiable today: the worklist loop's trip count is an
// amortization argument (each vertex enqueued once), not a syntactic size.
// Graduates with: workset/amortized reasoning (Phase 2/3) — or annotate.
//
//bigo:max O(n^2)
func BFS(adj [][]int, src int) []bool { // want `cannot verify budget O\(len\(adj\)\^2\)`
	seen := make([]bool, len(adj))
	queue := []int{src}
	seen[src] = true
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		for i := 0; i < len(adj[v]); i++ {
			w := adj[v][i]
			if !seen[w] {
				seen[w] = true
				queue = append(queue, w)
			}
		}
	}
	return seen
}

// DFS is O(V+E). Unverifiable today: recursion (and the same amortization
// argument). Graduates with: recurrence solving + workset reasoning.
//
//bigo:max O(n^2)
func DFS(adj [][]int, v int, seen []bool) { // want `cannot verify budget O\(len\(adj\)\^2\)`
	seen[v] = true
	for i := 0; i < len(adj[v]); i++ {
		if w := adj[v][i]; !seen[w] {
			DFS(adj, w, seen)
		}
	}
}
