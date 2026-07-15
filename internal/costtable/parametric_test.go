package costtable

import "testing"

func TestParametricTableShape(t *testing.T) {
	for _, key := range []string{"sort.Slice", "sort.SliceStable", "sort.Search",
		"slices.SortFunc", "slices.SortStableFunc", "slices.BinarySearchFunc",
		"slices.ContainsFunc", "slices.IndexFunc", "slices.MaxFunc",
		"slices.MinFunc", "slices.CompactFunc", "slices.EqualFunc"} {
		e, ok := parametric[key]
		if !ok {
			t.Errorf("%s: missing parametric entry", key)
			continue
		}
		if e.Base == nil {
			t.Errorf("%s: nil Base", key)
		}
		if len(e.PerArg) == 0 {
			t.Errorf("%s: no PerArg (a callback function is never invoked?)", key)
		}
	}
}
