// Package list is the canonical-structures corpus for linked lists. Pointer
// chasing has no size-variable trip count; the honest resolution is
// annotate-or-trust (//bigo:cost on the traversal), not inference. The same
// boundary applies to pointer recursion: the recurrence solver measures slice
// length and integer magnitude, neither of which a *Node chain exposes.
package list

type Node struct {
	Next *Node
	V    int
}

type List struct {
	Head *Node
}

// PushFront is O(1). Bounded today: no loops.
//
//bigo:max O(1)
func PushFront(l *List, v int) {
	l.Head = &Node{Next: l.Head, V: v}
}

// Len is O(n) in the list length. Unverifiable today (and by design in v1):
// a pointer-chasing loop has no parameter-size trip count. The budget below
// is O(1) only to force a diagnostic. Resolution: annotate-or-trust.
//
//bigo:max O(1)
func Len(l *List) int { // want `cannot verify budget O\(1\)`
	n := 0
	for p := l.Head; p != nil; p = p.Next {
		n++
	}
	return n
}

// Contains is O(n) in the list length. Same story as Len.
//
//bigo:max O(1)
func Contains(l *List, v int) bool { // want `cannot verify budget O\(1\)`
	for p := l.Head; p != nil; p = p.Next {
		if p.V == v {
			return true
		}
	}
	return false
}
