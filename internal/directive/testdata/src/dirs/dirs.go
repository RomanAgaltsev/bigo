package dirs

//bigo:cost O(1)
func opaque(x int) int

//bigo:mx O(n)
func typo(xs []int) int {
	return len(xs)
}

//bigo:max O(n)
//bigo:max O(n^2)
func duplicate(xs []int) int {
	return len(xs)
}

//bigo:cost O(1)
//bigo:ignore
func conflict(x int) int {
	return x
}

type Doer interface {
	//bigo:cost O(1)
	Do(x int) int
}

func plain() {}

type FS struct{ items []int }

//bigo:cost O(k) where k=len(s.items)
func fieldCost(s *FS) int

type Scanner interface {
	//bigo:cost O(k) where k=len(s.items)
	Scan(s *FS) int
}
