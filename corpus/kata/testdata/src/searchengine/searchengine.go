// Package searchengine is the kata corpus's sprint-4 first final: an inverted
// word index with a relevance-ranked query, returning the top documents by
// match count.
//
// Reduced from the submitted solution; the input loop and printing are not in
// this repository.
package searchengine

import (
	"cmp"
	"strings"
)

// SearchEngine holds the inverted index and the result-size limit.
type SearchEngine struct {
	index map[string]map[int]int
	limit int
}

// UpdateIndex adds one document's words to the index.
//
//oracle:time O(len(doc)) where n=len(doc)
//oracle:space O(len(doc)) where n=len(doc)
//oracle:source ya_algo sprint 4 final 1; author's claim "Построение поискового индекса - O(N)" over the words of the documents. Pinned against len(doc) because the word count is not a size bigo can name; words ≤ characters, so the pin is sound in the direction that matters.
func (s *SearchEngine) UpdateIndex(docNumber int, doc string) {
	for _, word := range strings.Split(doc, " ") {
		if wordsCount, ok := s.index[word]; ok {
			wordsCount[docNumber+1]++
		} else {
			s.index[word] = map[int]int{docNumber + 1: 1}
		}
	}
}

// ProcessQuery returns the most relevant document numbers for query.
//
// The ranking is a partial bubble sort: `limit` passes over the candidate list,
// which is where the author's K*L term comes from once the unique query words
// have been matched against the documents holding them.
//
//oracle:time O(len(query)^2) where n=len(query)
//oracle:space O(len(query)) where n=len(query)
//oracle:source ya_algo sprint 4 final 1; author's claim "O(N + M + K*L)" for the whole solution, of which this function is the M + K*L part. Pinned against len(query) as the only size bigo can name here: both the unique-word count K and the candidate-document count are bounded by the query length in this fixture's terms, and the squared term covers the limited bubble sort.
func (s *SearchEngine) ProcessQuery(query string) []int {
	wordsCount := make(map[int]int)
	words := GetUniqueWords(query)
	for _, word := range words {
		if docs, ok := s.index[word]; ok {
			for doc, count := range docs {
				wordsCount[doc] += count
			}
		}
	}
	wordsCountSorted := make([][2]int, 0)
	for doc, count := range wordsCount {
		if count != 0 {
			wordsCountSorted = append(wordsCountSorted, [2]int{doc, count})
		}
	}
	limit := s.limit
	lenRelevance := len(wordsCountSorted)
	if lenRelevance < limit {
		limit = lenRelevance
	}
	for i := 0; i < limit; i++ {
		for j := lenRelevance - 1; j > i; j-- {
			if Less(wordsCountSorted[j-1], wordsCountSorted[j]) == -1 {
				wordsCountSorted[j-1], wordsCountSorted[j] = wordsCountSorted[j], wordsCountSorted[j-1]
			}
		}
	}
	relevance := make([]int, 0)
	for i := 0; i < limit; i++ {
		relevance = append(relevance, wordsCountSorted[i][0])
	}
	return relevance
}

// NewSearchEngine returns an empty engine returning at most limit documents.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 4 final 1; map construction, constant by inspection
func NewSearchEngine(limit int) *SearchEngine {
	return &SearchEngine{
		index: make(map[string]map[int]int),
		limit: limit,
	}
}

// GetUniqueWords returns the distinct words of query.
//
//oracle:time O(len(query)) where n=len(query)
//oracle:space O(len(query)) where n=len(query)
//oracle:source ya_algo sprint 4 final 1; author's claim "Обработка входщих запросов, получение уникальных слов - O(M)". Pinned against len(query): words ≤ characters.
func GetUniqueWords(query string) []string {
	wordsCount := make(map[string]int)
	for _, word := range strings.Split(query, " ") {
		wordsCount[word]++
	}
	result := make([]string, 0)
	for word := range wordsCount {
		result = append(result, word)
	}
	return result
}

// Less orders (document, count) pairs by count descending, document ascending.
//
//oracle:time O(1)
//oracle:space O(1)
//oracle:source ya_algo sprint 4 final 1; comparator, constant work per call under the kata cost model
func Less(a, b [2]int) int {
	if res := cmp.Compare(a[1], b[1]); res != 0 {
		return res
	}
	if res := cmp.Compare(b[0], a[0]); res != 0 {
		return res
	}
	return 0
}
