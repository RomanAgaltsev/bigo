// Package hashtable is the kata corpus's sprint-4 second final: a fixed-capacity
// hash table resolving collisions by chaining, keyed and valued by int.
//
// Reduced from the submitted solution; the command loop and I/O are not in this
// repository.
//
// This was the corpus's clearest PRICING row. The D7 baseline measured `Index`
// unverifiable because `hash/fnv.New32` and `(hash.Hash32).Sum32` carried no
// price, and `Put`, `Get` and `Delete` propagated from it — four functions
// resting on two missing cost-table entries.
//
// Both halves are settled as of 2026-08-29. The pricing shipped in v1.55.0, and
// `Index` is now exact on both axes. The propagation did NOT follow: those four
// were never blocked by `Index` alone, only reported that way, and their real
// blocker is the chain walk in `FindNode` — which has no expressible bound at
// all. See the note above `node`.
package hashtable

import (
	"errors"
	"hash/fnv"
	"strconv"
)

var (
	hashTableCapacity = 1000001
	errValueIsAbsent  = errors.New("value is absent")
)

// THE FIVE CHAIN-WALK FUNCTIONS ARE TIME-UNPINNED ON PURPOSE, since 2026-08-29:
// FindNode, DeleteNode, Put, Get and Delete. Their SPACE pins stay, because
// O(1) is the honest answer there — the walk allocates nothing.
//
// They used to be pinned `O(n) where n=ht.capacity`, and capacity does not bound
// the walk. The table is allocated once at hashTableCapacity and there is no
// resize, no rehash, no load factor and no element cap anywhere in this file,
// while Put prepends without limit. So with M buckets and N colliding
// insertions the chain is length N, and N can exceed capacity without bound.
//
// The author's claim is O(N) over the ELEMENT count; the pin stated O(M) over
// the BUCKET count. Different quantities, and the pin's own source note said so
// — "the element count N is not a size bigo can name here". The corpus already
// forbids exactly this, on packedprefix.(*Stack).String: a pin against
// something else states a claim about a different quantity, so a function
// contributes no row rather than a misleading one.
//
// Nothing emitted O(ht.capacity), so nothing was ever scored wrong. The risk was
// latent and one-directional: any future rule, trust entry or assumption that
// produced it would have been scored EXACT, certifying a prime-directive break
// as ground truth. Unpinning removes the reward.
//
// //oracle:time became optional in the same change, so dropping five unsound
// time pins no longer costs five sound space pins.
type node struct {
	key   int
	value int
	next  *node
}

func newNode(key, value int, next *node) *node {
	return &node{key: key, value: value, next: next}
}

// HashTable is a chaining hash table of fixed capacity.
type HashTable struct {
	capacity int
	table    []*node
}

// Index returns the bucket key hashes into.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 4 final 2; author's claim "Вычисление хеша ключа - O(1)" and "Вычисление номера ячейки таблицы ХТ - O(1)"
func (ht *HashTable) Index(key int) int {
	h := fnv.New32()
	_, _ = h.Write([]byte(strconv.Itoa(key)))
	hash := int(h.Sum32())
	return hash % ht.capacity
}

// FindNode walks a bucket's chain for key, returning the node and its
// predecessor.
//
//oracle:space O(1) where n=ht.capacity
//oracle:source ya_algo sprint 4 final 2; author's worst case "когда вообще все значения попадут в одну ячейку таблицы ХТ, сложность операций будет O(N)" — the chain walk is the N term. The N is the ELEMENT count, which no parameter names, so TIME is unpinned; space O(1) is pinned and sound.
func (ht *HashTable) FindNode(n *node, key int) (*node, *node) {
	var p *node
	for n != nil {
		if n.key == key {
			return n, p
		}
		p = n
		n = n.next
	}
	return nil, nil
}

// DeleteNode unlinks key from a bucket's chain, returning the removed node and
// the chain's new head.
//
//oracle:space O(1) where n=ht.capacity
//oracle:source ya_algo sprint 4 final 2; same chain walk as FindNode, which it calls; time unpinned for the same reason, space O(1) pinned
func (ht *HashTable) DeleteNode(n *node, key int) (*node, *node) {
	if n == nil {
		return nil, nil
	}
	if n.key == key {
		return n, n.next
	}
	curr, prev := ht.FindNode(n, key)
	if prev != nil {
		prev.next = curr.next
	}
	return curr, n
}

// Put stores value under key.
//
//oracle:space O(1) where n=ht.capacity
//oracle:source ya_algo sprint 4 final 2; author's average case "O(1+N/M) или O(1+α)", worst case O(N) when every key lands in one bucket. That N is the element count, which no parameter names, so time is unpinned; space O(1) is pinned and sound.
func (ht *HashTable) Put(key, value int) {
	i := ht.Index(key)
	n := ht.table[i]
	switch {
	case n == nil:
		ht.table[i] = newNode(key, value, nil)
	default:
		if nodeByKey, _ := ht.FindNode(n, key); nodeByKey != nil {
			n.value = value
		} else {
			ht.table[i] = newNode(key, value, n)
		}
	}
}

// Get returns the value stored under key.
//
//oracle:space O(1) where n=ht.capacity
//oracle:source ya_algo sprint 4 final 2; author's worst case O(N) for the chain walk; N is the element count, unnameable, so time is unpinned and space O(1) is pinned
func (ht *HashTable) Get(key int) (int, error) {
	i := ht.Index(key)
	n, _ := ht.FindNode(ht.table[i], key)
	if n != nil {
		return n.value, nil
	}
	return 0, errValueIsAbsent
}

// Delete removes key and returns the value it held.
//
//oracle:space O(1) where n=ht.capacity
//oracle:source ya_algo sprint 4 final 2; author's worst case O(N) for the chain walk; N is the element count, unnameable, so time is unpinned and space O(1) is pinned
func (ht *HashTable) Delete(key int) (int, error) {
	i := ht.Index(key)
	n := ht.table[i]
	deletedNode, headNode := ht.DeleteNode(n, key)
	ht.table[i] = headNode
	if deletedNode != nil {
		return deletedNode.value, nil
	}
	return 0, errValueIsAbsent
}

// NewHashTable returns an empty table of the fixed capacity.
//
//oracle:time O(1)
//oracle:space O(n) where n=hashTableCapacity
//oracle:source ya_algo sprint 4 final 2; the table slice is allocated at capacity M whether or not anything is stored
func NewHashTable() *HashTable {
	return &HashTable{
		capacity: hashTableCapacity,
		table:    make([]*node, hashTableCapacity),
	}
}
