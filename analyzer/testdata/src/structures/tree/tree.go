// Package tree is the canonical-structures corpus for binary search trees.
// Everything here is recursive; recursion is ⊤ in v1 and graduates with
// Phase-2 recurrence solving (Master/Akra–Bazzi per the design spec §12).
package tree

type Node struct {
	Left, Right *Node
	V           int
}

// Insert is O(h) (tree height). Unverifiable today: recursion. Budget O(1)
// only to force a diagnostic. Graduates with: recurrence solving (Phase 2).
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

// Height is O(n). Unverifiable today: recursion. Graduates with: recurrence
// solving (Phase 2).
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
// recursive callee. The engine must report the callee as the blocker, not
// claim a bound. Graduates with the callee.
//
//bigo:max O(n)
func InsertAll(root *Node, vals []int) *Node { // want `cannot verify budget O\(len\(vals\)\)`
	for i := 0; i < len(vals); i++ {
		root = Insert(root, vals[i])
	}
	return root
}
