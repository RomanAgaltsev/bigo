package recognize

import (
	"strings"
	"testing"
)

const linkedSrc = `package input
type node struct {
	key  int
	next *node
}
func f(n *node, key int) *node {
	for n != nil {
		if n.key == key {
			return n
		}
		n = n.next
	}
	return nil
}`

func TestLinkedTraversalRecognized(t *testing.T) {
	rs := detectIn(t, linkedSrc, "f")
	if len(rs) != 1 {
		t.Fatalf("want exactly 1 recognition, got %d: %+v", len(rs), rs)
	}
	r := rs[0]
	if r.Pattern != "linked-structure traversal" {
		t.Errorf("Pattern = %q", r.Pattern)
	}
	if r.Kind != KindWorst {
		t.Errorf("Kind = %q, want %q", r.Kind, KindWorst)
	}
	// The bound names a quantity the bound algebra CANNOT express. That is
	// only sound because a recognition never participates in a verdict.
	if r.Bound != "O(n)" {
		t.Errorf("Bound = %q, want O(n)", r.Bound)
	}
	if !strings.Contains(r.Assumption, "reachable") {
		t.Errorf("Assumption must say what n counts, got %q", r.Assumption)
	}
	// The link is what the reader has to check, so it must be named.
	if !strings.Contains(r.Assumption, ".next") {
		t.Errorf("Assumption must name the link it follows, got %q", r.Assumption)
	}
	if !r.Pos.IsValid() {
		t.Error("Pos must point at the loop that justifies the claim")
	}
}

// The real targets named in the spec, reduced to their shapes.
//
// hashTable.findNode is the specced shape and fires. removeNode.successor is
// NOT the specced shape: it guards on `node.left != nil`, a loaded field, where
// R-B requires the pointer itself. The spec counted it as a target anyway, so
// this test pins the truth rather than the claim -- widening R-B to cover the
// field-guard form is a SPEC decision, not something to slip in here.
const findNodeSrc = `package input
type node struct {
	key  int
	next *node
}
func f(n *node, key int) (*node, *node) {
	var p *node
	for n != nil {
		if n.key == key {
			return n, p
		}
		p = n
		n = n.next
	}
	return nil, nil
}`

const successorSrc = `package input
type Node struct {
	key   int
	left  *Node
	right *Node
}
func f(node *Node) *Node {
	for node.left != nil {
		node = node.left
	}
	return node
}`

func TestLinkedTraversalOnTheRealTargets(t *testing.T) {
	t.Run("findNode fires", func(t *testing.T) {
		rs := detectIn(t, findNodeSrc, "f")
		if len(rs) != 1 {
			t.Fatalf("want exactly 1 recognition, got %d: %+v", len(rs), rs)
		}
		if rs[0].Pattern != "linked-structure traversal" {
			t.Errorf("Pattern = %q", rs[0].Pattern)
		}
		if !strings.Contains(rs[0].Assumption, ".next") {
			t.Errorf("Assumption must name the link, got %q", rs[0].Assumption)
		}
	})
	// Pinned as a KNOWN non-match, not as a defect. R-B's shape is
	// `for p != nil`; this is `for p.field != nil`. Refusing an inexact shape
	// is the channel's rule, so the pin records the gap where the spec's
	// target count can be corrected against it.
	t.Run("successor does not fire: the guard is a field, not the pointer", func(t *testing.T) {
		if rs := detectIn(t, successorSrc, "f"); len(rs) != 0 {
			t.Errorf("expected no recognition, got %+v", rs)
		}
	})
}

func TestLinkedTraversalRefusesNearMisses(t *testing.T) {
	cases := map[string]string{
		// The load-bearing negative: visiting each node at most once would
		// need .alt to be acyclic too, and nothing proves that.
		"pointer also assigned from elsewhere": `package input
type node struct{ key int; next, alt *node }
func f(n *node, key int) *node {
	for n != nil {
		if n.key == key {
			n = n.alt
		}
		n = n.next
	}
	return nil
}`,
		"guard is not a nil check": `package input
type node struct{ key int; next *node }
func f(n *node, k int) *node {
	for k > 0 {
		n = n.next
		k--
	}
	return n
}`,
		// A conditional advance: on the else path the pointer does not move at
		// all, so "visits each node at most once" says nothing about how many
		// times the loop runs. The claim would be about the wrong quantity.
		"advance is conditional": `package input
type node struct{ key int; next *node }
func f(n *node, key int) *node {
	for n != nil {
		if n.key == key {
			n = n.next
		}
	}
	return nil
}`,
	}
	for name, src := range cases {
		if rs := detectIn(t, src, "f"); len(rs) != 0 {
			t.Errorf("%s: expected no recognition, got %+v", name, rs)
		}
	}
}
