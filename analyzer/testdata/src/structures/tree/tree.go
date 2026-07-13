// Package tree is the canonical-structures corpus for binary search trees.
// Everything here recurses over a POINTER structure (*Node), whose size does
// not decrease as a slice length or integer measure. Pointer-structure
// recursion is an explicit non-goal of the recurrence-solving slice (subtractive
// / Master / Akra–Bazzi over slice-len and integer measures), so these stay ⊤:
// the size-decrease proof the solver requires has no measure to attach to.
package tree

type Node struct {
	Left, Right *Node
	V           int
}

// Insert is O(h) (tree height). Unverifiable: recursion over a *Node pointer
// structure — outside the size-measure recurrence solver. Budget O(1) only to
// force a diagnostic.
//
//bigo:max O(1)
func Insert(root *Node, v int) *Node { // want `cannot verify budget O\(1\)`
	if root == nil {
		return &Node{V: v}
	}
	if v < root.V {
		root.Left = Insert(root.Left, v)
	} else {
		root.Right = Insert(root.Right, v)
	}
	return root
}

// Height is O(n). Unverifiable: recursion over a *Node pointer structure —
// outside the size-measure recurrence solver.
//
//bigo:max O(1)
func Height(root *Node) int { // want `cannot verify budget O\(1\)`
	if root == nil {
		return 0
	}
	l, r := Height(root.Left), Height(root.Right)
	if l > r {
		return l + 1
	}
	return r + 1
}

// InsertAll is O(len(vals) · h): a bounded outer loop over an unverifiable
// pointer-recursive callee. The engine must report the callee as the blocker,
// not claim a bound.
//
//bigo:max O(n)
func InsertAll(root *Node, vals []int) *Node { // want `cannot verify budget O\(len\(vals\)\)`
	for i := 0; i < len(vals); i++ {
		root = Insert(root, vals[i])
	}
	return root
}
