// Package deque is the kata corpus's sprint-2 final: a fixed-capacity circular
// deque over a preallocated buffer.
//
// Reduced from the submitted solution. The command loop and the I/O scaffolding
// are not part of the data structure and are not in this repository; the
// author's claim scores the deque operations separately from the command
// reading in any case.
package deque

import "errors"

const (
	stepRight = 1
	stepLeft  = -1
)

var (
	errDequeIsEmpty = errors.New("deque is empty")
	errDequeIsFull  = errors.New("deque is full")
)

// Deque is a ring buffer of fixed capacity with head and tail indices.
type Deque struct {
	deque []int
	head  int
	tail  int
	max   int
	size  int
}

// NewDeque allocates a deque of capacity n.
//
// The allocation is the author's O(k) term, where k is the buffer size: the
// buffer is created whether or not any command ever arrives.
//
//oracle:time O(1) where n=n
//oracle:space O(n) where n=n
//oracle:source ya_algo sprint 2 final 1; author's claim "Ииницализация слайса для хранения буфера - О(k), где k - размер буфера", constant work beyond the allocation
func NewDeque(n int) *Deque {
	return &Deque{
		deque: make([]int, n),
		head:  0,
		tail:  0,
		max:   n,
		size:  0,
	}
}

// IsEmpty reports whether the deque holds no elements.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)"
func (d *Deque) IsEmpty() bool {
	return d.size == 0
}

// IsFull reports whether the deque is at capacity.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)"
func (d *Deque) IsFull() bool {
	return d.size == d.max
}

// MoveHead advances the head index by step, wrapping.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 1; index arithmetic, constant by inspection
func (d *Deque) MoveHead(step int) {
	d.head = CalculatePosition(d.head, step, d.max)
}

// MoveTail advances the tail index by step, wrapping.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 1; index arithmetic, constant by inspection
func (d *Deque) MoveTail(step int) {
	d.tail = CalculatePosition(d.tail, step, d.max)
}

// CalculatePosition wraps pointer+step into [0,max).
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 1; one modulo, constant by inspection
func CalculatePosition(pointer, step, max int) int {
	return (pointer + step + max) % max
}

// PushBack appends value at the tail.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)"
func (d *Deque) PushBack(value int) error {
	if d.IsFull() {
		return errDequeIsFull
	}
	d.deque[d.tail] = value
	d.MoveTail(stepRight)
	d.size++
	return nil
}

// PushFront prepends value at the head.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)"
func (d *Deque) PushFront(value int) error {
	if d.IsFull() {
		return errDequeIsFull
	}
	d.MoveHead(stepLeft)
	d.deque[d.head] = value
	d.size++
	return nil
}

// PopFront removes and returns the head element.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)"
func (d *Deque) PopFront() (int, error) {
	if d.IsEmpty() {
		return 0, errDequeIsEmpty
	}
	value := d.deque[d.head]
	d.MoveHead(stepRight)
	d.size--
	return value, nil
}

// PopBack removes and returns the tail element.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)"
func (d *Deque) PopBack() (int, error) {
	if d.IsEmpty() {
		return 0, errDequeIsEmpty
	}
	d.MoveTail(stepLeft)
	value := d.deque[d.tail]
	d.size--
	return value, nil
}
