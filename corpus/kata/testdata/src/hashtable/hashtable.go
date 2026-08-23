// Package hashtable is the kata corpus's sprint-4 second final: a fixed-capacity
// hash table resolving collisions by chaining, keyed and valued by int.
//
// Reduced from the submitted solution; the command loop and I/O are not in this
// repository.
//
// This is the corpus's clearest PRICING row rather than an inference row. The
// D7 baseline measured `index` unverifiable because `hash/fnv.New32` and
// `(hash.Hash32).Sum32` carry no price, and `Put`, `Get` and `Delete` all
// propagate from it — four functions resting on two missing cost-table entries.
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
//oracle:time O(n) where n=ht.capacity
//oracle:space O(1) where n=ht.capacity
//oracle:source ya_algo sprint 4 final 2; author's worst case "когда вообще все значения попадут в одну ячейку таблицы ХТ, сложность операций будет O(N)" — the chain walk is the N term. Pinned against capacity as the only size in scope; the element count N is not a size bigo can name here.
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
//oracle:time O(n) where n=ht.capacity
//oracle:space O(1) where n=ht.capacity
//oracle:source ya_algo sprint 4 final 2; same chain walk as FindNode, which it calls
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
//oracle:time O(n) where n=ht.capacity
//oracle:space O(1) where n=ht.capacity
//oracle:source ya_algo sprint 4 final 2; author's average case "O(1+N/M) или O(1+α)", worst case O(N) when every key lands in one bucket. The worst case is pinned, since that is the question bigo answers.
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
//oracle:time O(n) where n=ht.capacity
//oracle:space O(1) where n=ht.capacity
//oracle:source ya_algo sprint 4 final 2; author's worst case O(N) for the chain walk
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
//oracle:time O(n) where n=ht.capacity
//oracle:space O(1) where n=ht.capacity
//oracle:source ya_algo sprint 4 final 2; author's worst case O(N) for the chain walk
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
