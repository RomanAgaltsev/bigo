// Package removenode is the kata corpus's sprint-5 second final: deletion of a
// key from a binary search tree, replacing a two-child node with its in-order
// successor.
//
// Reduced from the submitted solution; the hand-built test tree and printing
// are not in this repository.
//
// Worth knowing when reading this row: recognizer R-B (list walk) was written
// with `successor` in mind and DOES NOT FIRE on it. R-B's shape is a `p != nil`
// guard, and this loop guards on `node.left != nil` — a loaded field. Widening
// R-B to the field-guard form is a spec decision that was deliberately not
// taken during v1.52.0's implementation, and bigo ships a failing-on-purpose
// test pinning that it does not fire.
package removenode

// Node is one binary-search-tree node.
type Node struct {
	value int
	left  *Node
	right *Node
}

// Successor returns the leftmost node of the subtree rooted at node — the
// in-order successor used when deleting a node with two children.
//
//oracle:time O(n) where n=n
//oracle:space O(1) where n=n
//oracle:source ya_algo sprint 5 final 2; author's claim "найти его преемника - O(n) в худшем, O(h) в среднем случае". The worst case is pinned: h = n when the tree degenerates to a chain, and that is the question bigo answers.
func Successor(node *Node) *Node {
	for node.left != nil {
		node = node.left
	}
	return node
}

// Remove deletes key from the tree rooted at node and returns the new root.
//
//oracle:time O(n) where n=n
//oracle:space O(n) where n=n
//oracle:source ya_algo sprint 5 final 2; author's claim "получаем сложность решения O(h)" for the average case, with O(n) named as the worst case when the tree is one chain. Space is the recursion stack, author's "для хранения стека рекурсивных вызовов требуется O(h)".
func Remove(node *Node, key int) *Node {
	if node == nil {
		return node
	}
	switch {
	case key < node.value:
		node.left = Remove(node.left, key)
	case key > node.value:
		node.right = Remove(node.right, key)
	default:
		if node.left == nil {
			right := node.right
			node = nil
			return right
		}
		if node.right == nil {
			left := node.left
			node = nil
			return left
		}
		succ := Successor(node.right)
		node.value = succ.value
		node.right = Remove(node.right, succ.value)
	}
	return node
}
