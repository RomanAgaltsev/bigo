// Package levenshtein is the kata corpus's sprint-7 first final: edit distance
// between two strings, computed row by row over two rolling arrays rather than
// a full matrix.
//
// Reduced from the submitted solution; reading the two lines and printing the
// result are not in this repository.
package levenshtein

// LevenshteinDistance returns the edit distance between s and t.
//
// The two strings are swapped so the longer one indexes the rows, and only the
// previous and current rows are kept — which is where the author's O(N) space
// claim comes from, against O(M*N) for the naive matrix.
//
//oracle:time O(len(s) * len(t)) where n=len(s)
//oracle:space O(len(s)) where n=len(s)
//oracle:source ya_algo sprint 7 final 1; author's claim "для решения необходимо выполнить M * N операций. Получается оценка O(M * N)", space "создается два слайса размера N+1 ... получаем оценку по памяти - O(N)"
func LevenshteinDistance(s, t string) int {
	if len(s) < len(t) {
		s, t = t, s
	}
	lenS, lenT := len(s), len(t)
	prev := make([]int, lenS+1)
	curr := make([]int, lenS+1)
	for i := 0; i <= lenS; i++ {
		prev[i] = i
	}
	for i := 1; i < lenT+1; i++ {
		curr[0] = i + 1
		for j := 1; j < lenS+1; j++ {
			deletionCost := prev[j] + 1
			insertionCost := curr[j-1] + 1
			substitutionCost := 0
			if t[i-1] == s[j-1] {
				substitutionCost = prev[j-1]
			} else {
				substitutionCost = prev[j-1] + 1
			}
			curr[j] = min(deletionCost, insertionCost, substitutionCost)
		}
		prev, curr = curr, prev
	}
	return prev[lenS]
}
