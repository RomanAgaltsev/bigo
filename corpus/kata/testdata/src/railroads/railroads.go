// Package railroads is the kata corpus's sprint-6 second final: detecting
// whether a directed road map contains a cycle, by three-colour depth-first
// search.
//
// Reduced from the submitted solution. `init` is omitted rather than reduced —
// it reads the adjacency matrix off a scanner, so what survives the I/O removal
// is `AddRoad` in a loop, which is pinned on its own.
package railroads

const (
	white = iota
	gray
	black
)

const typeB = "B"

// colors holds each vertex's DFS colour. Package-level in the original, kept
// that way here: making it a field would change what the space walk sees.
var colors []int

// Graph is a directed adjacency list.
type Graph struct {
	verticesCount int
	edges         [][]int
}

// AddRoad records a directed road, reversing it for the "B" road type.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 6 final 2; one append of a single element, riding the amortization licence
func (g *Graph) AddRoad(u, v int, t string) {
	if t == typeB {
		u, v = v, u
	}
	if len(g.edges[u]) == 0 {
		g.edges[u] = []int{v}
	} else {
		g.edges[u] = append(g.edges[u], v)
	}
}

// MapIsOptimal reports whether the map is acyclic, running a DFS from every
// still-white vertex.
//
//oracle:time O(n) where n=g.verticesCount
//oracle:space O(n) where n=g.verticesCount
//oracle:source ya_algo sprint 6 final 2; author's claim "фактически используется DFS, то и сложность алгоритма будет такой же - O(V + E)". Pinned against the vertex count, the only one of V and E this function names as a size.
func (g *Graph) MapIsOptimal() bool {
	mapIsOptimal := true
	colors = make([]int, g.verticesCount+1)
	for i := 1; i < g.verticesCount; i++ {
		if colors[i] == white {
			mapIsOptimal = mapIsOptimal && !g.PathHasCycles(i)
		}
	}
	return mapIsOptimal
}

// PathHasCycles reports whether a back edge is reachable from v — a grey vertex
// encountered while it is still on the stack.
//
//oracle:time O(n) where n=g.verticesCount
//oracle:space O(n) where n=g.verticesCount
//oracle:source ya_algo sprint 6 final 2; the DFS visit of the author's O(V + E) claim; space is the recursion stack, at most one frame per vertex
func (g *Graph) PathHasCycles(v int) bool {
	pathHasCycles := false
	colors[v] = gray
	for _, w := range g.edges[v] {
		color := colors[w]
		if color == gray {
			return true
		}
		if color == white {
			pathHasCycles = pathHasCycles || g.PathHasCycles(w)
		}
	}
	colors[v] = black
	return pathHasCycles
}

// NewGraph returns an empty graph.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 6 final 2; struct construction, constant by inspection
func NewGraph() *Graph {
	return &Graph{}
}
