// Package hashmap is the canonical-structures corpus for Go maps. Map
// index/assign/delete are O(1) in the documented v0.x cost model.
package hashmap

// Has is O(1). Bounded today.
//
//bigo:max O(1)
func Has(m map[string]int, k string) bool {
	_, ok := m[k]
	return ok
}

// LookupAll is O(len(keys)): a counted loop of O(1) map lookups. Bounded today.
// The budget binds n to keys explicitly: a bare O(n) would default to the first
// size parameter (the map m), and len(m) is not comparable to len(keys).
//
//bigo:max O(n) where n=len(keys)
func LookupAll(m map[string]int, keys []string) int {
	s := 0
	for i := 0; i < len(keys); i++ {
		s += m[keys[i]]
	}
	return s
}

// SumValues is O(len(m)). Bounded since the loop-algebra slice: range-Next
// over a map parameter with a clean body.
//
//bigo:max O(n)
func SumValues(m map[string]int) int {
	s := 0
	for _, v := range m {
		s += v
	}
	return s
}
