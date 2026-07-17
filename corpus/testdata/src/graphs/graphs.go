// Package graphs is the canonical-corpus graph family. Graphs are adjacency
// lists adj [][]int with n = len(adj) vertices. Literature bounds of O(V+E)
// are pinned at their expressible worst case O(n^2), since E ≤ n².
package graphs

// BFS returns vertices in breadth-first order from src.
//
//oracle:time O(n^2) where n=len(adj)
//oracle:space O(n) where n=len(adj)
//oracle:source CLRS §22.2 — O(V+E), pinned at the E≤n² worst case
func BFS(adj [][]int, src int) []int {
	visited := make([]bool, len(adj))
	order := make([]int, 0, len(adj))
	queue := []int{src}
	visited[src] = true
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		order = append(order, v)
		for _, w := range adj[v] {
			if !visited[w] {
				visited[w] = true
				queue = append(queue, w)
			}
		}
	}
	return order
}

// DFSIter returns vertices in depth-first order from src, iterative stack.
//
//oracle:time O(n^2) where n=len(adj)
//oracle:space O(n) where n=len(adj)
//oracle:source CLRS §22.3 — O(V+E), pinned at the E≤n² worst case
func DFSIter(adj [][]int, src int) []int {
	visited := make([]bool, len(adj))
	order := make([]int, 0, len(adj))
	stack := []int{src}
	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[v] {
			continue
		}
		visited[v] = true
		order = append(order, v)
		for _, w := range adj[v] {
			if !visited[w] {
				stack = append(stack, w)
			}
		}
	}
	return order
}

// DFSRec fills order depth-first from src, recursively.
//
//oracle:time O(n^2) where n=len(adj)
//oracle:space O(n) where n=len(adj)
//oracle:source CLRS §22.3 (recursive form) — stack depth ≤ n
func DFSRec(adj [][]int, src int, visited []bool, order *[]int) {
	visited[src] = true
	*order = append(*order, src)
	for _, w := range adj[src] {
		if !visited[w] {
			DFSRec(adj, w, visited, order)
		}
	}
}

// FloydWarshall computes all-pairs shortest paths in place on a dense
// distance matrix.
//
//oracle:time O(n^3) where n=len(dist)
//oracle:space O(1) where n=len(dist)
//oracle:source CLRS §25.2; en.wikipedia.org/wiki/Floyd%E2%80%93Warshall_algorithm
func FloydWarshall(dist [][]int) {
	n := len(dist)
	for k := 0; k < n; k++ {
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if d := dist[i][k] + dist[k][j]; d < dist[i][j] {
					dist[i][j] = d
				}
			}
		}
	}
}

// TopoSortKahn returns a topological order of a DAG (Kahn's algorithm), or
// a shorter slice if the graph has a cycle.
//
//oracle:time O(n^2) where n=len(adj)
//oracle:space O(n) where n=len(adj)
//oracle:source CLRS §22.4 / Kahn 1962 — O(V+E), pinned at the E≤n² worst case
func TopoSortKahn(adj [][]int) []int {
	indeg := make([]int, len(adj))
	for _, ws := range adj {
		for _, w := range ws {
			indeg[w]++
		}
	}
	queue := make([]int, 0, len(adj))
	for v, d := range indeg {
		if d == 0 {
			queue = append(queue, v)
		}
	}
	order := make([]int, 0, len(adj))
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		order = append(order, v)
		for _, w := range adj[v] {
			indeg[w]--
			if indeg[w] == 0 {
				queue = append(queue, w)
			}
		}
	}
	return order
}

// Components labels each vertex with its connected-component id (undirected
// adjacency), iterative flood from each unvisited vertex.
//
//oracle:time O(n^2) where n=len(adj)
//oracle:space O(n) where n=len(adj)
//oracle:source CLRS §21 intro — O(V+E) via repeated DFS, pinned at the E≤n² worst case
func Components(adj [][]int) []int {
	comp := make([]int, len(adj))
	for i := range comp {
		comp[i] = -1
	}
	id := 0
	for v := range adj {
		if comp[v] != -1 {
			continue
		}
		stack := []int{v}
		comp[v] = id
		for len(stack) > 0 {
			u := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, w := range adj[u] {
				if comp[w] == -1 {
					comp[w] = id
					stack = append(stack, w)
				}
			}
		}
		id++
	}
	return comp
}
