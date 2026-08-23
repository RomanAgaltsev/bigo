// Package rpncalc is the kata corpus's sprint-2 second final: an evaluator for
// expressions in reverse Polish notation, over an integer stack.
//
// Reduced from the submitted solution; the line reading the expression is
// excluded from the author's own claim ("чтение строки с ввода исключаем") and
// is not in this repository.
package rpncalc

import (
	"math"
	"strconv"
	"strings"
)

var operations = map[string]func(int, int) int{
	"+": func(a, b int) int { return a + b },
	"-": func(a, b int) int { return a - b },
	"*": func(a, b int) int { return a * b },
	"/": func(a, b int) int { return int(math.Floor(float64(a) / float64(b))) },
}

// Stack holds the operands seen so far.
type Stack struct {
	items []int
}

// Push puts item on top of the stack.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 2; author's claim "На каждый токен-операнд выполняется помещение в стек - O(1)". Amortized append rides bigo's documented primitive licence.
func (s *Stack) Push(item int) {
	s.items = append(s.items, item)
}

// Pop removes and returns the top of the stack.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 2; author's claim "извлечение двух операндов из стека - O(1)"
func (s *Stack) Pop() int {
	lastIndex := len(s.items) - 1
	lastItem := s.items[lastIndex]
	s.items = s.items[:lastIndex]
	return lastItem
}

// NewStack returns an empty operand stack.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 2; allocation of an empty slice, constant by inspection
func NewStack() *Stack {
	return &Stack{items: []int{}}
}

// Calculator evaluates one RPN expression.
type Calculator struct {
	exp   string
	stack *Stack
}

// Calculate evaluates the expression and returns its value.
//
// One pass over the tokens, constant work per token: an operand is pushed, an
// operator pops two and pushes one.
//
//oracle:time O(n) where n=len(c.exp)
//oracle:space O(n) where n=len(c.exp)
//oracle:source ya_algo sprint 2 final 2; author's claim "Общее время - O(n+n) - O(2n) - O(n)", where n is the token count. Pinned against len(c.exp) because the token count is not a size bigo can name; tokens ≤ characters, so the pin is sound in the direction that matters.
func (c *Calculator) Calculate() int {
	tokens := strings.Split(c.exp, " ")
	for _, token := range tokens {
		if operation, ok := operations[token]; ok {
			b, a := c.stack.Pop(), c.stack.Pop()
			c.stack.Push(operation(a, b))
		} else {
			intToken, _ := strconv.Atoi(token)
			c.stack.Push(intToken)
		}
	}
	return c.stack.Pop()
}

// NewCalculator returns a calculator for expression.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 2 final 2; struct construction, constant by inspection
func NewCalculator(expression string) *Calculator {
	return &Calculator{
		exp:   expression,
		stack: NewStack(),
	}
}
