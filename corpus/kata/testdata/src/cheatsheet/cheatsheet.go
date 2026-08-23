// Package cheatsheet is the kata corpus's sprint-8 second final: word-break
// over a trie — can a line be split entirely into dictionary words.
//
// Reduced from the submitted solution. `getTrie` is omitted rather than
// reduced: it exists only to read the dictionary off a scanner, so what remains
// of it after the I/O is removed is `insert` in a loop, which is already pinned
// on its own.
package cheatsheet

// Node is one trie node: its byte transitions and whether a word ends here.
type Node struct {
	children   map[byte]*Node
	isTerminal bool
}

// Insert adds word to the trie rooted at n.
//
//oracle:time O(len(word)) where n=len(word)
//oracle:space O(len(word)) where n=len(word)
//oracle:source ya_algo sprint 8 final 2; author's claim "Сложность формирования префиксного дерева составляет O(L)" over the total length of all words — one word contributes its own length
func (n *Node) Insert(word string) {
	node := n
	for i := 0; i < len(word); i++ {
		next, ok := node.children[word[i]]
		if !ok {
			next = NewNode()
			node.children[word[i]] = next
		}
		node = next
	}
	node.isTerminal = true
}

// NewNode returns an empty trie node.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 8 final 2; map construction, constant by inspection
func NewNode() *Node {
	return &Node{
		children:   make(map[byte]*Node),
		isTerminal: false,
	}
}

// IsLineSplitable reports whether line splits entirely into trie words.
//
// The outer loop walks split points; the inner one walks the trie from that
// point, which is bounded by the longest word rather than by the line. The
// author's claim names that distinction — bigo cannot, and reports the
// conservative square.
//
//oracle:time O(len(line)^2) where n=len(line)
//oracle:space O(len(line)) where n=len(line)
//oracle:source ya_algo sprint 8 final 2; author's claim "Проверка возможности разбивки строки составляет O(T * M)", T the line length and M the longest word. M is not a size this function's parameters can name, so the pin states the author's own bound with M <= T substituted — a word longer than the line can never match, so the substitution is sound and does not weaken the claim.
func IsLineSplitable(line string, root *Node) bool {
	lenLine := len(line)
	dp := make([]bool, lenLine+1)
	dp[0] = true
	for i := 0; i <= lenLine; i++ {
		node := root
		if dp[i] {
			for j := i; j <= lenLine; j++ {
				if node.isTerminal {
					dp[j] = true
				}
				if j == lenLine || node.children[line[j]] == nil {
					break
				}
				node = node.children[line[j]]
			}
		}
	}
	return dp[lenLine]
}
