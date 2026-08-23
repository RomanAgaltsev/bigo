// Package packedprefix is the kata corpus's sprint-8 first final: expansion of
// a run-length-style packed string such as "3[ab]" via a byte-slice stack.
//
// Reduced from the submitted solution; the read loop and the common-prefix
// comparison it drives are not in this repository — only the unpacking, which
// is the half the author's claim scores per line.
package packedprefix

import (
	"bytes"
	"strconv"
	"strings"
)

const (
	openBracket  = byte('[')
	closeBracket = byte(']')
)

// Stack holds the fragments assembled so far.
type Stack [][]byte

// Push puts b on top of the stack.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 8 final 1; one append of a single element, riding the amortization licence
func (s *Stack) Push(b []byte) {
	*s = append(*s, b)
}

// Pop removes and returns the top fragment.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 8 final 1; a reslice and an index, constant by inspection
func (s *Stack) Pop() []byte {
	if s.Size() == 0 {
		return nil
	}
	lastIndex := len(*s) - 1
	lastBytes := (*s)[lastIndex]
	*s = (*s)[:lastIndex]
	return lastBytes
}

// Size reports how many fragments the stack holds.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 8 final 1; one len, constant by inspection
func (s *Stack) Size() int {
	return len(*s)
}

// String concatenates every fragment on the stack.
//
// DELIBERATELY UNPINNED. Its cost is one pass over the receiver, and the
// receiver is a *Stack — a pointer to a named slice type. The pin grammar binds
// size variables to parameter-rooted `len()` expressions, and neither `len(*s)`
// nor `len(s)` is one, so the honest bound is inexpressible here. A pin against
// something else would state a claim about a different quantity, so this method
// contributes no corpus row rather than a misleading one. Recorded in
// corpus/kata/README.md beside the equalSums exclusion, which is the same
// limitation in a larger form.
func (s *Stack) String() string {
	var builder strings.Builder
	for i := 0; i < s.Size(); i++ {
		builder.WriteString(string((*s)[i]))
	}
	return builder.String()
}

// Unpack expands a packed line.
//
// Note the author's own correction in the submitted claim: a first estimate of
// O(n * M^2) is struck through and replaced with O(n * M), on the grounds that
// unpacking and the prefix comparison each touch every character once per line.
//
//oracle:time O(len(packedLine)) where n=len(packedLine)
//oracle:space O(len(packedLine)) where n=len(packedLine)
//oracle:source ya_algo sprint 8 final 1; author's claim "Функция распаковки работает за O(M) для каждой строки - обрабатывается каждый символ строки", after his own correction from O(n * M^2) to O(n * M) for the whole solution
func Unpack(packedLine string) string {
	result := Stack{}
	for i := 0; i < len(packedLine); i++ {
		if packedLine[i] == openBracket {
			continue
		}
		if packedLine[i] == closeBracket {
			fragment := result.Pop()
			multiplier, err := strconv.Atoi(string(fragment[0]))
			if err == nil {
				fragment = bytes.Repeat(fragment[1:], multiplier)
			}
			prevFragment := result.Pop()
			prevFragment = append(prevFragment, fragment...)
			result.Push(prevFragment)
			continue
		}
		if _, err := strconv.Atoi(string(packedLine[i])); err == nil {
			result.Push([]byte{packedLine[i]})
			continue
		}
		fragment := result.Pop()
		fragment = append(fragment, packedLine[i])
		result.Push(fragment)
	}
	return result.String()
}
