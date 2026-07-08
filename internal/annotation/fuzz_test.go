package annotation

import "testing"

func FuzzParse(f *testing.F) {
	seeds := []string{
		"//bigo:max O(n log n)",
		"//bigo:max O(n*m) where n=len(a), m=len(b)",
		"//bigo:cost O(log n)",
		"//bigo:ignore",
		"//bigo:space O(n)",
		"// not a directive",
		"//bigo:max O(",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		// Must never panic, regardless of input.
		_, _ = Parse(s)
	})
}
