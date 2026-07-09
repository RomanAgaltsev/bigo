# big O

A Go static analyzer that infers the **asymptotic time complexity** of Go code and
checks it against developer-declared budgets:

```go
//bigo:max O(n log n)
func Search(xs []int, target int) int { ... }
