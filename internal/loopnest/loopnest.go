// Package loopnest detects natural loops and their nesting from SSA.
package loopnest

import "golang.org/x/tools/go/ssa"

// Loop is a natural loop: its body blocks and its nesting links.
type Loop struct {
	Header   *ssa.BasicBlock
	Blocks   map[*ssa.BasicBlock]bool
	Parent   *Loop
	Children []*Loop
	Depth    int
}

// Forest is the loop-nesting forest of a function.
type Forest struct {
	Roots   []*Loop
	loopOf  map[*ssa.BasicBlock]*Loop
	byBlock map[*ssa.BasicBlock][]*Loop
}

// EnclosingLoops returns every loop whose body contains b (outermost first is
// not guaranteed. Callers that need order should sort by Depth).
func (f *Forest) EnclosingLoops(b *ssa.BasicBlock) []*Loop {
	return f.byBlock[b]
}

// LoopOf returns the loop with the given header, or nil.
func (f *Forest) LoopOf(header *ssa.BasicBlock) *Loop {
	return f.loopOf[header]
}

// dominates reports whether a dominates b by walking b's immediate-dominator
// chain. A block dominates itself.
func dominates(a, b *ssa.BasicBlock) bool {
	for x := b; x != nil; x = x.Idom() {
		if x == a {
			return true
		}
	}
	return false
}

// UncoveredCycle reports whether fn's CFG contains a cycle that natural-loop
// detection did not cover — an irreducible (multi-entry) cycle. Such a cycle
// has no header and no trip count, so callers must treat the function as
// unverifiable rather than costing its blocks loop-free.
func (f *Forest) UncoveredCycle(fn *ssa.Function) bool {
	// Iterative Tarjan SCC over the block graph.
	index := make(map[*ssa.BasicBlock]int, len(fn.Blocks))
	low := make(map[*ssa.BasicBlock]int, len(fn.Blocks))
	onStack := make(map[*ssa.BasicBlock]bool, len(fn.Blocks))
	var stack []*ssa.BasicBlock
	next := 0
	bad := false

	var connect func(v *ssa.BasicBlock)
	connect = func(v *ssa.BasicBlock) {
		index[v] = next
		low[v] = next
		next++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range v.Succs {
			if _, seen := index[w]; !seen {
				connect(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] != index[v] {
			return
		}
		var scc []*ssa.BasicBlock
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		if !isCycle(scc) {
			return
		}
		for _, b := range scc {
			if len(f.byBlock[b]) == 0 {
				bad = true
			}
		}
	}
	for _, b := range fn.Blocks {
		if _, seen := index[b]; !seen {
			connect(b)
		}
	}
	return bad
}

// isCycle reports whether an SCC actually contains a cycle: more than one
// block, or a single block with an edge to itself.
func isCycle(scc []*ssa.BasicBlock) bool {
	if len(scc) > 1 {
		return true
	}
	for _, s := range scc[0].Succs {
		if s == scc[0] {
			return true
		}
	}
	return false
}

// Build constructs the loop-nesting forest of fn.
func Build(fn *ssa.Function) *Forest {
	headers := map[*ssa.BasicBlock]*Loop{}
	for _, b := range fn.Blocks {
		for _, s := range b.Succs {
			if dominates(s, b) { // b -> s is a back-edge, s is a loop header
				lp := headers[s]
				if lp == nil {
					lp = &Loop{
						Header: s,
						Blocks: map[*ssa.BasicBlock]bool{s: true},
					}
					headers[s] = lp
				}
				addLoopBody(lp, b)
			}
		}
	}
	loops := make([]*Loop, 0, len(headers))
	for _, lp := range headers {
		loops = append(loops, lp)
	}

	// Parent = the smallest other loop whose body contains this loop's header.
	for _, l := range loops {
		var best *Loop
		for _, cand := range loops {
			if cand == l || !cand.Blocks[l.Header] {
				continue
			}
			if best == nil || len(cand.Blocks) < len(best.Blocks) {
				best = cand
			}
		}
		l.Parent = best
	}

	f := &Forest{
		loopOf:  map[*ssa.BasicBlock]*Loop{},
		byBlock: map[*ssa.BasicBlock][]*Loop{},
	}
	for _, l := range loops {
		f.loopOf[l.Header] = l
		if l.Parent == nil {
			f.Roots = append(f.Roots, l)
		} else {
			l.Parent.Children = append(l.Parent.Children, l)
		}
		for b := range l.Blocks {
			f.byBlock[b] = append(f.byBlock[b], l)
		}
	}
	for _, r := range f.Roots {
		assignDepth(r, 0)
	}
	return f
}

// addLoopBody adds tail and all blocks that reach tail without passing through
// the header (the natural loop of the back-edge).
func addLoopBody(lp *Loop, tail *ssa.BasicBlock) {
	if lp.Blocks[tail] {
		return
	}
	lp.Blocks[tail] = true
	stack := []*ssa.BasicBlock{tail}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, p := range n.Preds {
			if p == lp.Header || lp.Blocks[p] {
				continue
			}
			lp.Blocks[p] = true
			stack = append(stack, p)
		}
	}
}

func assignDepth(l *Loop, d int) {
	l.Depth = d
	for _, c := range l.Children {
		assignDepth(c, d+1)
	}
}
