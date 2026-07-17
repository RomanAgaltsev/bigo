// Package stringops is the canonical-corpus string-algorithms family.
package stringops

// NaiveSearch returns the first index of pat in text, or -1, by checking
// every alignment.
//
//oracle:time O(n*m) where n=len(text), m=len(pat)
//oracle:space O(1) where n=len(text)
//oracle:source CLRS §32.1; en.wikipedia.org/wiki/String-searching_algorithm
func NaiveSearch(text, pat string) int {
	if len(pat) == 0 {
		return 0
	}
	for i := 0; i+len(pat) <= len(text); i++ {
		j := 0
		for j < len(pat) && text[i+j] == pat[j] {
			j++
		}
		if j == len(pat) {
			return i
		}
	}
	return -1
}

// KMPSearch returns the first index of pat in text, or -1, via the
// Knuth–Morris–Pratt failure table.
//
//oracle:time O(n + m) where n=len(text), m=len(pat)
//oracle:space O(m) where m=len(pat)
//oracle:source CLRS §32.4; en.wikipedia.org/wiki/Knuth%E2%80%93Morris%E2%80%93Pratt_algorithm
func KMPSearch(text, pat string) int {
	if len(pat) == 0 {
		return 0
	}
	fail := make([]int, len(pat))
	k := 0
	for i := 1; i < len(pat); i++ {
		for k > 0 && pat[k] != pat[i] {
			k = fail[k-1]
		}
		if pat[k] == pat[i] {
			k++
		}
		fail[i] = k
	}
	k = 0
	for i := 0; i < len(text); i++ {
		for k > 0 && pat[k] != text[i] {
			k = fail[k-1]
		}
		if pat[k] == text[i] {
			k++
		}
		if k == len(pat) {
			return i - len(pat) + 1
		}
	}
	return -1
}

// RabinKarp returns the first index of pat in text, or -1. Worst case
// (all hash collisions) re-verifies every alignment.
//
//oracle:time O(n*m) where n=len(text), m=len(pat)
//oracle:space O(1) where n=len(text)
//oracle:source CLRS §32.2 (worst case); en.wikipedia.org/wiki/Rabin%E2%80%93Karp_algorithm
func RabinKarp(text, pat string) int {
	const base, mod = 256, 1000000007
	if len(pat) == 0 {
		return 0
	}
	if len(pat) > len(text) {
		return -1
	}
	var hp, ht, pow int64 = 0, 0, 1
	for i := 0; i < len(pat); i++ {
		hp = (hp*base + int64(pat[i])) % mod
		ht = (ht*base + int64(text[i])) % mod
		if i > 0 {
			pow = pow * base % mod
		}
	}
	for i := 0; ; i++ {
		if hp == ht {
			j := 0
			for j < len(pat) && text[i+j] == pat[j] {
				j++
			}
			if j == len(pat) {
				return i
			}
		}
		if i+len(pat) >= len(text) {
			return -1
		}
		ht = ((ht-int64(text[i])*pow%mod+mod)%mod*base + int64(text[i+len(pat)])) % mod
	}
}

// IsPalindrome reports whether s reads the same both ways (bytes).
//
//oracle:time O(n) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source ru.algorithmica.org (strings, bound reference); en.wikipedia.org/wiki/Palindrome
func IsPalindrome(s string) bool {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		if s[i] != s[j] {
			return false
		}
	}
	return true
}

// CommonPrefix returns the longest common prefix of a and b. The worst case
// min(|a|,|b|) is attained when |b| ≥ |a|, so O(n) in len(a) is the tight
// expressible worst case.
//
//oracle:time O(n) where n=len(a)
//oracle:space O(1) where n=len(a)
//oracle:source en.wikipedia.org/wiki/LCP_array (pairwise base case, bound reference)
func CommonPrefix(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return a[:i]
}

// RunLengthEncode encodes s as value/count pairs.
//
//oracle:time O(n) where n=len(s)
//oracle:space O(n) where n=len(s)
//oracle:source en.wikipedia.org/wiki/Run-length_encoding
func RunLengthEncode(s string) []byte {
	var out []byte
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && s[j] == s[i] && j-i < 255 {
			j++
		}
		out = append(out, s[i], byte(j-i))
		i = j
	}
	return out
}

// AreAnagrams reports whether a and b are byte-level anagrams via a fixed
// 256-entry count table.
//
//oracle:time O(n + m) where n=len(a), m=len(b)
//oracle:space O(1) where n=len(a)
//oracle:source www.geeksforgeeks.org/check-whether-two-strings-are-anagram-of-each-other/ (bound reference)
func AreAnagrams(a, b string) bool {
	var counts [256]int
	for i := 0; i < len(a); i++ {
		counts[a[i]]++
	}
	for i := 0; i < len(b); i++ {
		counts[b[i]]--
	}
	for _, c := range counts {
		if c != 0 {
			return false
		}
	}
	return true
}

// Reverse returns s reversed (bytes).
//
//oracle:time O(n) where n=len(s)
//oracle:space O(n) where n=len(s)
//oracle:source ru.wikibooks.org/wiki/Реализации_алгоритмов (strings, bound reference)
func Reverse(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		out[len(s)-1-i] = s[i]
	}
	return string(out)
}
